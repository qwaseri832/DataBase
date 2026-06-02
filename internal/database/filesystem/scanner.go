package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// DirScanner перебирает файлы сегментов в директории.
type DirScanner struct {
	dir string
}

func NewDirScanner(dir string) *DirScanner {
	return &DirScanner{dir: dir}
}

func (d *DirScanner) ForEach(fn func(data []byte) error) error {
	entries, err := os.ReadDir(d.dir)
	if err != nil {
		return fmt.Errorf("scan dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

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
