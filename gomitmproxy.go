// This example shows a proxy server that uses go-mitm to man-in-the-middle
// HTTPS connections opened with CONNECT requests

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	Version = "1.1"
)

var (
	wg sync.WaitGroup
)

var logFile *os.File
var logger *log.Logger

func main() {
	var conf Cfg

	conf.Port = flag.String("port", "8080", "Listen port")
	conf.Raddr = flag.String("raddr", "", "Remote addr")
	conf.Log = flag.String("log", "./error.log", "log file path")
	conf.Monitor = flag.Bool("m", false, "monitor mode")
	conf.Tls = flag.Bool("tls", false, "tls connect")
	help := flag.Bool("h", false, "help")
	flag.Parse()

	if *help {
		flag.PrintDefaults()
	}
	var err error
	logFile, err = os.Create(*conf.Log)
	if err != nil {
		log.Fatalln("fail to create log file!")
	}

	logger = log.New(logFile, "[gomitmproxy]", log.LstdFlags|log.Llongfile)

	wg.Add(1)
	gomitmproxy(&conf)
	wg.Wait()
}

func gomitmproxy(conf *Cfg) {
	tlsConfig := NewTlsConfig("gomitmproxy-ca-pk.pem", "gomitmproxy-ca-cert.pem", "", "")

	handler, err := InitConfig(conf, tlsConfig)
	if err != nil {
		logger.Fatalf("InitConfig error: %s", err)
	}

	server := &http.Server{
		Addr:         ":" + *conf.Port,
		Handler:      handler,
		ReadTimeout:  1 * time.Hour,
		WriteTimeout: 1 * time.Hour,
	}

	go func() {
		log.Printf("proxy listening port:%s", *conf.Port)

		if *conf.Tls {
			log.Println("ListenAndServeTLS")
			err = server.ListenAndServeTLS("gomitmproxy-ca-cert.pem", "gomitmproxy-ca-pk.pem")
		} else {
			log.Println("ListenAndServe")
			err = server.ListenAndServe()
		}
		if err != nil {
			logger.Fatalf("Unable to start HTTP proxy: %s", err)
		}

		wg.Done()

		log.Printf("gomitmproxy stop!!!!")
	}()

	return
}
