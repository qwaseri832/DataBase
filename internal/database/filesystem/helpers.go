package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func segmentNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), SegmentPrefix) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func NextSegment(dir, after string) (string, error) {
	names, err := segmentNames(dir)
	if err != nil {
		return "", err
	}

	idx := upperBound(names, after)
	if idx < len(names) {
		return names[idx], nil
	}
	return "", nil
}

func LastSegment(dir string) (string, error) {
	names, err := segmentNames(dir)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", nil
	}
	return names[len(names)-1], nil
}

func WriteSegment(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func upperBound(a []string, target string) int {
	lo, hi := 0, len(a)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if a[mid] <= target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}
