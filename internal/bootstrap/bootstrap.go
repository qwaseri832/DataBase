package bootstrap

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"

	"spider/internal/config"
	"spider/internal/database"
	"spider/internal/database/compute"
	"spider/internal/database/filesystem"
	"spider/internal/database/storage"
	"spider/internal/database/storage/engine"
	"spider/internal/database/storage/replication"
	"spider/internal/database/storage/wal"
	"spider/internal/network"
	"spider/internal/tools"
)

// App содержит все компоненты, готовые к запуску.
type App struct {
	db     *database.Database
	server *network.TCPServer
	wal    *wal.WAL
	master *replication.Master
	slave  *replication.Slave
	logger *zap.Logger
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}

	logger, err := buildLogger(cfg.Logging)
	if err != nil {
		return nil, err
	}

	w, err := buildWAL(cfg.WAL, logger)
	if err != nil {
		return nil, err
	}

	eng := buildEngine(cfg.Engine, logger)
	srv, err := buildServer(cfg.Server, logger)
	if err != nil {
		return nil, err
	}

	master, slave, err := buildReplication(cfg.Replication, cfg.WAL, logger)
	if err != nil {
		return nil, err
	}

	// Собираем Storage
	var sOpts []storage.Option
	if w != nil {
		sOpts = append(sOpts, storage.WithWAL(w))
	}
	if master != nil {
		sOpts = append(sOpts, storage.WithReplica(master))
	} else if slave != nil {
		sOpts = append(sOpts, storage.WithReplica(slave))
		sOpts = append(sOpts, storage.WithStream(slave.Stream()))
	}

	st := storage.New(eng, logger, sOpts...)
	parser := compute.NewParser()
	db := database.New(parser, st, logger)

	return &App{
		db: db, server: srv, wal: w,
		master: master, slave: slave,
		logger: logger,
	}, nil
}

// Run запускает все подсистемы и блокируется.
func (a *App) Run(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)

	if a.wal != nil {
		if a.slave != nil {
			g.Go(func() error { a.slave.Start(gctx); return nil })
		} else {
			g.Go(func() error { a.wal.Start(gctx); return nil })
		}
		if a.master != nil {
			g.Go(func() error { a.master.Start(gctx); return nil })
		}
	}

	g.Go(func() error {
		a.server.Serve(gctx, func(ctx context.Context, req []byte) []byte {
			resp := a.db.Handle(ctx, string(req))
			return []byte(resp)
		})
		return nil
	})

	err := g.Wait()
	_ = a.logger.Sync()
	return err
}

// ---- builders ----

func buildLogger(cfg *config.LoggingConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	output := "spider.log"
	if cfg != nil {
		if cfg.File != "" {
			output = cfg.File
		}
		switch cfg.Level {
		case "debug":
			level = zapcore.DebugLevel
		case "warn":
			level = zapcore.WarnLevel
		case "error":
			level = zapcore.ErrorLevel
		}
	}
	return zap.Config{
		Encoding:    "json",
		Level:       zap.NewAtomicLevelAt(level),
		OutputPaths: []string{output},
	}.Build()
}

func buildEngine(cfg *config.EngineConfig, logger *zap.Logger) *engine.Engine {
	var opts []engine.Option
	if cfg != nil && cfg.Partitions > 0 {
		opts = append(opts, engine.WithPartitions(cfg.Partitions))
	}
	return engine.New(logger, opts...)
}

func buildWAL(cfg *config.WALConfig, logger *zap.Logger) (*wal.WAL, error) {
	if cfg == nil {
		return nil, nil
	}
	dir := cfg.Directory
	if dir == "" {
		dir = "./data/wal"
	}
	maxSeg := 10 << 20 // 10MB default
	if cfg.SegmentMaxSize != "" {
		n, err := tools.ParseByteSize(cfg.SegmentMaxSize)
		if err != nil {
			return nil, err
		}
		maxSeg = n
	}
	batchSize := cfg.BatchSize
	if batchSize == 0 {
		batchSize = 100
	}
	timeout := cfg.BatchTimeout
	if timeout == 0 {
		timeout = 10_000_000 // 10ms
	}

	scanner := filesystem.NewDirScanner(dir)
	reader := wal.NewReader(scanner)
	seg := filesystem.NewSegment(dir, maxSeg)
	writer := wal.NewWriter(seg, logger)

	return wal.New(writer, reader, timeout, batchSize), nil
}

func buildServer(cfg *config.ServerConfig, logger *zap.Logger) (*network.TCPServer, error) {
	addr := ":4200"
	var opts []network.ServerOpt
	if cfg != nil {
		if cfg.Addr != "" {
			addr = cfg.Addr
		}
		if cfg.MaxClients > 0 {
			opts = append(opts, network.WithServerMaxClients(cfg.MaxClients))
		}
		if cfg.ReadBuffer != "" {
			n, _ := tools.ParseByteSize(cfg.ReadBuffer)
			if n > 0 {
				opts = append(opts, network.WithServerBuffer(n))
			}
		}
		if cfg.IdleTimeout > 0 {
			opts = append(opts, network.WithServerTimeout(cfg.IdleTimeout))
		}
	}
	return network.ListenTCP(addr, logger, opts...)
}

func buildReplication(
	repCfg *config.ReplicationConfig,
	walCfg *config.WALConfig,
	logger *zap.Logger,
) (*replication.Master, *replication.Slave, error) {
	if repCfg == nil {
		return nil, nil, nil
	}
	if walCfg == nil {
		return nil, nil, errors.New("replication requires WAL")
	}

	walDir := walCfg.Directory
	if walDir == "" {
		walDir = "./data/wal"
	}

	interval := repCfg.SyncInterval
	if interval == 0 {
		interval = 1_000_000_000 // 1s
	}

	if repCfg.Role == "master" {
		srv, err := network.ListenTCP(repCfg.MasterAddr, logger)
		if err != nil {
			return nil, nil, err
		}
		m := replication.NewMaster(srv, walDir, logger)
		return m, nil, nil
	}

	// slave
	client, err := network.DialTCP(repCfg.MasterAddr)
	if err != nil {
		return nil, nil, err
	}
	s := replication.NewSlave(client, walDir, interval, logger)
	return nil, s, nil
}
