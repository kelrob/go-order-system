package config

import "github.com/kelrob/shared/env"

type Config struct {
	DatabaseURL string
	KafkaBroker string
	Port        string
}

func Load() *Config {
	return &Config{
		DatabaseURL: env.Get("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/inventorydb"),
		KafkaBroker: env.Get("KAFKA_BROKER", "localhost:9092"),
		Port:        env.Get("PORT", "8081"),
	}
}
