package database

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"spider/internal/database/compute"
	"spider/internal/database/storage"
	"spider/internal/database/storage/engine"
)

func setupTestDB() *Database {
	logger, _ := zap.NewDevelopment()
	eng := engine.New(logger, engine.WithPartitions(4))
	st := storage.New(eng, logger)
	parser := compute.NewParser()
	return New(parser, st, logger)
}

func TestDatabase_SetGet(t *testing.T) {
	db := setupTestDB()
	ctx := context.Background()

	// Test SET
	if got := db.Handle(ctx, "SET name John"); got != "[ok]" {
		t.Errorf("SET = %v, want [ok]", got)
	}

	// Test GET
	if got := db.Handle(ctx, "GET name"); got != "[ok] John" {
		t.Errorf("GET = %v, want [ok] John", got)
	}
}

func TestDatabase_Del(t *testing.T) {
	db := setupTestDB()
	ctx := context.Background()

	db.Handle(ctx, "SET temp value")
	db.Handle(ctx, "DEL temp")

	if got := db.Handle(ctx, "GET temp"); got != "[not found]" {
		t.Errorf("GET after DEL = %v, want [not found]", got)
	}
}

func TestDatabase_Errors(t *testing.T) {
	db := setupTestDB()
	ctx := context.Background()

	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"unknown command", "FOO bar", "[error] unknown command"},
		{"wrong args SET", "SET key", "[error] wrong number of arguments"},
		{"wrong args GET", "GET", "[error] wrong number of arguments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := db.Handle(ctx, tt.command); got != tt.want {
				t.Errorf("Handle(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}