package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Worker   WorkerConfig
	Logging  LoggingConfig
	EventBus EventBusConfig
}

type ServerConfig struct {
	Port            string
	Host            string
	ShutdownTimeout time.Duration
}

type WorkerConfig struct {
	PoolSize   int
	MaxRetries int
}

type LoggingConfig struct {
	Level string
}

type EventBusConfig struct {
	ChannelBufferSize int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default values")
	}

	return &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Worker: WorkerConfig{
			PoolSize:   getIntEnv("WORKER_POOL_SIZE", 10),
			MaxRetries: getIntEnv("MAX_RETRIES", 5),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		EventBus: EventBusConfig{
			ChannelBufferSize: getIntEnv("EVENT_CHANNEL_BUFFER_SIZE", 1000),
		},
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getIntEnv(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Invalid value for %s: %s, using default: %d", key, valueStr, defaultValue)
		return defaultValue
	}

	return value
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		log.Printf("Invalid duration for %s: %s, using default: %s", key, valueStr, defaultValue)
		return defaultValue
	}

	return value
}
