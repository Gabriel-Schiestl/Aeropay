package config

import "github.com/caarlos0/env/v6"

type DBConfig struct {
	Host     string `env:"DB_HOST"`
	Port     string `env:"DB_PORT"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	Name     string `env:"DB_NAME"`
	Driver   string `env:"DB_DRIVER"`
}

func LoadDBConfig() *DBConfig {
	var config DBConfig
	if err := env.Parse(&config); err != nil {
		panic(err)
	}
	return &config
}