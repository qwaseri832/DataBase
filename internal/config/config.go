package config

import (
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Engine      *EngineConfig      `yaml:"engine"`
	WAL         *WALConfig         `yaml:"wal"`
	Replication *ReplicationConfig `yaml:"replication"`
	Server      *ServerConfig      `yaml:"server"`
	Logging     *LoggingConfig     `yaml:"logging"`
}

type EngineConfig struct {
	Type       string `yaml:"type"`
	Partitions int    `yaml:"partitions"`
}

type WALConfig struct {
	BatchSize      int           `yaml:"batch_size"`
	BatchTimeout   time.Duration `yaml:"batch_timeout"`
	SegmentMaxSize string        `yaml:"segment_max_size"`
	Directory      string        `yaml:"directory"`
}

type ReplicationConfig struct {
	Role         string        `yaml:"role"`
	MasterAddr   string        `yaml:"master_addr"`
	SyncInterval time.Duration `yaml:"sync_interval"`
}

type ServerConfig struct {
	Addr        string        `yaml:"addr"`
	MaxClients  int           `yaml:"max_clients"`
	ReadBuffer  string        `yaml:"read_buffer"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`

	MaxMessageSize string `yaml:"max_message_size"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

func LoadFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	cfg, err := Load(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func Load(r io.Reader) (*Config, error) {
	dec := yaml.NewDecoder(r)

	dec.KnownFields(true)

	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		if err == io.EOF {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
