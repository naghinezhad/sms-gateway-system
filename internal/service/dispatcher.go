package service

import (
	"context"
	"fmt"
	"time"

	"github.com/naghinezhad/sms-gateway-system/internal/model"
	"github.com/naghinezhad/sms-gateway-system/internal/provider"
	"github.com/naghinezhad/sms-gateway-system/internal/redis"
	"github.com/naghinezhad/sms-gateway-system/internal/repository"
	"go.uber.org/zap"
)

type DispatcherService struct {
	smsRepo      *repository.SmsRepository
	redis        redis.Client
	provider     provider.Provider
	lockTTL      time.Duration
	maxRetries   int
	retryBackoff time.Duration
	expressSLA   time.Duration
	logger       *zap.Logger
}

func NewDispatcherService(
	smsRepo *repository.SmsRepository,
	redisClient redis.Client,
	providerClient provider.Provider,
	lockTTL time.Duration,
	maxRetries int,
	retryBackoff time.Duration,
	expressSLA time.Duration,
	logger *zap.Logger,
) *DispatcherService {
	return &DispatcherService{
		smsRepo:      smsRepo,
		redis:        redisClient,
		provider:     providerClient,
		lockTTL:      lockTTL,
		maxRetries:   maxRetries,
		retryBackoff: retryBackoff,
		expressSLA:   expressSLA,
		logger:       logger,
	}
}

func (s *DispatcherService) ProcessEvent(ctx context.Context, event *model.SmsSendEvent) error {
	lockKey := fmt.Sprintf("lock:sms:%s", event.MessageID)
	lockToken, err := s.redis.AcquireLock(ctx, lockKey, s.lockTTL)
	if err != nil {
		return err
	}
	if lockToken == "" {
		return ErrLockNotAcquired
	}

	defer func() {
		_, _ = s.redis.ReleaseLock(ctx, lockKey, lockToken)
	}()

	existing, err := s.smsRepo.GetByID(ctx, event.MessageID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			return ErrMessageNotFound
		}
		return err
	}

	if existing.Status == model.StatusSent || existing.Status == model.StatusFailed {
		return ErrDuplicateEvent
	}

	attempts := 0
	var providerMessageID string
	var sendErr error

	for attempts < s.maxRetries {
		attempts++
		providerMessageID, sendErr = s.provider.Send(ctx, event)
		if sendErr == nil {
			break
		}
		if s.logger != nil {
			s.logger.Warn("provider send failed", zap.Error(sendErr))
		}
		if attempts < s.maxRetries && s.retryBackoff > 0 {
			time.Sleep(s.retryBackoff * time.Duration(attempts))
		}
	}

	if sendErr != nil {
		_ = s.smsRepo.MarkFailed(ctx, event.MessageID, time.Now().UTC(), sendErr.Error(), attempts)
		return sendErr
	}

	sentAt := time.Now().UTC()
	slaBreached := false
	if event.Priority == model.PriorityExpress && s.expressSLA > 0 {
		if sentAt.Sub(event.CreatedAt) > s.expressSLA {
			slaBreached = true
		}
	}

	return s.smsRepo.MarkSent(ctx, event.MessageID, providerMessageID, sentAt, slaBreached, attempts)
}
