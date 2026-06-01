package config

import "os"

type Config struct {
	Port        string
	StorageType string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		StorageType: getEnv("STORAGE_TYPE", "memory"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
