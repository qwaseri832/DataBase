package database

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/database/compute"
	"github.com/qwaseri832/DataBase/internal/database/storage"
	"github.com/qwaseri832/DataBase/internal/database/storage/engine"
)

func setupTestDB(t *testing.T) *Database {
	t.Helper()

	logger := zap.NewNop()
	st, err := storage.New(engine.New(logger, engine.WithPartitions(4)), logger)
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	return New(compute.NewParser(), st, logger)
}

func TestDatabaseSetGet(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	if got := db.Handle(ctx, "SET name John"); got != "[ok]" {
		t.Errorf("SET = %v, ожидалось [ok]", got)
	}
	if got := db.Handle(ctx, "GET name"); got != "[ok] John" {
		t.Errorf("GET = %v, ожидалось [ok] John", got)
	}
}

func TestDatabaseGetMissingKey(t *testing.T) {
	db := setupTestDB(t)

	if got := db.Handle(context.Background(), "GET нет-такого"); got != "[not found]" {
		t.Errorf("GET несуществующего = %v, ожидалось [not found]", got)
	}
}

func TestDatabaseDel(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	db.Handle(ctx, "SET temp value")
	if got := db.Handle(ctx, "DEL temp"); got != "[ok]" {
		t.Errorf("DEL = %v, ожидалось [ok]", got)
	}
	if got := db.Handle(ctx, "GET temp"); got != "[not found]" {
		t.Errorf("GET после DEL = %v, ожидалось [not found]", got)
	}
}

func TestDatabaseErrors(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"неизвестная команда", "FOO bar", "[error] unknown command"},
		{"мало аргументов у SET", "SET key", "[error] wrong number of arguments"},
		{"нет аргументов у GET", "GET", "[error] wrong number of arguments"},
		{"пустой запрос", "   ", "[error] empty query"},
		{"лишние аргументы у DEL", "DEL a b", "[error] wrong number of arguments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := db.Handle(ctx, tt.command); got != tt.want {
				t.Errorf("Handle(%q) = %v, ожидалось %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestDatabaseCommandsAreCaseInsensitive(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	db.Handle(ctx, "set key значение")
	if got := db.Handle(ctx, "Get key"); got != "[ok] значение" {
		t.Errorf("Get = %v, ожидалось [ok] значение", got)
	}
}

func TestDatabaseRespectsCancelledContext(t *testing.T) {
	db := setupTestDB(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if got := db.Handle(ctx, "SET key value"); got == "[ok]" {
		t.Error("SET с отменённым контекстом вернул [ok]")
	}
}
