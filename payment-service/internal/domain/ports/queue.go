package ports

import "context"

type Message struct {
	ID    int
	Key   string
	Value any
}

type Publisher interface {
	Publish(ctx context.Context, messages []Message) ([]int, error)
	Close()
	CreateTopic() error
}

type Consumer interface {
	Consume(ctx context.Context) error
}
