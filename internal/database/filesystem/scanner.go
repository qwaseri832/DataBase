package filesystem

import (
	"os"
	"path/filepath"
)

type DirScanner struct {
	dir string
}

func NewDirScanner(dir string) *DirScanner {
	return &DirScanner{dir: dir}
}

func (d *DirScanner) ForEach(fn func(data []byte) error) error {
	names, err := segmentNames(d.dir)
	if err != nil {
		return err
	}

	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(d.dir, name))
		if err != nil {
			return err
		}
		if err := fn(data); err != nil {
			return err
		}
	}
	return nil
}
