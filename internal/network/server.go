package network

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"spider/internal/syncx"
)

// Handler обрабатывает сырой запрос и возвращает сырой ответ.
type Handler = func(ctx context.Context, request []byte) []byte

// TCPServer принимает TCP-соединения и обрабатывает запросы.
type TCPServer struct {
	listener    net.Listener
	sem         *syncx.Semaphore
	logger      *zap.Logger
	idleTimeout time.Duration
	bufSize     int
	maxClients  int
}

func ListenTCP(addr string, logger *zap.Logger, opts ...ServerOpt) (*TCPServer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &TCPServer{listener: ln, logger: logger, bufSize: defaultBufSize}
	for _, o := range opts {
		o(s)
	}
	if s.maxClients > 0 {
		s.sem = syncx.NewSemaphore(s.maxClients)
	}
	return s, nil
}

// Serve блокируется до отмены ctx.
func (s *TCPServer) Serve(ctx context.Context, handler Handler) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				s.logger.Error("accept", zap.Error(err))
				continue
			}
			if s.sem != nil {
				s.sem.Acquire()
			}
			go func(c net.Conn) {
				if s.sem != nil {
					defer s.sem.Release()
				}
				s.handleConn(ctx, c, handler)
			}(conn)
		}
	}()

	<-ctx.Done()
	_ = s.listener.Close()
	wg.Wait()
}

func (s *TCPServer) handleConn(ctx context.Context, conn net.Conn, handler Handler) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic in handler", zap.Any("recover", r))
		}
		_ = conn.Close()
	}()

	buf := make([]byte, s.bufSize)
	for {
		if s.idleTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(s.idleTimeout))
		}

		n, err := conn.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.logger.Debug("read", zap.Error(err))
			}
			return
		}

		if s.idleTimeout > 0 {
			_ = conn.SetWriteDeadline(time.Now().Add(s.idleTimeout))
		}

		resp := handler(ctx, buf[:n])
		if _, err := conn.Write(resp); err != nil {
			s.logger.Debug("write", zap.Error(err))
			return
		}
	}
}
