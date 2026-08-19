package config

import "github.com/caarlos0/env/v6"

type HTTPConfig struct {
	Port     string `env:"HTTP_PORT"`
}

func LoadHTTPConfig() *HTTPConfig {
	var config HTTPConfig
	if err := env.Parse(&config); err != nil {
		panic(err)
	}
	return &config
}