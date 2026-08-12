package network_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/network"
)

func TestFrameRoundTrip(t *testing.T) {
	payloads := [][]byte{
		nil,
		[]byte(""),
		[]byte("SET key value"),
		bytes.Repeat([]byte{0x00, 0xFF, '\n'}, 1000),
	}

	for _, want := range payloads {
		var buf bytes.Buffer
		if err := network.WriteFrame(&buf, want); err != nil {
			t.Fatalf("WriteFrame: %v", err)
		}

		got, err := network.ReadFrame(&buf, network.DefaultMaxFrameSize)
		if err != nil {
			t.Fatalf("ReadFrame: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("получено %d байт, ожидалось %d", len(got), len(want))
		}
	}
}

func TestFrameSplitsBackToBackMessages(t *testing.T) {
	var buf bytes.Buffer
	for _, s := range []string{"GET a", "GET b"} {
		if err := network.WriteFrame(&buf, []byte(s)); err != nil {
			t.Fatalf("WriteFrame: %v", err)
		}
	}

	for _, want := range []string{"GET a", "GET b"} {
		got, err := network.ReadFrame(&buf, network.DefaultMaxFrameSize)
		if err != nil {
			t.Fatalf("ReadFrame: %v", err)
		}
		if string(got) != want {
			t.Errorf("получено %q, ожидалось %q", got, want)
		}
	}
}

func TestReadFrameRejectsOversizedMessage(t *testing.T) {
	var buf bytes.Buffer
	if err := network.WriteFrame(&buf, bytes.Repeat([]byte("x"), 1024)); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}

	_, err := network.ReadFrame(&buf, 16)
	if !errors.Is(err, network.ErrFrameTooLarge) {
		t.Errorf("ReadFrame() error = %v, ожидалось ErrFrameTooLarge", err)
	}
}

func TestServerReturnsResponseLargerThanBuffer(t *testing.T) {
	const respSize = 1 << 20
	big := strings.Repeat("s", respSize)

	srv, err := network.ListenTCP("127.0.0.1:0", zap.NewNop())
	if err != nil {
		t.Fatalf("ListenTCP: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	served := make(chan struct{})
	go func() {
		defer close(served)
		srv.Serve(ctx, func(_ context.Context, req []byte) []byte {
			if string(req) == "big" {
				return []byte(big)
			}
			return append([]byte("echo:"), req...)
		})
	}()

	client, err := network.DialTCP(srv.Addr().String(), network.WithClientTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("DialTCP: %v", err)
	}
	defer client.Close()

	resp, err := client.Send([]byte("big"))
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(resp) != respSize {
		t.Fatalf("получено %d байт, ожидалось %d", len(resp), respSize)
	}

	resp, err = client.Send([]byte("ping"))
	if err != nil {
		t.Fatalf("второй Send: %v", err)
	}
	if string(resp) != "echo:ping" {
		t.Errorf("получено %q, ожидалось \"echo:ping\"", resp)
	}

	cancel()
	select {
	case <-served:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve не завершился по отмене контекста")
	}
}

func TestServerAcceptsRequestLargerThanBuffer(t *testing.T) {
	srv, err := network.ListenTCP("127.0.0.1:0", zap.NewNop())
	if err != nil {
		t.Fatalf("ListenTCP: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.Serve(ctx, func(_ context.Context, req []byte) []byte {
		return []byte(strings.Repeat("=", len(req)))
	})

	client, err := network.DialTCP(srv.Addr().String(), network.WithClientTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("DialTCP: %v", err)
	}
	defer client.Close()

	const reqSize = 256 << 10
	resp, err := client.Send(bytes.Repeat([]byte("q"), reqSize))
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(resp) != reqSize {
		t.Errorf("сервер увидел %d байт запроса, ожидалось %d", len(resp), reqSize)
	}
}
