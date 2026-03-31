package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gmaps-scraper/config"
	"gmaps-scraper/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal...")
		cancel()
	}()

	srv, err := server.New(cfg)
	if err != nil {
		cancel()
		log.Fatalf("failed to create server: %v", err)
	}

	defer func() {
		if err := srv.Close(); err != nil {
			log.Printf("cleanup error: %v", err)
		}
	}()

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
