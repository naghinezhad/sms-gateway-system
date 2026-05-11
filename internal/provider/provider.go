package provider

import (
	"context"

	"github.com/naghinezhad/sms-gateway-system/internal/model"
)

type Provider interface {
	Send(ctx context.Context, event *model.SmsSendEvent) (string, error)
}
