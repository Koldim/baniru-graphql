package config

import "os"

type Config struct {
	DatabaseURL string
	HTTPAddr    string
}

func FromEnv() Config {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddr:    os.Getenv("HTTP_ADDR"),
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	return cfg
}
