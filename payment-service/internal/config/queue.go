package config

import "github.com/caarlos0/env/v6"

type QueueConfig struct {
	Topic     string `env:"QUEUE_TOPIC"`
	Brokers   []string `env:"QUEUE_BROKERS" envSeparator:","`
	MaxConsumers int    `env:"QUEUE_MAX_CONSUMERS" envDefault:"5"`
}

func LoadQueueConfig() *QueueConfig {
	var config QueueConfig
	if err := env.Parse(&config); err != nil {
		panic(err)
	}
	return &config
}