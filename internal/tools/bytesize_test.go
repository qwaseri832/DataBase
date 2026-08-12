package tools

import "testing"

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"empty", "", 0, true},
		{"only digits", "500", 500, false},
		{"KB", "4KB", 4096, false},
		{"MB", "10MB", 10485760, false},
		{"GB", "1GB", 1073741824, false},
		{"lowercase", "5kb", 5120, false},
		{"no spaces", "2KB", 2048, false},
		{"invalid suffix", "10XB", 0, true},
		{"no digits", "MB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseByteSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseByteSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseByteSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
