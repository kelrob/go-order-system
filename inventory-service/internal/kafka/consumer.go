package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kelrob/shared/logger"
	"github.com/twmb/franz-go/pkg/kgo"
)

type ConsumerHandler func(data []byte) error

type Consumer struct {
	client   *kgo.Client
	topic    string
	handler  ConsumerHandler
	dlq      *DLQ
	maxRetry int
	appLog   *logger.Logger
}

func NewConsumer(brokers []string, topic, groupId string, handler ConsumerHandler, dlq *DLQ, appLog *logger.Logger) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupId),
		kgo.ConsumeTopics(topic),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	return &Consumer{
		client:   client,
		topic:    topic,
		handler:  handler,
		dlq:      dlq,
		maxRetry: 3,
		appLog:   appLog,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	c.appLog.Log("Consumer started, waiting for messages...", nil)

	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return nil
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("fetch error: %v", errs)
		}

		fetches.EachRecord(func(record *kgo.Record) {
			var envelope struct {
				TraceId string `json:"trace_id"`
			}
			json.Unmarshal(record.Value, &envelope)

			c.appLog.Info(envelope.TraceId, record.Topic, "Received event", nil)

			var err error
			for attempt := 1; attempt <= c.maxRetry; attempt++ {
				err = c.handler(record.Value)
				if err == nil {
					return
				}
				c.appLog.Warn(envelope.TraceId, record.Topic, "Attempt failed", map[string]any{
					"attempt": attempt,
					"error":   err.Error(),
				})
			}

			c.appLog.Error(envelope.TraceId, record.Topic, "Message failed, sending to DLQ after max retries", nil, map[string]any{
				"max_retries": c.maxRetry,
			})
			dlqErr := c.dlq.Send(ctx, record.Topic, record.Value)
			if dlqErr != nil {
				c.appLog.Error(envelope.TraceId, c.topic, "failed to send to DLQ", dlqErr, nil)
			}
		})
	}
}

func (c *Consumer) Close() {
	c.client.Close()
}
