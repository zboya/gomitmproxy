package mitm

import (
	"io"
	"net"
)

type mitmListener struct {
	conn net.Conn
}

func (listener *mitmListener) Accept() (net.Conn, error) {
	if listener.conn != nil {
		conn := listener.conn
		listener.conn = nil
		return conn, nil
	} else {
		return nil, io.EOF
	}
}

func (listener *mitmListener) Close() error {
	return nil
}

func (listener *mitmListener) Addr() net.Addr {
	return nil
}
