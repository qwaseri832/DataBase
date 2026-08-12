package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

const SegmentPrefix = "wal_"

var segmentSeq atomic.Uint32

type Segment struct {
	dir     string
	maxSize int

	file *os.File
	size int
}

func NewSegment(dir string, maxSize int) *Segment {
	if maxSize <= 0 {
		maxSize = 10 << 20
	}
	return &Segment{dir: dir, maxSize: maxSize}
}

func (s *Segment) Write(data []byte) error {
	if s.file == nil || s.size+len(data) > s.maxSize {
		if err := s.rotate(); err != nil {
			return fmt.Errorf("rotate segment: %w", err)
		}
	}

	n, err := s.file.Write(data)
	s.size += n
	if err != nil {
		return err
	}
	return s.file.Sync()
}

func (s *Segment) Close() error {
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	s.file = nil
	return err
}

func (s *Segment) rotate() error {
	if err := s.Close(); err != nil {
		return err
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", s.dir, err)
	}

	name := filepath.Join(s.dir, nextSegmentName())

	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	s.file = f
	s.size = 0
	return nil
}

func nextSegmentName() string {
	return fmt.Sprintf("%s%013d_%06d.log",
		SegmentPrefix, time.Now().UnixMilli(), segmentSeq.Add(1))
}
