package mylog

import (
	"log"
	"os"
	"testing"
)

func TestMyLog(t *testing.T) {
	logFile, err := os.Create("test.log")
	if err != nil {
		log.Fatalln("fail to create log file!")
	}
	logger := log.New(logFile, "[gomitmproxy]", log.LstdFlags|log.Llongfile)
	SetLog(logger)
	Println("log test")
}
