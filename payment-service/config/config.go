package config

import "github.com/kelrob/shared/helpers"

type Config struct {
	DatabaseURL string
	KafkaBroker string
	Port        string
}

func Load() *Config {
	return &Config{
		DatabaseURL: helpers.GetEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5434/paymentdb"),
		KafkaBroker: helpers.GetEnv("KAFKA_BROKER", "localhost:9092"),
		Port:        helpers.GetEnv("PORT", "8082"),
	}
}
