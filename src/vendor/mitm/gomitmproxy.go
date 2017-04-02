// This example shows a proxy server that uses go-mitm to man-in-the-middle
// HTTPS connections opened with CONNECT requests

package mitm

import (
	"config"
	"mylog"
	"net/http"
	"sync"
	"time"
)

func Gomitmproxy(conf *config.Cfg, wg *sync.WaitGroup) {
	tlsConfig := config.NewTlsConfig("gomitmproxy-ca-pk.pem", "gomitmproxy-ca-cert.pem", "", "")

	handler, err := InitConfig(conf, tlsConfig)
	if err != nil {
		mylog.Fatalf("InitConfig error: %s", err)
	}

	server := &http.Server{
		Addr:         ":" + *conf.Port,
		Handler:      handler,
		ReadTimeout:  1 * time.Hour,
		WriteTimeout: 1 * time.Hour,
	}

	go func() {
		mylog.Printf("Gomitmproxy Listening On: %s", *conf.Port)
		if *conf.Tls {
			mylog.Println("Listen And Serve HTTP TLS")
			err = server.ListenAndServeTLS("gomitmproxy-ca-cert.pem", "gomitmproxy-ca-pk.pem")
		} else {
			mylog.Println("Listen And Serve HTTP")
			err = server.ListenAndServe()
		}
		if err != nil {
			mylog.Fatalf("Unable To Start HTTP proxy: %s", err)
		}

		wg.Done()

		mylog.Printf("Gomitmproxy Stop!!!!")
	}()

	return
}
