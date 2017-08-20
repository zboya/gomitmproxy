package main

import (
	"config"
	"flag"
	"io"
	"mitm"
	"mylog"
	"os"
	"sync"
)

func main() {
	var log io.WriteCloser
	var err error
	// cofig
	conf := new(config.Cfg)
	conf.Port = flag.String("port", "8080", "Listen port")
	conf.Raddr = flag.String("raddr", "", "Remote addr")
	conf.Log = flag.String("logFile", "", "log file path")
	conf.Monitor = flag.Bool("m", false, "monitor mode")
	conf.Tls = flag.Bool("tls", false, "tls connect")

	flag.Parse()

	// init log
	if *conf.Log != "" {
		log, err = os.Create(*conf.Log)
		if err != nil {
			mylog.Fatalln("fail to create log file " + err.Error())
		}
	} else {
		log = os.Stderr
	}
	mylog.SetLog(log)

	// init tls config
	tlsConfig := config.NewTlsConfig("gomitmproxy-ca-pk.pem", "gomitmproxy-ca-cert.pem", "", "")
	// start mitm proxy
	wg := new(sync.WaitGroup)
	wg.Add(1)
	mitm.Gomitmproxy(conf, tlsConfig, wg)
	wg.Wait()
}
