package network

import "time"

const defaultBufSize = 4 << 10 // 4KB

// --- Client options ---

type ClientOpt func(*TCPClient)

func WithClientTimeout(d time.Duration) ClientOpt {
	return func(c *TCPClient) { c.timeout = d }
}

func WithClientBuffer(n int) ClientOpt {
	return func(c *TCPClient) { c.bufSize = n }
}

// --- Server options ---

type ServerOpt func(*TCPServer)

func WithServerTimeout(d time.Duration) ServerOpt {
	return func(s *TCPServer) { s.idleTimeout = d }
}

func WithServerBuffer(n int) ServerOpt {
	return func(s *TCPServer) { s.bufSize = n }
}

func WithServerMaxClients(n int) ServerOpt {
	return func(s *TCPServer) { s.maxClients = n }
}
