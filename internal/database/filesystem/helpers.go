package filesystem

import (
	"fmt"
	"os"
	"sort"
)

// NextSegment возвращает имя следующего сегмента после given (или "" если нет).
func NextSegment(dir, after string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("scan dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	idx := upperBound(names, after)
	if idx < len(names) {
		return names[idx], nil
	}
	return "", nil
}

// LastSegment возвращает имя последнего файла-сегмента.
func LastSegment(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if !entries[i].IsDir() {
			return entries[i].Name(), nil
		}
	}
	return "", nil
}

// WriteSegment создаёт файл и записывает data.
func WriteSegment(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return f.Sync()
}

func upperBound(a []string, target string) int {
	lo, hi := 0, len(a)
	for lo < hi {
		mid := (lo + hi) / 2
		if a[mid] <= target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}
