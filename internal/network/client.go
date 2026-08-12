package network

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

type TCPClient struct {
	conn    net.Conn
	r       *bufio.Reader
	w       *bufio.Writer
	timeout time.Duration
	bufSize int

	mu sync.Mutex
}

func DialTCP(addr string, opts ...ClientOpt) (*TCPClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	c := &TCPClient{conn: conn, bufSize: defaultBufSize}
	for _, o := range opts {
		o(c)
	}

	c.r = bufio.NewReaderSize(conn, c.bufSize)
	c.w = bufio.NewWriterSize(conn, c.bufSize)
	return c, nil
}

func (c *TCPClient) Send(req []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.timeout > 0 {
		if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
			return nil, err
		}
	}

	if err := WriteFrame(c.w, req); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	if err := c.w.Flush(); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	resp, err := ReadFrame(c.r, DefaultMaxFrameSize)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return resp, nil
}

func (c *TCPClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
