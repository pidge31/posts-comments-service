package main

import (
	"log"

	"github.com/pidge31/posts-comments-service/internal/config"
)

func main() {
	cfg := config.Load()
	log.Printf("start posts-comments-service")
	log.Printf("port: %s", cfg.Port)
	log.Printf("storage type: %s", cfg.StorageType)
}
