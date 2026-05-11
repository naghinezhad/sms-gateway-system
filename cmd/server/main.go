package main

import (
	"context"
	"flag"

	"github.com/naghinezhad/sms-gateway-system/config"
	"github.com/naghinezhad/sms-gateway-system/internal/api"
	"github.com/naghinezhad/sms-gateway-system/internal/database"
	"github.com/naghinezhad/sms-gateway-system/internal/kafka"
	"github.com/naghinezhad/sms-gateway-system/internal/service"
	"github.com/naghinezhad/sms-gateway-system/internal/utils/logger"
	"go.uber.org/zap"
)

func main() {
	runMigration := flag.Bool("migration", false, "run database migrations and exit")
	flag.Parse()

	cfg := config.LoadConfig()
	logger.Init()

	ctx := context.Background()

	if *runMigration {
		if err := database.RunMigrations(ctx, cfg.PostgresDSN); err != nil {
			logger.Log.Fatal("failed to run migrations", zap.Error(err))
		}
		logger.Log.Info("migrations applied")
		return
	}

	db, err := database.NewPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		logger.Log.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer db.Close()

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopicStandard, cfg.KafkaTopicExpress, logger.Log)
	defer producer.Close()

	smsService := service.NewSmsService(
		db,
		producer,
		cfg.SMSPrice,
		cfg.SMSExpressPrice,
		cfg.MaxSMSLength,
		logger.Log,
	)

	walletService := service.NewWalletService(db)
	customerService := service.NewCustomerService(db)

	router := api.SetupRouter(smsService, walletService, customerService, logger.Log)

	logger.Log.Info("http server started", zap.String("port", cfg.ServerPort))
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		logger.Log.Fatal("server error", zap.Error(err))
	}
}
