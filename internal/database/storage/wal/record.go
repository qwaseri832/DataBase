package wal

import (
	"bytes"
	"encoding/gob"
)

// Record — одна запись WAL.
type Record struct {
	LSN  int64
	Cmd  int
	Args []string
}

func (r *Record) Encode(buf *bytes.Buffer) error {
	return gob.NewEncoder(buf).Encode(r)
}

func (r *Record) Decode(buf *bytes.Buffer) error {
	return gob.NewDecoder(buf).Decode(r)
}
