package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"

	"github.com/qwaseri832/DataBase/internal/config"
	"github.com/qwaseri832/DataBase/internal/database"
	"github.com/qwaseri832/DataBase/internal/database/compute"
	"github.com/qwaseri832/DataBase/internal/database/filesystem"
	"github.com/qwaseri832/DataBase/internal/database/storage"
	"github.com/qwaseri832/DataBase/internal/database/storage/engine"
	"github.com/qwaseri832/DataBase/internal/database/storage/replication"
	"github.com/qwaseri832/DataBase/internal/database/storage/wal"
	"github.com/qwaseri832/DataBase/internal/network"
	"github.com/qwaseri832/DataBase/internal/tools"
)

const (
	defaultWALDir       = "./data/wal"
	defaultSegmentSize  = 10 << 20
	defaultBatchSize    = 100
	defaultBatchTimeout = 10 * time.Millisecond
	defaultSyncInterval = time.Second
	defaultServerAddr   = ":4200"
	defaultLogFile      = "spider.log"
	engineInMemory      = "in_memory"
)

type App struct {
	db      *database.Database
	storage *storage.Storage
	server  *network.TCPServer
	wal     *wal.WAL
	master  *replication.Master
	slave   *replication.Slave
	logger  *zap.Logger
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	logger, err := buildLogger(cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("logger: %w", err)
	}

	eng, err := buildEngine(cfg.Engine, logger)
	if err != nil {
		return nil, fmt.Errorf("engine: %w", err)
	}

	w, err := buildWAL(cfg.WAL, logger)
	if err != nil {
		return nil, fmt.Errorf("WAL: %w", err)
	}

	master, slave, err := buildReplication(cfg.Replication, cfg.WAL, logger)
	if err != nil {
		return nil, fmt.Errorf("replication: %w", err)
	}

	srv, err := buildServer(cfg.Server, logger)
	if err != nil {
		return nil, fmt.Errorf("server: %w", err)
	}

	var sOpts []storage.Option
	if w != nil {
		sOpts = append(sOpts, storage.WithWAL(w))
	}
	switch {
	case master != nil:
		sOpts = append(sOpts, storage.WithReplica(master))
	case slave != nil:
		sOpts = append(sOpts, storage.WithReplica(slave), storage.WithStream(slave.Stream()))
	}

	st, err := storage.New(eng, logger, sOpts...)
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}

	return &App{
		db:      database.New(compute.NewParser(), st, logger),
		storage: st,
		server:  srv,
		wal:     w,
		master:  master,
		slave:   slave,
		logger:  logger,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)

	if a.wal != nil {
		g.Go(func() error { a.wal.Run(gctx); return nil })
	}
	if a.master != nil {
		g.Go(func() error { a.master.Run(gctx); return nil })
	}
	if a.slave != nil {
		g.Go(func() error { a.slave.Run(gctx); return nil })
		g.Go(func() error { a.storage.ApplyStream(gctx); return nil })
	}

	g.Go(func() error {
		a.server.Serve(gctx, func(ctx context.Context, req []byte) []byte {
			return []byte(a.db.Handle(ctx, string(req)))
		})
		return nil
	})

	a.logger.Info("spider started", zap.String("addr", a.server.Addr().String()))

	err := g.Wait()
	_ = a.logger.Sync()
	return err
}

func buildLogger(cfg *config.LoggingConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	output := defaultLogFile

	if cfg != nil {
		if cfg.File != "" {
			output = cfg.File
		}
		if cfg.Level != "" {
			if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
				return nil, fmt.Errorf("unknown log level %q", cfg.Level)
			}
		}
	}

	return zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(level),
		OutputPaths:      []string{output, "stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig:    zap.NewProductionEncoderConfig(),
	}.Build()
}

func buildEngine(cfg *config.EngineConfig, logger *zap.Logger) (*engine.Engine, error) {
	if cfg == nil {
		return engine.New(logger), nil
	}

	if cfg.Type != "" && cfg.Type != engineInMemory {
		return nil, fmt.Errorf("unknown engine type %q, want %s", cfg.Type, engineInMemory)
	}

	var opts []engine.Option
	if cfg.Partitions > 0 {
		opts = append(opts, engine.WithPartitions(cfg.Partitions))
	}

	return engine.New(logger, opts...), nil
}

func buildWAL(cfg *config.WALConfig, logger *zap.Logger) (*wal.WAL, error) {
	if cfg == nil {
		return nil, nil
	}

	dir := orString(cfg.Directory, defaultWALDir)

	maxSeg := defaultSegmentSize
	if cfg.SegmentMaxSize != "" {
		n, err := tools.ParseByteSize(cfg.SegmentMaxSize)
		if err != nil {
			return nil, fmt.Errorf("segment_max_size: %w", err)
		}
		maxSeg = n
	}

	reader := wal.NewReader(filesystem.NewDirScanner(dir))
	writer := wal.NewWriter(filesystem.NewSegment(dir, maxSeg), logger)

	return wal.New(writer, reader,
		orDuration(cfg.BatchTimeout, defaultBatchTimeout),
		orInt(cfg.BatchSize, defaultBatchSize),
	), nil
}

func buildServer(cfg *config.ServerConfig, logger *zap.Logger) (*network.TCPServer, error) {
	addr := defaultServerAddr
	var opts []network.ServerOpt

	if cfg != nil {
		addr = orString(cfg.Addr, defaultServerAddr)
		if cfg.MaxClients > 0 {
			opts = append(opts, network.WithServerMaxClients(cfg.MaxClients))
		}
		if cfg.ReadBuffer != "" {
			n, err := tools.ParseByteSize(cfg.ReadBuffer)
			if err != nil {
				return nil, fmt.Errorf("read_buffer: %w", err)
			}
			opts = append(opts, network.WithServerBuffer(n))
		}
		if cfg.MaxMessageSize != "" {
			n, err := tools.ParseByteSize(cfg.MaxMessageSize)
			if err != nil {
				return nil, fmt.Errorf("max_message_size: %w", err)
			}
			opts = append(opts, network.WithServerMaxMessage(n))
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
	if repCfg == nil || repCfg.Role == "" {
		return nil, nil, nil
	}
	if walCfg == nil {
		return nil, nil, errors.New("replication requires WAL to be enabled")
	}
	if repCfg.MasterAddr == "" {
		return nil, nil, errors.New("master_addr is required")
	}

	walDir := orString(walCfg.Directory, defaultWALDir)

	switch repCfg.Role {
	case "master":
		srv, err := network.ListenTCP(repCfg.MasterAddr, logger)
		if err != nil {
			return nil, nil, err
		}
		return replication.NewMaster(srv, walDir, logger), nil, nil

	case "slave":
		client, err := network.DialTCP(repCfg.MasterAddr)
		if err != nil {
			return nil, nil, err
		}
		interval := orDuration(repCfg.SyncInterval, defaultSyncInterval)
		return nil, replication.NewSlave(client, walDir, interval, logger), nil

	default:
		return nil, nil, fmt.Errorf("unknown replication role %q, want master or slave", repCfg.Role)
	}
}

func orString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func orInt(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

func orDuration(v, fallback time.Duration) time.Duration {
	if v <= 0 {
		return fallback
	}
	return v
}
