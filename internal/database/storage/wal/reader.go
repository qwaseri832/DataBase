package wal

import (
	"cmp"
	"slices"
)

type Scanner interface {
	ForEach(fn func(data []byte) error) error
}

type Reader struct {
	scanner Scanner
}

func NewReader(s Scanner) *Reader {
	return &Reader{scanner: s}
}

func (r *Reader) ReadAll() ([]Record, error) {
	var recs []Record

	err := r.scanner.ForEach(func(data []byte) error {
		segment, err := DecodeSegment(data)
		if err != nil {
			return err
		}
		recs = append(recs, segment...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(recs, func(a, b Record) int { return cmp.Compare(a.LSN, b.LSN) })
	return recs, nil
}
