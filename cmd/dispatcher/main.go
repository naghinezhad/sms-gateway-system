package main

import (
	"context"
	"errors"
	"time"

	"github.com/naghinezhad/sms-gateway-system/config"
	"github.com/naghinezhad/sms-gateway-system/internal/database"
	"github.com/naghinezhad/sms-gateway-system/internal/kafka"
	"github.com/naghinezhad/sms-gateway-system/internal/provider"
	"github.com/naghinezhad/sms-gateway-system/internal/redis"
	"github.com/naghinezhad/sms-gateway-system/internal/repository"
	"github.com/naghinezhad/sms-gateway-system/internal/service"
	"github.com/naghinezhad/sms-gateway-system/internal/utils/logger"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadConfig()
	logger.Init()

	ctx := context.Background()

	db, err := database.NewPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		logger.Log.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer db.Close()

	redisClient, err := redis.NewRedis(ctx, cfg.RedisAddr, cfg.RedisPass)
	if err != nil {
		logger.Log.Fatal("failed to connect to redis", zap.Error(err))
	}

	providerClient := provider.NewSimulatedProvider(
		cfg.ProviderMinLatencyMs,
		cfg.ProviderMaxLatencyMs,
		cfg.ProviderFailureRate,
	)

	dispatcher := service.NewDispatcherService(
		repository.NewSmsRepository(db),
		redisClient,
		providerClient,
		time.Duration(cfg.LockTTLSeconds)*time.Second,
		cfg.DispatchMaxRetries,
		time.Duration(cfg.DispatchRetryBackoffMs)*time.Millisecond,
		time.Duration(cfg.ExpressSLAMs)*time.Millisecond,
		logger.Log,
	)

	mode := cfg.DispatcherMode
	switch mode {
	case "standard":
		runConsumer(ctx, cfg, dispatcher, "standard")
	case "express":
		runConsumer(ctx, cfg, dispatcher, "express")
	case "all":
		errCh := make(chan error, 2)
		go func() { errCh <- runConsumer(ctx, cfg, dispatcher, "standard") }()
		go func() { errCh <- runConsumer(ctx, cfg, dispatcher, "express") }()
		err := <-errCh
		logger.Log.Fatal("dispatcher error", zap.Error(err))
	default:
		logger.Log.Fatal("invalid DISPATCHER_MODE", zap.String("mode", mode))
	}
}

func runConsumer(ctx context.Context, cfg *config.Config, dispatcher *service.DispatcherService, mode string) error {
	var topic string
	var dltTopic string
	var groupID string
	var consumerID string

	switch mode {
	case "express":
		topic = cfg.KafkaTopicExpress
		dltTopic = cfg.KafkaDLTExpress
		groupID = cfg.ConsumerGroupID + "-express"
		consumerID = cfg.ConsumerID + "-express"
	case "standard":
		topic = cfg.KafkaTopicStandard
		dltTopic = cfg.KafkaDLTStandard
		groupID = cfg.ConsumerGroupID + "-standard"
		consumerID = cfg.ConsumerID + "-standard"
	default:
		return errors.New("invalid mode")
	}

	dltWriter := &kafkago.Writer{
		Addr:         kafkago.TCP(cfg.KafkaBrokers...),
		Topic:        dltTopic,
		RequiredAcks: kafkago.RequireOne,
	}
	defer dltWriter.Close()

	consumer := kafka.NewConsumer(
		cfg.KafkaBrokers,
		topic,
		groupID,
		consumerID,
		logger.Log,
		dispatcher.ProcessEvent,
		dltWriter,
	)
	defer consumer.Close()

	logger.Log.Info("dispatcher started", zap.String("mode", mode), zap.String("consumer_id", consumerID))

	if err := consumer.Run(ctx); err != nil {
		return err
	}

	return nil
}
