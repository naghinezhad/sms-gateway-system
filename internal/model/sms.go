package model

import "time"

const (
	PriorityStandard = "STANDARD"
	PriorityExpress  = "EXPRESS"

	StatusPending = "PENDING"
	StatusQueued  = "QUEUED"
	StatusSent    = "SENT"
	StatusFailed  = "FAILED"
)

type SmsMessage struct {
	MessageID       string     `json:"messageId"`
	CustomerID      string     `json:"customerId"`
	To              string     `json:"to"`
	Text            string     `json:"text"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status"`
	Cost            int64      `json:"cost"`
	ClientRef       *string    `json:"clientRef,omitempty"`
	SendAttempts    int        `json:"sendAttempts"`
	SLABreached     bool       `json:"slaBreached"`
	ProviderMessage *string    `json:"providerMessageId,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	QueuedAt        *time.Time `json:"queuedAt,omitempty"`
	OperatorSentAt  *time.Time `json:"operatorSentAt,omitempty"`
	FailedAt        *time.Time `json:"failedAt,omitempty"`
	FailReason      *string    `json:"failReason,omitempty"`
}

type SendSmsInput struct {
	To        string `json:"to" binding:"required"`
	Text      string `json:"text" binding:"required"`
	ClientRef string `json:"clientRef"`
}

type SmsSendEvent struct {
	EventID    string    `json:"eventId"`
	MessageID  string    `json:"messageId"`
	CustomerID string    `json:"customerId"`
	To         string    `json:"to"`
	Text       string    `json:"text"`
	Priority   string    `json:"priority"`
	Cost       int64     `json:"cost"`
	CreatedAt  time.Time `json:"createdAt"`
}

type SmsReportFilters struct {
	CustomerID string
	Status     string
	Priority   string
	To         string
	FromTime   *time.Time
	ToTime     *time.Time
	Limit      int
	Offset     int
}
