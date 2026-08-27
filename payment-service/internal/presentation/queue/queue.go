package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/config"
	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/domain/ports"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

type publisher[T any] struct {
	client *kgo.Client
	config *config.QueueConfig
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
		config: config,
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

func (p *publisher[T]) CreateTopic() error {
	adminClient := kadm.NewClient(p.client)

	resp, err := adminClient.CreateTopics(context.Background(), int32(p.config.TopicMaxPartitions), 1, nil, p.config.Topic)
	if err != nil {
		panic(err) 
	}

	for _, ctr := range resp {
		if ctr.Err != nil {
			if errors.Is(ctr.Err, kerr.TopicAlreadyExists) {
				fmt.Println("Topic already exists.")
			} else {
				fmt.Printf("Failed: %v\n", ctr.Err)
				return ctr.Err
			}
		} else {
			fmt.Println("Topic created.")
		}
	}

	return nil
}

func (p *publisher[T]) Publish(message T) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	record := &kgo.Record{
		Topic: p.config.Topic,
		Value: data,
	}

	if err := p.client.ProduceSync(context.Background(), record).FirstErr(); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}
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
