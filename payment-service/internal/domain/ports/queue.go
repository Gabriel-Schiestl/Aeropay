package ports

import "context"

type Publisher[T any] interface {
	Publish(message T) error
	Close()
}

type Consumer interface {
	Consume(ctx context.Context) error
}
