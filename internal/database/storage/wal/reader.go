package wal

import (
	"bytes"
	"fmt"
	"sort"
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
		buf := bytes.NewBuffer(data)
		for buf.Len() > 0 {
			var rec Record
			if err := rec.Decode(buf); err != nil {
				return fmt.Errorf("decode record: %w", err)
			}
			recs = append(recs, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(recs, func(i, j int) bool {
		return recs[i].LSN < recs[j].LSN
	})
	return recs, nil
}
