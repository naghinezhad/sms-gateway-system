package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naghinezhad/sms-gateway-system/internal/model"
	"github.com/naghinezhad/sms-gateway-system/internal/repository"
	"go.uber.org/zap"
)

type EventPublisher interface {
	PublishStandard(ctx context.Context, event *model.SmsSendEvent) error
	PublishExpress(ctx context.Context, event *model.SmsSendEvent) error
}

type SmsService struct {
	db            *pgxpool.Pool
	smsRepo       *repository.SmsRepository
	wallets       *repository.WalletRepository
	customers     *repository.CustomerRepository
	publisher     EventPublisher
	standardPrice int64
	expressPrice  int64
	maxSMSLength  int
	logger        *zap.Logger
}

func NewSmsService(
	db *pgxpool.Pool,
	publisher EventPublisher,
	standardPrice int64,
	expressPrice int64,
	maxSMSLength int,
	logger *zap.Logger,
) *SmsService {
	return &SmsService{
		db:            db,
		smsRepo:       repository.NewSmsRepository(db),
		wallets:       repository.NewWalletRepository(db),
		customers:     repository.NewCustomerRepository(db),
		publisher:     publisher,
		standardPrice: standardPrice,
		expressPrice:  expressPrice,
		maxSMSLength:  maxSMSLength,
		logger:        logger,
	}
}

func (s *SmsService) SendSMS(ctx context.Context, customerID string, input *model.SendSmsInput, priority string) (*model.SmsMessage, error) {
	if input == nil {
		return nil, ErrInvalidMessage
	}

	if s.publisher == nil {
		return nil, errors.New("publisher not configured")
	}

	text := strings.TrimSpace(input.Text)
	if text == "" || strings.TrimSpace(input.To) == "" {
		return nil, ErrInvalidMessage
	}

	if s.maxSMSLength > 0 && utf8.RuneCountInString(text) > s.maxSMSLength {
		return nil, ErrMessageTooLong
	}

	price := s.standardPrice
	if priority == model.PriorityExpress {
		price = s.expressPrice
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	customerRepo := repository.NewCustomerRepository(tx)
	walletRepo := repository.NewWalletRepository(tx)
	smsRepo := repository.NewSmsRepository(tx)

	if _, err := customerRepo.GetByID(ctx, customerID); err != nil {
		if err == repository.ErrCustomerNotFound {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	balance, err := walletRepo.GetForUpdate(ctx, customerID)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}

	if input.ClientRef != "" {
		existing, err := smsRepo.GetByClientRef(ctx, customerID, input.ClientRef)
		if err == nil {
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			return existing, nil
		}
		if !errors.Is(err, repository.ErrMessageNotFound) {
			return nil, err
		}
	}

	if balance < price {
		return nil, ErrInsufficientBalance
	}

	newBalance := balance - price
	if err := walletRepo.UpdateBalance(ctx, customerID, newBalance); err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}

	messageID := uuid.NewString()
	createdAt := time.Now().UTC()
	var clientRef *string
	if input.ClientRef != "" {
		clientRef = &input.ClientRef
	}

	msg := &model.SmsMessage{
		MessageID:    messageID,
		CustomerID:   customerID,
		To:           strings.TrimSpace(input.To),
		Text:         text,
		Priority:     priority,
		Status:       model.StatusPending,
		Cost:         price,
		ClientRef:    clientRef,
		SendAttempts: 0,
		SLABreached:  false,
		CreatedAt:    createdAt,
	}

	if err := smsRepo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	event := &model.SmsSendEvent{
		EventID:    uuid.NewString(),
		MessageID:  msg.MessageID,
		CustomerID: msg.CustomerID,
		To:         msg.To,
		Text:       msg.Text,
		Priority:   msg.Priority,
		Cost:       msg.Cost,
		CreatedAt:  msg.CreatedAt,
	}

	var publishErr error
	if priority == model.PriorityExpress {
		publishErr = s.publisher.PublishExpress(ctx, event)
	} else {
		publishErr = s.publisher.PublishStandard(ctx, event)
	}

	if publishErr != nil {
		if s.logger != nil {
			s.logger.Error("failed to publish sms event", zap.Error(publishErr))
		}
		_ = s.failAndRefund(ctx, msg, "publish failed: "+publishErr.Error())
		return nil, publishErr
	}

	if err := s.smsRepo.UpdateStatusQueued(ctx, msg.MessageID, time.Now().UTC()); err != nil && s.logger != nil {
		s.logger.Warn("failed to update queued status", zap.Error(err))
	}

	return msg, nil
}

func (s *SmsService) GetMessage(ctx context.Context, customerID string, messageID string) (*model.SmsMessage, error) {
	msg, err := s.smsRepo.GetByID(ctx, messageID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	if msg.CustomerID != customerID {
		return nil, ErrMessageNotFound
	}

	return msg, nil
}

func (s *SmsService) ListMessages(ctx context.Context, filters *model.SmsReportFilters) ([]model.SmsMessage, error) {
	return s.smsRepo.List(ctx, filters)
}

func (s *SmsService) failAndRefund(ctx context.Context, msg *model.SmsMessage, reason string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	smsRepo := repository.NewSmsRepository(tx)
	walletRepo := repository.NewWalletRepository(tx)

	_ = smsRepo.MarkFailed(ctx, msg.MessageID, time.Now().UTC(), reason, 0)
	_, _ = walletRepo.AddBalance(ctx, msg.CustomerID, msg.Cost)

	return tx.Commit(ctx)
}
