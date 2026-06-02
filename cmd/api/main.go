package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pidge31/posts-comments-service/internal/app"
	"github.com/pidge31/posts-comments-service/internal/config"
	"github.com/pidge31/posts-comments-service/internal/graph"
	"github.com/pidge31/posts-comments-service/internal/server"
	"github.com/pidge31/posts-comments-service/internal/storage/memory"
	"github.com/pidge31/posts-comments-service/internal/subscriptions"
)

func main() {
	cfg := config.Load()

	if cfg.StorageType != "memory" {
		log.Fatalf("unsupported storage type %q", cfg.StorageType)
	}

	store := memory.NewStore()

	postRepository := memory.NewPostRepository(store)
	commentRepository := memory.NewCommentRepository(store)

	commentBroker := subscriptions.NewBroker()

	postService := app.NewPostService(postRepository)
	commentService := app.NewCommentService(postRepository, commentRepository, commentBroker)

	graphQLHandler := graph.NewHandler(postService, commentService, commentBroker)

	httpServer := server.New(cfg.Port, graphQLHandler)

	go func() {
		log.Printf("starting posts-comments-service on port %s", cfg.Port)

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
