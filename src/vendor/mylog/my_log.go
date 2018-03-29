package mylog

import "log"
import "io"

var logger *log.Logger

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func SetLog(l io.WriteCloser) {
	logger = log.New(l, "[gomitmproxy]", log.LstdFlags)
}

func Fatalf(format string, v ...interface{}) {
	logger.Fatalf(format, v)
}

func Fatalln(v ...interface{}) {
	logger.Fatalln(v)
}

func Printf(format string, v ...interface{}) {
	logger.Printf(format, v)
}

func Println(v ...interface{}) {
	logger.Println(v)
}

func Panicln(v ...interface{}) {
	logger.Panicln(v)
}
