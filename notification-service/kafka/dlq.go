package kafka

import (
	"context"
	"fmt"

	"github.com/kelrob/shared/logger"
	"github.com/twmb/franz-go/pkg/kgo"
)

type DLQ struct {
	client *kgo.Client
	appLog *logger.Logger
}

func NewDLQ(brokers []string, appLog *logger.Logger) (*DLQ, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create dlq client: %w", err)
	}

	return &DLQ{client: client, appLog: appLog}, nil
}

func (d *DLQ) Send(ctx context.Context, topic string, data []byte) error {
	dlqTopic := topic + ".dlq"

	record := &kgo.Record{
		Topic: dlqTopic,
		Value: data,
	}

	err := d.client.ProduceSync(ctx, record).FirstErr()
	if err != nil {
		return fmt.Errorf("failed to send to dlq: %w", err)
	}

	d.appLog.Log("Message sent to DLQ topic", map[string]any{
		"topic": dlqTopic,
	})
	return nil
}

func (d *DLQ) Close() {
	d.client.Close()
}
