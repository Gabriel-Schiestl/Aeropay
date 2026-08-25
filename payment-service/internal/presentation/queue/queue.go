package queue

import (
	"context"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
)

type publisher[T any] struct {
	topic string
}

type consumer[T any] struct {
	topic string
	handler func(props T) error
}

func NewPublisher[T any](config *config.QueueConfig) ports.Publisher[T] {
	return &publisher[T]{
		topic: config.Topic,
	}
}

func NewConsumer[T any](config *config.QueueConfig, handler func(props T) error) *consumer[T] {
	return &consumer[T]{
		topic: config.Topic,
		handler: handler,
	}
}

func (q *publisher[T]) Publish(message T) error {
	// Implement the logic to publish a message to the queue
	return nil
}

func (q *consumer[T]) Consume(ctx context.Context) error {
	// Implement the logic to consume messages from the queue and call the handler,
	// stopping when ctx is done.
	return nil
}

