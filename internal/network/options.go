package network

import "time"

const defaultBufSize = 4 << 10

type ClientOpt func(*TCPClient)

func WithClientTimeout(d time.Duration) ClientOpt {
	return func(c *TCPClient) { c.timeout = d }
}

func WithClientBuffer(n int) ClientOpt {
	return func(c *TCPClient) {
		if n > 0 {
			c.bufSize = n
		}
	}
}

type ServerOpt func(*TCPServer)

func WithServerTimeout(d time.Duration) ServerOpt {
	return func(s *TCPServer) { s.idleTimeout = d }
}

func WithServerBuffer(n int) ServerOpt {
	return func(s *TCPServer) {
		if n > 0 {
			s.bufSize = n
		}
	}
}

func WithServerMaxMessage(n int) ServerOpt {
	return func(s *TCPServer) {
		if n > 0 {
			s.maxMessage = n
		}
	}
}

func WithServerMaxClients(n int) ServerOpt {
	return func(s *TCPServer) { s.maxClients = n }
}
