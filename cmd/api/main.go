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
	"github.com/pidge31/posts-comments-service/internal/ports"
	"github.com/pidge31/posts-comments-service/internal/server"
	"github.com/pidge31/posts-comments-service/internal/storage/memory"
	"github.com/pidge31/posts-comments-service/internal/storage/postgres"
	"github.com/pidge31/posts-comments-service/internal/subscriptions"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var postRepository ports.PostRepository
	var commentRepository ports.CommentRepository

	switch cfg.StorageType {
	case "memory":
		store := memory.NewStore()

		postRepository = memory.NewPostRepository(store)
		commentRepository = memory.NewCommentRepository(store)

	case "postgres":
		if cfg.DatabaseURL == "" {
			log.Fatal("DATABASE_URL is required when STORAGE_TYPE=postgres")
		}

		pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("failed to connect postgres: %v", err)
		}
		defer pool.Close()

		postRepository = postgres.NewPostRepository(pool)
		commentRepository = postgres.NewCommentRepository(pool)

	default:
		log.Fatalf("unsupported storage type %q", cfg.StorageType)
	}

	commentBroker := subscriptions.NewBroker()

	postService := app.NewPostService(postRepository)
	commentService := app.NewCommentService(postRepository, commentRepository, commentBroker)

	graphQLHandler := graph.NewHandler(postService, commentService, commentBroker)

	httpServer := server.New(cfg.Port, graphQLHandler)

	go func() {
		log.Printf("starting posts-comments-service on port %s", cfg.Port)
		log.Printf("storage type: %s", cfg.StorageType)

		if err := httpServer.Run(); err != nil {
			log.Fatalf("failed to run server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	log.Println("shutting down server")

	commentBroker.Shutdown()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("failed to shutdown server: %v", err)
	}

	log.Println("server stopped")
}
