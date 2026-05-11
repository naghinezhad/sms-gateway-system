package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/naghinezhad/sms-gateway-system/internal/model"
	"github.com/naghinezhad/sms-gateway-system/internal/service"
	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type EventHandler func(ctx context.Context, event *model.SmsSendEvent) error

type Consumer struct {
	reader     *kafka.Reader
	dltWriter  *kafka.Writer
	logger     *zap.Logger
	consumerID string
	handler    EventHandler
}

func NewConsumer(brokers []string, topic string, groupID string, consumerID string, logger *zap.Logger, handler EventHandler, dltWriter *kafka.Writer) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})

	return &Consumer{
		reader:     reader,
		dltWriter:  dltWriter,
		logger:     logger,
		consumerID: consumerID,
		handler:    handler,
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		c.logger.Info("sms event received",
			zap.String("consumer_id", c.consumerID),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
		)

		var event model.SmsSendEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("failed to decode sms event",
				zap.String("consumer_id", c.consumerID),
				zap.Int("partition", msg.Partition),
				zap.Error(err),
			)
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}

		handledErr := c.handler(ctx, &event)
		shouldCommit := handledErr == nil

		if handledErr != nil {
			switch {
			case errors.Is(handledErr, service.ErrDuplicateEvent):
				c.logger.Info("duplicate event detected. skipping",
					zap.String("consumer_id", c.consumerID),
					zap.String("message_id", event.MessageID),
				)
				shouldCommit = true
			case errors.Is(handledErr, service.ErrLockNotAcquired):
				c.logger.Info("lock not acquired. retry later",
					zap.String("consumer_id", c.consumerID),
					zap.String("message_id", event.MessageID),
				)
				shouldCommit = false
			default:
				c.logger.Error("failed to process sms event",
					zap.String("consumer_id", c.consumerID),
					zap.String("message_id", event.MessageID),
					zap.Error(handledErr),
				)
				if c.dltWriter != nil {
					dltMsg := kafka.Message{
						Value: msg.Value,
						Headers: append(msg.Headers, kafka.Header{
							Key:   "error",
							Value: []byte(handledErr.Error()),
						}),
						Time: time.Now().UTC(),
					}
					if err := c.dltWriter.WriteMessages(ctx, dltMsg); err != nil {
						c.logger.Error("failed to publish to dead-letter topic",
							zap.String("consumer_id", c.consumerID),
							zap.Error(err),
						)
						continue
					}
					shouldCommit = true
				}
			}
		}

		if !shouldCommit {
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("failed to commit message",
				zap.String("consumer_id", c.consumerID),
				zap.Error(err),
			)
		}
	}
}
