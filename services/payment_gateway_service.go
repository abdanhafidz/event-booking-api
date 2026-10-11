package services

import (
	"context"
	"time"

	entity "abdanhafidz.com/event-booking-api/models/entity"
)

// PaymentGatewayService is the only seam tests vary: latency and outcome. The
// implementation is a mock; the token picks the result.
type PaymentGatewayService interface {
	Charge(ctx context.Context, amount int64, token string) (string, error)
}

type mockPaymentGateway struct {
	latency time.Duration
	timeout time.Duration
}

func NewMockPaymentGatewayService(latency, timeout time.Duration) PaymentGatewayService {
	return &mockPaymentGateway{latency: latency, timeout: timeout}
}

// Charge returns SUCCEEDED, DECLINED or TIMEOUT. tok_timeout waits past the
// client timeout, so the caller sees a timeout and no charge is recorded.
func (g *mockPaymentGateway) Charge(ctx context.Context, amount int64, token string) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	delay := g.latency
	if token == entity.PaymentTokenTimeout {
		delay = g.timeout + time.Second
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-callCtx.Done():
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return entity.PaymentStatusTimeout, nil
	}

	if token == entity.PaymentTokenSuccess {
		return entity.PaymentStatusSucceeded, nil
	}
	return entity.PaymentStatusDeclined, nil
}
