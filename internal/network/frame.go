package network

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const frameHeaderSize = 4

const DefaultMaxFrameSize = 64 << 20

var ErrFrameTooLarge = errors.New("message exceeds size limit")

func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > DefaultMaxFrameSize {
		return fmt.Errorf("%w: %d bytes", ErrFrameTooLarge, len(payload))
	}

	var header [frameHeaderSize]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))

	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	return nil
}

func ReadFrame(r io.Reader, max int) ([]byte, error) {
	if max <= 0 || max > DefaultMaxFrameSize {
		max = DefaultMaxFrameSize
	}

	var header [frameHeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	size := int(binary.BigEndian.Uint32(header[:]))
	if size > max {
		return nil, fmt.Errorf("%w: declared %d bytes, limit %d", ErrFrameTooLarge, size, max)
	}
	if size == 0 {
		return nil, nil
	}

	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}
