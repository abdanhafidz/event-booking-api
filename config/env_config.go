package config

import (
	"strconv"
	"time"

	"abdanhafidz.com/event-booking-api/utils"
	"github.com/joho/godotenv"
)

type EnvConfig interface {
	GetTCPAddress() string
	GetDatabaseHost() string
	GetDatabasePort() string
	GetDatabaseUser() string
	GetDatabasePassword() string
	GetDatabaseName() string
	GetDatabaseSSLMode() string
	GetDatabasePoolSize() int
	GetHoldTTL() time.Duration
	GetSweepInterval() time.Duration
	IsSweeperEnabled() bool
	GetPaymentTimeout() time.Duration
	GetMockPayLatency() time.Duration
	GetIdempotencyTTL() time.Duration
	GetRequestTimeout() time.Duration
}

type envConfig struct{}

func NewEnvConfig() EnvConfig {
	godotenv.Load()
	return &envConfig{}
}

func (e *envConfig) GetTCPAddress() string {
	return getOr("HOST_ADDRESS", "0.0.0.0") + ":" + getOr("HOST_PORT", "8080")
}

func (e *envConfig) GetDatabaseHost() string     { return getOr("DB_HOST", "localhost") }
func (e *envConfig) GetDatabasePort() string     { return getOr("DB_PORT", "5432") }
func (e *envConfig) GetDatabaseUser() string     { return getOr("DB_USER", "postgres") }
func (e *envConfig) GetDatabasePassword() string { return utils.GetEnv("DB_PASSWORD") }
func (e *envConfig) GetDatabaseName() string     { return getOr("DB_NAME", "postgres") }
func (e *envConfig) GetDatabaseSSLMode() string  { return getOr("DB_SSLMODE", "disable") }

func (e *envConfig) GetDatabasePoolSize() int {
	n, err := strconv.Atoi(getOr("DB_POOL_SIZE", "20"))
	if err != nil || n < 1 {
		return 20
	}
	return n
}

func (e *envConfig) GetHoldTTL() time.Duration { return durationOr("HOLD_TTL", 10*time.Minute) }
func (e *envConfig) GetSweepInterval() time.Duration {
	return durationOr("SWEEP_INTERVAL", 5*time.Second)
}
func (e *envConfig) IsSweeperEnabled() bool { return getOr("SWEEPER_ENABLED", "true") == "true" }
func (e *envConfig) GetPaymentTimeout() time.Duration {
	return durationOr("PAYMENT_TIMEOUT", 2*time.Second)
}
func (e *envConfig) GetIdempotencyTTL() time.Duration {
	return durationOr("IDEMPOTENCY_TTL", 24*time.Hour)
}
func (e *envConfig) GetRequestTimeout() time.Duration {
	return durationOr("REQUEST_TIMEOUT", 4*time.Second)
}

func (e *envConfig) GetMockPayLatency() time.Duration {
	ms, err := strconv.Atoi(getOr("MOCKPAY_LATENCY_MS", "0"))
	if err != nil || ms < 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func getOr(key, fallback string) string {
	if v := utils.GetEnv(key); v != "" {
		return v
	}
	return fallback
}

func durationOr(key string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(utils.GetEnv(key))
	if err != nil {
		return fallback
	}
	return d
}
