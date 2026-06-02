package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"spider/internal/config"
	"spider/internal/bootstrap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := &config.Config{}

	if path := os.Getenv("CONFIG_FILE_NAME"); path != "" {
		var err error
		cfg, err = config.LoadFromFile(path)
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
	}

	app, err := bootstrap.New(cfg)
	if err != nil {
		log.Fatalf("failed to bootstrap: %v", err)
	}

	if err := app.Run(ctx); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
