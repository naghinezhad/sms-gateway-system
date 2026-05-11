package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/naghinezhad/sms-gateway-system/internal/model"
)

var ErrMessageNotFound = errors.New("message not found")

type SmsRepository struct {
	db DBTX
}

func NewSmsRepository(db DBTX) *SmsRepository {
	return &SmsRepository{db: db}
}

func (r *SmsRepository) CreateMessage(ctx context.Context, msg *model.SmsMessage) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sms_messages (
			message_id, customer_id, to_number, text, priority, status, cost,
			client_ref, send_attempts, sla_breached, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, msg.MessageID, msg.CustomerID, msg.To, msg.Text, msg.Priority, msg.Status, msg.Cost,
		msg.ClientRef, msg.SendAttempts, msg.SLABreached, msg.CreatedAt)
	return err
}

func (r *SmsRepository) GetByID(ctx context.Context, messageID string) (*model.SmsMessage, error) {
	return r.getSingle(ctx, `
		SELECT message_id, customer_id, to_number, text, priority, status, cost,
			client_ref, send_attempts, sla_breached, provider_message_id,
			created_at, queued_at, operator_sent_at, failed_at, fail_reason
		FROM sms_messages
		WHERE message_id = $1
	`, messageID)
}

func (r *SmsRepository) GetByClientRef(ctx context.Context, customerID string, clientRef string) (*model.SmsMessage, error) {
	return r.getSingle(ctx, `
		SELECT message_id, customer_id, to_number, text, priority, status, cost,
			client_ref, send_attempts, sla_breached, provider_message_id,
			created_at, queued_at, operator_sent_at, failed_at, fail_reason
		FROM sms_messages
		WHERE customer_id = $1 AND client_ref = $2
	`, customerID, clientRef)
}

func (r *SmsRepository) getSingle(ctx context.Context, query string, args ...any) (*model.SmsMessage, error) {
	row := r.db.QueryRow(ctx, query, args...)

	msg := &model.SmsMessage{}
	var clientRef *string
	var providerMessageID *string
	var queuedAt *time.Time
	var operatorSentAt *time.Time
	var failedAt *time.Time
	var failReason *string

	err := row.Scan(
		&msg.MessageID,
		&msg.CustomerID,
		&msg.To,
		&msg.Text,
		&msg.Priority,
		&msg.Status,
		&msg.Cost,
		&clientRef,
		&msg.SendAttempts,
		&msg.SLABreached,
		&providerMessageID,
		&msg.CreatedAt,
		&queuedAt,
		&operatorSentAt,
		&failedAt,
		&failReason,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	msg.ClientRef = clientRef
	msg.ProviderMessage = providerMessageID
	msg.QueuedAt = queuedAt
	msg.OperatorSentAt = operatorSentAt
	msg.FailedAt = failedAt
	msg.FailReason = failReason

	return msg, nil
}

func (r *SmsRepository) List(ctx context.Context, filters *model.SmsReportFilters) ([]model.SmsMessage, error) {
	if filters == nil {
		return nil, errors.New("filters are required")
	}

	args := make([]any, 0, 8)
	args = append(args, filters.CustomerID)
	idx := 2

	var clauses []string
	clauses = append(clauses, "customer_id = $1")

	if filters.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", idx))
		args = append(args, filters.Status)
		idx++
	}

	if filters.Priority != "" {
		clauses = append(clauses, fmt.Sprintf("priority = $%d", idx))
		args = append(args, filters.Priority)
		idx++
	}

	if filters.To != "" {
		clauses = append(clauses, fmt.Sprintf("to_number = $%d", idx))
		args = append(args, filters.To)
		idx++
	}

	if filters.FromTime != nil {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *filters.FromTime)
		idx++
	}

	if filters.ToTime != nil {
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", idx))
		args = append(args, *filters.ToTime)
		idx++
	}

	limit := filters.Limit
	offset := filters.Offset
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	clausesStr := strings.Join(clauses, " AND ")
	query := fmt.Sprintf(`
		SELECT message_id, customer_id, to_number, text, priority, status, cost,
			client_ref, send_attempts, sla_breached, provider_message_id,
			created_at, queued_at, operator_sent_at, failed_at, fail_reason
		FROM sms_messages
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, clausesStr, idx, idx+1)

	args = append(args, limit, offset)

	rowsQ, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rowsQ.Close()

	var results []model.SmsMessage
	for rowsQ.Next() {
		msg := model.SmsMessage{}
		var clientRef *string
		var providerMessageID *string
		var queuedAt *time.Time
		var operatorSentAt *time.Time
		var failedAt *time.Time
		var failReason *string

		if err := rowsQ.Scan(
			&msg.MessageID,
			&msg.CustomerID,
			&msg.To,
			&msg.Text,
			&msg.Priority,
			&msg.Status,
			&msg.Cost,
			&clientRef,
			&msg.SendAttempts,
			&msg.SLABreached,
			&providerMessageID,
			&msg.CreatedAt,
			&queuedAt,
			&operatorSentAt,
			&failedAt,
			&failReason,
		); err != nil {
			return nil, err
		}

		msg.ClientRef = clientRef
		msg.ProviderMessage = providerMessageID
		msg.QueuedAt = queuedAt
		msg.OperatorSentAt = operatorSentAt
		msg.FailedAt = failedAt
		msg.FailReason = failReason

		results = append(results, msg)
	}

	return results, rowsQ.Err()
}

func (r *SmsRepository) UpdateStatusQueued(ctx context.Context, messageID string, queuedAt time.Time) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE sms_messages
		SET status = $1, queued_at = $2
		WHERE message_id = $3
	`, model.StatusQueued, queuedAt, messageID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *SmsRepository) MarkSent(ctx context.Context, messageID string, providerMessageID string, sentAt time.Time, slaBreached bool, attempts int) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE sms_messages
		SET status = $1,
			operator_sent_at = $2,
			provider_message_id = $3,
			sla_breached = $4,
			send_attempts = send_attempts + $5,
			failed_at = NULL,
			fail_reason = NULL
		WHERE message_id = $6
	`, model.StatusSent, sentAt, providerMessageID, slaBreached, attempts, messageID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *SmsRepository) MarkFailed(ctx context.Context, messageID string, failedAt time.Time, reason string, attempts int) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE sms_messages
		SET status = $1,
			failed_at = $2,
			fail_reason = $3,
			send_attempts = send_attempts + $4
		WHERE message_id = $5
	`, model.StatusFailed, failedAt, reason, attempts, messageID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}
