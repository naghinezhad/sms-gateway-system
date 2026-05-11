package provider

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/naghinezhad/sms-gateway-system/internal/model"
)

type SimulatedProvider struct {
	minLatency  time.Duration
	maxLatency  time.Duration
	failureRate float64
	rng         *rand.Rand
}

func NewSimulatedProvider(minLatencyMs int, maxLatencyMs int, failureRate float64) *SimulatedProvider {
	min := time.Duration(minLatencyMs) * time.Millisecond
	max := time.Duration(maxLatencyMs) * time.Millisecond
	if max < min {
		max = min
	}

	src := rand.NewSource(time.Now().UnixNano())
	return &SimulatedProvider{
		minLatency:  min,
		maxLatency:  max,
		failureRate: failureRate,
		rng:         rand.New(src),
	}
}

func (p *SimulatedProvider) Send(ctx context.Context, _ *model.SmsSendEvent) (string, error) {
	latency := p.minLatency
	if p.maxLatency > p.minLatency {
		jitter := p.rng.Int63n(int64(p.maxLatency - p.minLatency))
		latency = p.minLatency + time.Duration(jitter)
	}

	timer := time.NewTimer(latency)
	select {
	case <-ctx.Done():
		timer.Stop()
		return "", ctx.Err()
	case <-timer.C:
	}

	if p.rng.Float64() < p.failureRate {
		return "", errors.New("operator send failed")
	}

	return "op-" + uuid.NewString(), nil
}
