package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/naghinezhad/sms-gateway-system/internal/model"
	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer struct {
	standardWriter *kafka.Writer
	expressWriter  *kafka.Writer
	logger         *zap.Logger
}

func NewProducer(brokers []string, standardTopic string, expressTopic string, logger *zap.Logger) *Producer {
	standardWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        standardTopic,
		RequiredAcks: kafka.RequireOne,
		Balancer:     &kafka.Hash{},
	}

	expressWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        expressTopic,
		RequiredAcks: kafka.RequireOne,
		Balancer:     &kafka.Hash{},
	}

	return &Producer{
		standardWriter: standardWriter,
		expressWriter:  expressWriter,
		logger:         logger,
	}
}

func (p *Producer) Close() error {
	var err error
	if p.standardWriter != nil {
		if closeErr := p.standardWriter.Close(); closeErr != nil {
			err = closeErr
		}
	}
	if p.expressWriter != nil {
		if closeErr := p.expressWriter.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

func (p *Producer) PublishStandard(ctx context.Context, event *model.SmsSendEvent) error {
	return p.publish(ctx, p.standardWriter, event, "standard")
}

func (p *Producer) PublishExpress(ctx context.Context, event *model.SmsSendEvent) error {
	return p.publish(ctx, p.expressWriter, event, "express")
}

func (p *Producer) publish(ctx context.Context, writer *kafka.Writer, event *model.SmsSendEvent, label string) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(event.CustomerID),
		Value: payload,
		Time:  time.Now().UTC(),
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		return err
	}

	if p.logger != nil {
		p.logger.Info("produced sms event",
			zap.String("mode", label),
			zap.String("event_id", event.EventID),
			zap.String("message_id", event.MessageID),
		)
	}

	return nil
}
