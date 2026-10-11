package config

import "time"

// BookingConfig holds the booking rules that tests need to shorten (hold time,
// sweep interval) and the mock payment behaviour.
type BookingConfig interface {
	GetHoldTTL() time.Duration
	GetSweepInterval() time.Duration
	IsSweeperEnabled() bool
	GetPaymentTimeout() time.Duration
	GetMockPayLatency() time.Duration
	GetIdempotencyTTL() time.Duration
}

type bookingConfig struct {
	holdTTL        time.Duration
	sweepInterval  time.Duration
	sweeperEnabled bool
	paymentTimeout time.Duration
	mockPayLatency time.Duration
	idempotencyTTL time.Duration
}

func NewBookingConfig(holdTTL, sweepInterval time.Duration, sweeperEnabled bool, paymentTimeout, mockPayLatency, idempotencyTTL time.Duration) BookingConfig {
	return &bookingConfig{
		holdTTL:        holdTTL,
		sweepInterval:  sweepInterval,
		sweeperEnabled: sweeperEnabled,
		paymentTimeout: paymentTimeout,
		mockPayLatency: mockPayLatency,
		idempotencyTTL: idempotencyTTL,
	}
}

func (c *bookingConfig) GetHoldTTL() time.Duration        { return c.holdTTL }
func (c *bookingConfig) GetSweepInterval() time.Duration  { return c.sweepInterval }
func (c *bookingConfig) IsSweeperEnabled() bool           { return c.sweeperEnabled }
func (c *bookingConfig) GetPaymentTimeout() time.Duration { return c.paymentTimeout }
func (c *bookingConfig) GetMockPayLatency() time.Duration { return c.mockPayLatency }
func (c *bookingConfig) GetIdempotencyTTL() time.Duration { return c.idempotencyTTL }
