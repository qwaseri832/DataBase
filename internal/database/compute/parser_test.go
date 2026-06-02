package compute

import "testing"

func TestParser_Parse(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name    string
		input   string
		wantCmd int
		wantErr bool
	}{
		{"SET valid", "SET key value", CmdSet, false},
		{"GET valid", "GET key", CmdGet, false},
		{"DEL valid", "DEL key", CmdDel, false},
		{"SET wrong args", "SET key", CmdUnknown, true},
		{"GET wrong args", "GET", CmdUnknown, true},
		{"unknown command", "UNKNOWN foo", CmdUnknown, true},
		{"empty query", "", CmdUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLookupCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want int
	}{
		{"SET", "SET", CmdSet},
		{"GET", "GET", CmdGet},
		{"DEL", "DEL", CmdDel},
		{"unknown", "FOO", CmdUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lookupCommand(tt.cmd); got != tt.want {
				t.Errorf("lookupCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}