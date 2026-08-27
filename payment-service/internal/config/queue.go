package config

import "github.com/caarlos0/env/v6"

type QueueConfig struct {
	Topic         string   `env:"QUEUE_TOPIC"`
	Brokers       []string `env:"QUEUE_BROKERS" envSeparator:","`
	MaxConsumers  int      `env:"QUEUE_MAX_CONSUMERS" envDefault:"5"`
	ConsumerGroup string   `env:"QUEUE_CONSUMER_GROUP"`
	InstanceID    string   `env:"QUEUE_INSTANCE_ID"`
	TopicMaxPartitions int `env:"QUEUE_TOPIC_MAX_PARTITIONS" envDefault:"3"`
}

func LoadQueueConfig() *QueueConfig {
	var config QueueConfig
	if err := env.Parse(&config); err != nil {
		panic(err)
	}
	return &config
}
