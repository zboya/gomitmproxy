package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"sync"
	"time"
)

type HandlerWrapper struct {
	MyConfig        *Cfg
	tlsConfig       *TlsConfig
	wrapped         http.Handler
	pk              *PrivateKey
	pkPem           []byte
	issuingCert     *Certificate
	issuingCertPem  []byte
	serverTLSConfig *tls.Config
	dynamicCerts    *Cache
	certMutex       sync.Mutex
	https           bool
}

func (hw *HandlerWrapper) GenerateCertForClient() (err error) {
	if hw.tlsConfig.Organization == "" {
		hw.tlsConfig.Organization = "gomitmproxy" + Version
	}
	if hw.tlsConfig.CommonName == "" {
		hw.tlsConfig.CommonName = "gomitmproxy"
	}
	if hw.pk, err = LoadPKFromFile(hw.tlsConfig.PrivateKeyFile); err != nil {
		hw.pk, err = GeneratePK(2048)
		if err != nil {
			return fmt.Errorf("Unable to generate private key: %s", err)
		}
		hw.pk.WriteToFile(hw.tlsConfig.PrivateKeyFile)
	}
	hw.pkPem = hw.pk.PEMEncoded()
	hw.issuingCert, err = LoadCertificateFromFile(hw.tlsConfig.CertFile)
	if err != nil || hw.issuingCert.ExpiresBefore(time.Now().AddDate(0, ONE_MONTH, 0)) {
		hw.issuingCert, err = hw.pk.TLSCertificateFor(
			hw.tlsConfig.Organization,
			hw.tlsConfig.CommonName,
			time.Now().AddDate(ONE_YEAR, 0, 0),
			true,
			nil)
		if err != nil {
			return fmt.Errorf("Unable to generate self-signed issuing certificate: %s", err)
		}
		hw.issuingCert.WriteToFile(hw.tlsConfig.CertFile)
	}
	hw.issuingCertPem = hw.issuingCert.PEMEncoded()
	return
}

func (hw *HandlerWrapper) FakeCertForName(name string) (cert *tls.Certificate, err error) {
	kpCandidateIf, found := hw.dynamicCerts.Get(name)
	if found {
		return kpCandidateIf.(*tls.Certificate), nil
	}

	hw.certMutex.Lock()
	defer hw.certMutex.Unlock()
	kpCandidateIf, found = hw.dynamicCerts.Get(name)
	if found {
		return kpCandidateIf.(*tls.Certificate), nil
	}

	//create certificate
	certTTL := TWO_WEEKS
	generatedCert, err := hw.pk.TLSCertificateFor(
		hw.tlsConfig.Organization,
		name,
		time.Now().Add(certTTL),
		false,
		hw.issuingCert)
	if err != nil {
		return nil, fmt.Errorf("Unable to issue certificate: %s", err)
	}
	keyPair, err := tls.X509KeyPair(generatedCert.PEMEncoded(), hw.pkPem)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse keypair for tls: %s", err)
	}

	cacheTTL := certTTL - ONE_DAY
	hw.dynamicCerts.Set(name, &keyPair, cacheTTL)
	return &keyPair, nil
}

func (hw *HandlerWrapper) DumpHTTPAndHTTPs(resp http.ResponseWriter, req *http.Request) {
	req.Header.Del("Proxy-Connection")
	req.Header.Set("Connection", "Keep-Alive")

	// handle connection
	connIn, _, err := resp.(http.Hijacker).Hijack()
	if err != nil {
		logger.Println("hijack error:", err)
	}
	defer connIn.Close()

	var respOut *http.Response
	host := req.Host

	matched, _ := regexp.MatchString(":[0-9]+$", host)

	if !hw.https {
		if !matched {
			host += ":80"
		}

		connOut, err := net.DialTimeout("tcp", host, time.Second*30)
		if err != nil {
			logger.Println("dial to", host, "error:", err)
		}
		respOut, err = http.ReadResponse(bufio.NewReader(connOut), req)
		if err != nil && err != io.EOF {
			logger.Println("read response error:", err)
		}

	} else {
		if !matched {
			host += ":443"
		}

		connOut, err := tls.Dial("tcp", host, hw.tlsConfig.ServerTLSConfig)

		if err != nil {
			logger.Panicln("tls dial to", host, "error:", err)
		}
		if err = req.Write(connOut); err != nil {
			logger.Println("send to server error", err)
		}

		respOut, err = http.ReadResponse(bufio.NewReader(connOut), req)
		if err != nil && err != io.EOF {
			logger.Println("read response error:", err)
		}

	}

	if respOut == nil {
		log.Println("respOut is nil")
		return
	}

	respDump, err := httputil.DumpResponse(respOut, true)
	if err != nil {
		logger.Println("respDump error:", err)
	}

	_, err = connIn.Write(respDump)
	if err != nil {
		logger.Println("connIn write error:", err)
	}

	if *hw.MyConfig.Monitor {
		go httpDump(req, respOut)
	}

}

func (hw *HandlerWrapper) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method == "CONNECT" {
		hw.https = true
		hw.InterceptHTTPs(resp, req)
	} else {
		hw.DumpHTTPAndHTTPs(resp, req)
	}
}

func (hw *HandlerWrapper) InterceptHTTPs(resp http.ResponseWriter, req *http.Request) {
	addr := req.Host
	host := strings.Split(addr, ":")[0]

	cert, err := hw.FakeCertForName(host)
	if err != nil {
		msg := fmt.Sprintf("Could not get mitm cert for name: %s\nerror: %s", host, err)
		respBadGateway(resp, msg)
		return
	}

	// handle connection
	connIn, _, err := resp.(http.Hijacker).Hijack()
	if err != nil {
		msg := fmt.Sprintf("Unable to access underlying connection from client: %s", err)
		respBadGateway(resp, msg)
		return
	}
	tlsConfig := copyTlsConfig(hw.tlsConfig.ServerTLSConfig)
	tlsConfig.Certificates = []tls.Certificate{*cert}
	tlsConnIn := tls.Server(connIn, tlsConfig)

	listener := &mitmListener{tlsConnIn}

	handler := http.HandlerFunc(func(resp2 http.ResponseWriter, req2 *http.Request) {
		req2.URL.Scheme = "https"
		req2.URL.Host = req2.Host
		hw.DumpHTTPAndHTTPs(resp2, req2)

	})

	go func() {
		err = http.Serve(listener, handler)
		if err != nil && err != io.EOF {
			log.Printf("Error serving mitm'ed connection: %s", err)
		}
	}()

	connIn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
}

func Mitm(conf *Cfg, tlsConfig *TlsConfig) (*HandlerWrapper, error) {
	hw := &HandlerWrapper{
		MyConfig:     conf,
		tlsConfig:    tlsConfig,
		dynamicCerts: NewCache(),
	}
	err := hw.GenerateCertForClient()
	if err != nil {
		return nil, err
	}
	return hw, nil
}

func copyTlsConfig(template *tls.Config) *tls.Config {
	tlsConfig := &tls.Config{}
	if template != nil {
		*tlsConfig = *template
	}
	return tlsConfig
}

func respBadGateway(resp http.ResponseWriter, msg string) {
	log.Println(msg)
	resp.WriteHeader(502)
	resp.Write([]byte(msg))
}
