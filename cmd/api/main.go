package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pidge31/posts-comments-service/internal/config"
	"github.com/pidge31/posts-comments-service/internal/server"
)

func main() {
	cfg := config.Load()

	httpServer := server.New(cfg.Port)

	go func() {
		log.Printf("server starting posts-comments-service on port %s", cfg.Port)

		if err := httpServer.Run(); err != nil {
			log.Fatalf("failed to run server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	log.Println("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("failed to shutdown server: %v", err)
	}

	log.Println("server stopped")

}
