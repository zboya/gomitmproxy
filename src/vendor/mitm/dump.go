package mitm

import (
	"bufio"
	"bytes"
	"color"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"mylog"
	"net/http"
	"strconv"
)

func httpDump(reqDump []byte, resp *http.Response) {
	defer resp.Body.Close()
	var respStatusStr string
	respStatus := resp.StatusCode
	respStatusHeader := int(math.Floor(float64(respStatus / 100)))
	switch respStatusHeader {
	case 2:
		respStatusStr = color.Green("<--" + strconv.Itoa(respStatus))
	case 3:
		respStatusStr = color.Yellow("<--" + strconv.Itoa(respStatus))
	case 4:
		respStatusStr = color.Magenta("<--" + strconv.Itoa(respStatus))
	case 5:
		respStatusStr = color.Red("<--" + strconv.Itoa(respStatus))
	}

	fmt.Println(color.Green("Request:"), respStatusStr)
	req, _ := ParseReq(reqDump)
	fmt.Printf("%s %s %s\n", color.Blue(req.Method), req.Host+req.RequestURI, respStatusStr)
	fmt.Printf("%s %s\n", color.Blue("RemoteAddr:"), req.RemoteAddr)
	for headerName, headerContext := range req.Header {
		fmt.Printf("%s: %s\n", color.Blue(headerName), headerContext)
	}

	if req.Method == "POST" {
		fmt.Println(color.Green("POST Param:"))
		err := req.ParseForm()
		if err != nil {
			mylog.Println("parseForm error:", err)
		} else {
			for k, v := range req.Form {
				fmt.Printf("\t%s: %s\n", color.Blue(k), v)
			}
		}
	}
	fmt.Println(color.Green("Response:"))
	for headerName, headerContext := range resp.Header {
		fmt.Printf("%s: %s\n", color.Blue(headerName), headerContext)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		mylog.Println("func httpDump read resp body err:", err)
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
					mylog.Println("gzip reader err:", err)
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

	fmt.Printf("%s%s%s\n", color.Black("####################"), color.Cyan("END"), color.Black("####################"))
}

func ParseReq(b []byte) (*http.Request, error) {
	// func ReadRequest(b *bufio.Reader) (req *Request, err error) { return readRequest(b, deleteHostHeader) }
	fmt.Println(string(b))
	fmt.Println("-----------------------")
	var buf io.ReadWriter
	buf = new(bytes.Buffer)
	buf.Write(b)
	bufr := bufio.NewReader(buf)
	return http.ReadRequest(bufr)
}
