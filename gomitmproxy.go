/*
Author:shepbao
Time:2016-06-01
*/

package main

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"log"
)

const (
	MaxOutstanding = 10
)

var Version string = "1.0"
var sem = make(chan int, MaxOutstanding)

type AccessStatistics struct {
	AllCount  int64
	HostCount map[string]int64
}

var transport = &http.Transport{
	ResponseHeaderTimeout: 30 * time.Second,
}

//config json
type Config struct {
	Port  string `json :"port"`           //"8080"
	Raddr string `json:"raddr,omitempty"` //"localhost:8888"
}

var port = flag.String("port", "8080", "Listen port")
var raddr = flag.String("raddr", "", "Remote addr")
var monitor = flag.Bool("m", false, "monitor mode")

var cfg = flag.String("conf", "./config", "config file")
var cf *Config

var logFile *os.File
var logger *log.Logger

func main() {
	var err error
	logFile, err = os.Create("error.log")
	if err != nil {
		log.Fatalln("fail to create log file!")
	}

	logger = log.New(logFile, "[gomitmproxy]", log.LstdFlags|log.Llongfile)
	flag.Parse()

	cf, err = ParseConfig(*cfg)
	if err != nil {
		logger.Println("ParseConfig err:", err)
	} else {
		if len(cf.Port) == 0 {
			log.Fatal("Miss Port")
		}
		*port = cf.Port

		if len(cf.Raddr) == 0 {
			log.Fatal("Miss Raddr")
		}
		*raddr = cf.Raddr
	}

	log.Println("Listen port: ", *port)
	if len(*raddr) != 0 {
		log.Println("Connect to Raddr: ", *raddr)
	}

	ln, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		logger.Println("listen err:", err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.Println("Accept err:", err)
			continue
		}
		go handleConn(conn)

	}
}

//处理连接
func handleConn(conn net.Conn) {
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		logger.Println("readRequest err:", err)
		conn.Close()
		return
	}

	host := req.Host
	matched, _ := regexp.MatchString(":[0-9]+$", host)
	if !matched {
		host += ":80"
	}
	conn_proxy, err := Connect(host)
	if err != nil {
		return
	}
	defer conn_proxy.Close()

	if req.Method == "CONNECT" {
		b := []byte("HTTP/1.1 200 Connection Established\r\n" +
			"Proxy-Agent: golang_proxy/" + Version + "\r\n\r\n")
		_, err := conn.Write(b)
		if err != nil {
			logger.Println("Write Connect err:", err)
			return
		}
	} else {
		req.Header.Del("Proxy-Connection")
		req.Header.Set("Connection", "Keep-Alive")
		if err = req.Write(conn_proxy); err != nil {
			logger.Println("send to server err", err)
			return
		}
	}

	if *monitor && req.Method != "CONNECT" {

		resp, err := http.ReadResponse(bufio.NewReader(conn_proxy), req)
		if err != nil {
			logger.Println("read response err:", err)
			return
		}

		respDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			logger.Println("respDump err:", err)
		}

		_, err = conn.Write(respDump)
		if err != nil {
			logger.Println("conn write err:", err)
		}
		// reqDump, _ := httputil.DumpRequest(req, false)
		// log.Println(string(reqDump), string(respDump))
		go httpDump(req, resp)

	} else {
		err = Transport(conn, conn_proxy)
		if err != nil {
			logger.Println("Transport err:", err)
		}
	}
}

//连接真实的主机
func Connect(host string) (net.Conn, error) {
	if len(*raddr) == 0 {
		conn, err := net.DialTimeout("tcp", host, time.Second*30)
		if err != nil {
			// logger.Println("connect", host, "err", err)
			return nil, err
		}
		return conn, nil
	}

	conn_proxy, err := net.DialTimeout("tcp", *raddr, time.Second*30)
	if err != nil {
		logger.Println("connect proxy err", err)
		return nil, err
	}
	err = connectProxyServer(conn_proxy, host)
	if err != nil {
		logger.Println("connectServer err:", err)
		return nil, err
	}
	return conn_proxy, nil
}

//连接代理服务器
func connectProxyServer(conn net.Conn, addr string) error {

	req := &http.Request{
		Method:     "CONNECT",
		URL:        &url.URL{Host: addr},
		Host:       addr,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	req.Header.Set("Proxy-Connection", "keep-alive")

	if err := req.Write(conn); err != nil {
		return err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return nil
}

//两个io口的连接
func Transport(conn1, conn2 net.Conn) (err error) {
	rChan := make(chan error, 1)
	wChan := make(chan error, 1)

	go MyCopy(conn1, conn2, wChan)
	go MyCopy(conn2, conn1, rChan)

	select {
	case err = <-wChan:
	case err = <-rChan:
	}

	return
}

func MyCopy(src io.Reader, dst io.Writer, ch chan<- error) {
	_, err := io.Copy(dst, src)
	ch <- err
}

//配置文件解析
func ParseConfig(cfg string) (*Config, error) {
	var conf Config
	configContext, err := ioutil.ReadFile(cfg)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(configContext, &conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

//复制http头
func copyHeader(src, dst http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

//打印http请求和响应
func httpDump(req *http.Request, resp *http.Response) {
	defer resp.Body.Close()
	var respStatusStr string
	respStatus := resp.StatusCode
	respStatusHeader := int(math.Floor(float64(respStatus / 100)))
	switch respStatusHeader {
	case 2:
		respStatusStr = Green("<--" + strconv.Itoa(respStatus))
	case 3:
		respStatusStr = Yellow("<--" + strconv.Itoa(respStatus))
	case 4:
		respStatusStr = Magenta("<--" + strconv.Itoa(respStatus))
	case 5:
		respStatusStr = Red("<--" + strconv.Itoa(respStatus))
	}
	fmt.Println(Green("Request:"))
	fmt.Printf("%s %s %s\n", Blue(req.Method), req.RequestURI, respStatusStr)
	for headerName, headerContext := range req.Header {
		fmt.Printf("%s: %s\n", Blue(headerName), headerContext)
	}
	fmt.Println(Green("Response:"))
	for headerName, headerContext := range resp.Header {
		fmt.Printf("%s: %s\n", Blue(headerName), headerContext)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Println("read resp body err:", err)
	} else {
		acceptEncode := resp.Header["Content-Encoding"]
		var respBodyBin bytes.Buffer
		w := bufio.NewWriter(&respBodyBin)
		w.Write(respBody)
		w.Flush()
		for _, compress := range acceptEncode {
			switch compress {
			case "gzip":
				r, err := gzip.NewReader(&respBodyBin)
				if err != nil {
					logger.Println("gzip reader err:", err)
				} else {
					defer r.Close()
					respBody, _ = ioutil.ReadAll(r)
				}
				break
			case "deflate":
				r := flate.NewReader(&respBodyBin)
				defer r.Close()
				respBody, _ = ioutil.ReadAll(r)
				break
			}
		}
		fmt.Printf("%s\n", string(respBody))
	}

	fmt.Printf("%s%s%s\n", Black("####################"), Cyan("END"), Black("####################"))
}
