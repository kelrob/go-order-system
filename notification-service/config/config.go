package config

import "os"

type Config struct {
	DatabaseURL string
	KafkaBroker string
	Port        string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/orderdb"),
		KafkaBroker: getEnv("KAFKA_BROKER", "localhost:9092"),
		Port:        getEnv("PORT", "8083"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
