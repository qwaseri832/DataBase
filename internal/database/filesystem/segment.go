package filesystem

import (
	"fmt"
	"os"
	"time"
)

// Segment пишет данные в файл-сегмент с ротацией по размеру.
type Segment struct {
	dir     string
	maxSize int

	file *os.File
	size int
}

func NewSegment(dir string, maxSize int) *Segment {
	return &Segment{dir: dir, maxSize: maxSize}
}

func (s *Segment) Write(data []byte) error {
	if s.file == nil || s.size >= s.maxSize {
		if err := s.rotate(); err != nil {
			return fmt.Errorf("rotate: %w", err)
		}
	}

	n, err := s.file.Write(data)
	if err != nil {
		return err
	}
	if err := s.file.Sync(); err != nil {
		return err
	}
	s.size += n
	return nil
}

func (s *Segment) rotate() error {
	name := fmt.Sprintf("%s/wal_%d.log", s.dir, time.Now().UnixMilli())
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	s.file = f
	s.size = 0
	return nil
}
