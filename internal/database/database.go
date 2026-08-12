package database

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/database/compute"
	"github.com/qwaseri832/DataBase/internal/database/storage"
)

type Database struct {
	parser  *compute.Parser
	storage *storage.Storage
	logger  *zap.Logger
}

func New(p *compute.Parser, s *storage.Storage, l *zap.Logger) *Database {
	return &Database{parser: p, storage: s, logger: l}
}

func (d *Database) Handle(ctx context.Context, raw string) string {
	q, err := d.parser.Parse(raw)
	if err != nil {
		return fmt.Sprintf("[error] %s", err)
	}

	switch q.Cmd() {
	case compute.CmdSet:
		return d.doSet(ctx, q)
	case compute.CmdGet:
		return d.doGet(ctx, q)
	case compute.CmdDel:
		return d.doDel(ctx, q)
	default:
		return "[error] unknown command"
	}
}

func (d *Database) doSet(ctx context.Context, q compute.Query) string {
	a := q.Args()
	if err := d.storage.Set(ctx, a[0], a[1]); err != nil {
		return fmt.Sprintf("[error] %s", err)
	}
	return "[ok]"
}

func (d *Database) doGet(ctx context.Context, q compute.Query) string {
	val, err := d.storage.Get(ctx, q.Args()[0])
	if errors.Is(err, storage.ErrNotFound) {
		return "[not found]"
	}
	if err != nil {
		return fmt.Sprintf("[error] %s", err)
	}
	return fmt.Sprintf("[ok] %s", val)
}

func (d *Database) doDel(ctx context.Context, q compute.Query) string {
	if err := d.storage.Del(ctx, q.Args()[0]); err != nil {
		return fmt.Sprintf("[error] %s", err)
	}
	return "[ok]"
}
