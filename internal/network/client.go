package network

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// TCPClient — клиент для отправки запросов серверу.
type TCPClient struct {
	conn    net.Conn
	timeout time.Duration
	bufSize int
}

func DialTCP(addr string, opts ...ClientOpt) (*TCPClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	c := &TCPClient{conn: conn, bufSize: defaultBufSize}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

func (c *TCPClient) Send(req []byte) ([]byte, error) {
	if c.timeout > 0 {
		_ = c.conn.SetDeadline(time.Now().Add(c.timeout))
	}

	if _, err := c.conn.Write(req); err != nil {
		return nil, err
	}

	buf := make([]byte, c.bufSize)
	n, err := c.conn.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return buf[:n], nil
}

func (c *TCPClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
