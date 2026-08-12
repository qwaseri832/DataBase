package wal

import (
	"bytes"
	"encoding/gob"
)

type Op int

const (
	OpUnknown Op = iota
	OpSet
	OpDel
)

func (o Op) ArgCount() int {
	switch o {
	case OpSet:
		return 2
	case OpDel:
		return 1
	default:
		return 0
	}
}

func (o Op) String() string {
	switch o {
	case OpSet:
		return "SET"
	case OpDel:
		return "DEL"
	default:
		return "UNKNOWN"
	}
}

type Record struct {
	LSN  int64
	Op   Op
	Args []string
}

func (r *Record) Encode(buf *bytes.Buffer) error {
	return gob.NewEncoder(buf).Encode(r)
}

func (r *Record) Decode(buf *bytes.Buffer) error {
	return gob.NewDecoder(buf).Decode(r)
}
