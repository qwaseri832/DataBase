package storage

import (
	"testing"
)

func TestIDGen_Next(t *testing.T) {
	gen := NewIDGen(0)

	// Test sequential IDs
	for i := int64(1); i <= 10; i++ {
		if got := gen.Next(); got != i {
			t.Errorf("Next() = %v, want %v", got, i)
		}
	}
}

func TestIDGen_StartFrom(t *testing.T) {
	gen := NewIDGen(100)

	if got := gen.Next(); got != 101 {
		t.Errorf("Next() from start 100 = %v, want 101", got)
	}
}

func TestIDGen_Overflow(t *testing.T) {
	gen := NewIDGen(9223372036854775800) // близко к MaxInt64

	for i := 0; i < 30; i++ {
		_ = gen.Next()
	}
	// Не должно паниковать, сбросится до 0
	if gen.Next() < 0 {
		t.Error("IDGen overflow handling failed")
	}
}