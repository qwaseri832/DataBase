package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSegmentRotationDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	seg := NewSegment(dir, 4)
	defer seg.Close()

	payloads := []string{"aaaa", "bbbb", "cccc"}
	for _, p := range payloads {
		if err := seg.Write([]byte(p)); err != nil {
			t.Fatalf("Write(%q): %v", p, err)
		}
	}

	names, err := segmentNames(dir)
	if err != nil {
		t.Fatalf("segmentNames: %v", err)
	}
	if len(names) != len(payloads) {
		t.Fatalf("создано сегментов: %d, ожидалось %d (%v)", len(names), len(payloads), names)
	}

	for i, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("чтение %s: %v", name, err)
		}
		if string(data) != payloads[i] {
			t.Errorf("сегмент %s содержит %q, ожидалось %q", name, data, payloads[i])
		}
	}
}

func TestSegmentCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data", "wal")
	seg := NewSegment(dir, 1<<20)
	defer seg.Close()

	if err := seg.Write([]byte("запись")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("каталог %s не создан: %v", dir, err)
	}
}

func TestSegmentNamesSortInCreationOrder(t *testing.T) {
	dir := t.TempDir()
	seg := NewSegment(dir, 1)
	defer seg.Close()

	const count = 12
	for i := 0; i < count; i++ {
		if err := seg.Write([]byte{byte('a' + i)}); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	names, err := segmentNames(dir)
	if err != nil {
		t.Fatalf("segmentNames: %v", err)
	}
	if len(names) != count {
		t.Fatalf("сегментов: %d, ожидалось %d", len(names), count)
	}

	for i := 0; i < count; i++ {
		data, err := os.ReadFile(filepath.Join(dir, names[i]))
		if err != nil {
			t.Fatalf("чтение %s: %v", names[i], err)
		}
		if want := byte('a' + i); data[0] != want {
			t.Errorf("сегмент %d (%s) содержит %q, ожидалось %q", i, names[i], data[0], want)
		}
	}
}

func TestSegmentNamesIgnoresForeignFiles(t *testing.T) {
	dir := t.TempDir()
	seg := NewSegment(dir, 1<<20)
	if err := seg.Write([]byte("запись")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := seg.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "spider.log"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("подготовка: %v", err)
	}

	names, err := segmentNames(dir)
	if err != nil {
		t.Fatalf("segmentNames: %v", err)
	}
	if len(names) != 1 {
		t.Fatalf("сегментов: %v, ожидался ровно один", names)
	}
	if !strings.HasPrefix(names[0], SegmentPrefix) {
		t.Errorf("в выборку попал посторонний файл: %s", names[0])
	}
}

func TestWriteSegmentReplacesContentCompletely(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, SegmentPrefix+"0000000000001_000001.log")

	if err := WriteSegment(path, []byte("длинное прежнее содержимое")); err != nil {
		t.Fatalf("первая запись: %v", err)
	}
	if err := WriteSegment(path, []byte("коротко")); err != nil {
		t.Fatalf("вторая запись: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("чтение: %v", err)
	}
	if string(data) != "коротко" {
		t.Errorf("получено %q, ожидалось \"коротко\"", data)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("в каталоге %d файлов, ожидался 1", len(entries))
	}
}

func TestNextAndLastSegment(t *testing.T) {
	dir := t.TempDir()
	seg := NewSegment(dir, 1)
	defer seg.Close()

	for i := 0; i < 3; i++ {
		if err := seg.Write([]byte{byte('x')}); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}
	names, err := segmentNames(dir)
	if err != nil {
		t.Fatalf("segmentNames: %v", err)
	}

	t.Run("следующий после пустого — первый", func(t *testing.T) {
		got, err := NextSegment(dir, "")
		if err != nil || got != names[0] {
			t.Errorf("NextSegment(\"\") = %q, %v; ожидалось %q", got, err, names[0])
		}
	})

	t.Run("следующий после последнего — пусто", func(t *testing.T) {
		got, err := NextSegment(dir, names[len(names)-1])
		if err != nil || got != "" {
			t.Errorf("NextSegment(последний) = %q, %v; ожидалась пустая строка", got, err)
		}
	})

	t.Run("последний сегмент", func(t *testing.T) {
		got, err := LastSegment(dir)
		if err != nil || got != names[len(names)-1] {
			t.Errorf("LastSegment() = %q, %v; ожидалось %q", got, err, names[len(names)-1])
		}
	})
}

func TestMissingDirectoryIsNotAnError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "нет-такого")

	if _, err := LastSegment(dir); err != nil {
		t.Errorf("LastSegment() = %v, ожидалось отсутствие ошибки", err)
	}
	if err := NewDirScanner(dir).ForEach(func([]byte) error { return nil }); err != nil {
		t.Errorf("ForEach() = %v, ожидалось отсутствие ошибки", err)
	}
}
