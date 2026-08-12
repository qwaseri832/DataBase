package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/qwaseri832/DataBase/internal/bootstrap"
	"github.com/qwaseri832/DataBase/internal/config"
)

func main() {
	path := flag.String("config", os.Getenv("CONFIG_FILE_NAME"),
		"path to YAML config (may also be set via CONFIG_FILE_NAME)")
	flag.Parse()

	if err := run(*path); err != nil {
		fmt.Fprintf(os.Stderr, "spider-server: %v\n", err)
		os.Exit(1)
	}
}

func run(path string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := &config.Config{}
	if path != "" {
		var err error
		if cfg, err = config.LoadFromFile(path); err != nil {
			return err
		}
	}

	app, err := bootstrap.New(cfg)
	if err != nil {
		return err
	}

	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
