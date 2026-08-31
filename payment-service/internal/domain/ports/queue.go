package ports

import "context"

type Publisher[T any] interface {
	Publish(message T, key string) error
	Close()
	CreateTopic() error
}

type Consumer interface {
	Consume(ctx context.Context) error
}
