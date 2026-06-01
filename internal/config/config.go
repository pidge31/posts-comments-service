package config

import "os"

const (
	DefaultPort        = "8080"
	DefaultStorageType = "memory"
)

type Config struct {
	Port        string
	StorageType string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Port:        getEnv("PORT", DefaultPort),
		StorageType: getEnv("STORAGE_TYPE", DefaultStorageType),
		DatabaseURL: getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
