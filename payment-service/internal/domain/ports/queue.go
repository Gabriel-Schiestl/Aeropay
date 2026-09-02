package ports

import "context"

type Publisher interface {
	Publish(message any, key string) error
	Close()
	CreateTopic() error
}

type Consumer interface {
	Consume(ctx context.Context) error
}
