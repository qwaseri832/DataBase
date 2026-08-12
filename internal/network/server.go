package network

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/syncx"
)

type Handler = func(ctx context.Context, request []byte) []byte

type TCPServer struct {
	listener    net.Listener
	sem         *syncx.Semaphore
	logger      *zap.Logger
	idleTimeout time.Duration
	bufSize     int
	maxMessage  int
	maxClients  int
}

func ListenTCP(addr string, logger *zap.Logger, opts ...ServerOpt) (*TCPServer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &TCPServer{
		listener:   ln,
		logger:     logger,
		bufSize:    defaultBufSize,
		maxMessage: DefaultMaxFrameSize,
	}
	for _, o := range opts {
		o(s)
	}
	if s.maxClients > 0 {
		s.sem = syncx.NewSemaphore(s.maxClients)
	}
	return s, nil
}

func (s *TCPServer) Addr() net.Addr { return s.listener.Addr() }

func (s *TCPServer) Serve(ctx context.Context, handler Handler) {
	var conns sync.WaitGroup

	stopped := make(chan struct{})
	defer close(stopped)

	go func() {
		select {
		case <-ctx.Done():
			_ = s.listener.Close()
		case <-stopped:
		}
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			s.logger.Error("accept", zap.Error(err))
			continue
		}

		if s.sem != nil && !s.sem.AcquireContext(ctx) {
			_ = conn.Close()
			continue
		}

		conns.Add(1)
		go func(c net.Conn) {
			defer conns.Done()
			if s.sem != nil {
				defer s.sem.Release()
			}
			s.handleConn(ctx, c, handler)
		}(conn)
	}

	conns.Wait()
}

func (s *TCPServer) handleConn(ctx context.Context, conn net.Conn, handler Handler) {
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-connCtx.Done()
		_ = conn.Close()
	}()

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic in handler", zap.Any("recover", r))
		}
		_ = conn.Close()
	}()

	r := bufio.NewReaderSize(conn, s.bufSize)
	w := bufio.NewWriterSize(conn, s.bufSize)

	for {
		if s.idleTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(s.idleTimeout))
		}

		req, err := ReadFrame(r, s.maxMessage)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				s.logger.Debug("read request", zap.Error(err))
			}
			return
		}

		resp := handler(connCtx, req)

		if s.idleTimeout > 0 {
			_ = conn.SetWriteDeadline(time.Now().Add(s.idleTimeout))
		}
		if err := WriteFrame(w, resp); err != nil {
			s.logger.Debug("write response", zap.Error(err))
			return
		}
		if err := w.Flush(); err != nil {
			s.logger.Debug("flush response", zap.Error(err))
			return
		}
	}
}
