package ports

import "context"

type Publisher[T any] interface {
	Publish(message T) error
}

type Consumer interface {
	Consume(ctx context.Context) error
}