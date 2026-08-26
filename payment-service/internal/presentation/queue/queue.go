package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/twmb/franz-go/pkg/kgo"
)

type publisher[T any] struct {
	client *kgo.Client
}

type consumer[T any] struct {
	config  *config.QueueConfig
	handler func(props T) error
}

func NewPublisher[T any](config *config.QueueConfig) ports.Publisher[T] {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(config.Brokers...),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create Kafka client: %v", err))
	}

	return &publisher[T]{
		client: client,
	}
}

func NewConsumer[T any](config *config.QueueConfig, handler func(props T) error) *consumer[T] {
	return &consumer[T]{
		config:  config,
		handler: handler,
	}
}

func (p *publisher[T]) Close() {
	p.client.Close()
}

func (p *publisher[T]) Publish(message T) error {
	// Implement the logic to publish a message to the queue
	return nil
}

func (q *consumer[T]) Consume(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Add(q.config.MaxConsumers)
	for i := range q.config.MaxConsumers {
		go func(i int) {
			defer wg.Done()

			client, err := kgo.NewClient(
				kgo.SeedBrokers(q.config.Brokers...),
				kgo.ConsumerGroup(q.config.ConsumerGroup),
				kgo.ConsumeTopics(q.config.Topic),
				kgo.InstanceID(q.config.InstanceID+fmt.Sprintf("-%d", i)),
			)
			if err != nil {
				panic(fmt.Sprintf("failed to create Kafka client: %v", err))
			}
			defer client.Close()

			for {
				fetches := client.PollFetches(ctx)

				fetches.EachRecord(func(r *kgo.Record) {
					q.processRecord(r)
				})

				if err := fetches.Errors(); len(err) > 0 {
					for _, e := range err {
						if e.Err == context.Canceled {
							return
						}
						fmt.Printf("error fetching records: %v\n", e.Err)
					}
				}
			}
		}(i)
	}
	wg.Wait()

	return nil
}

func (q *consumer[T]) processRecord(r *kgo.Record) {
	var message T
	if err := json.Unmarshal(r.Value, &message); err != nil {
		fmt.Printf("error unmarshalling record: %v\n", err)
		return
	}

	if err := q.handler(message); err != nil {
		fmt.Printf("error processing message: %v\n", err)
	}
}
