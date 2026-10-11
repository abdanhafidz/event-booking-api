package provider

import (
	"abdanhafidz.com/event-booking-api/config"
)

type ConfigProvider interface {
	ProvideDatabaseConfig() config.DatabaseConfig
	ProvideEnvConfig() config.EnvConfig
	ProvideBookingConfig() config.BookingConfig
}

type configProvider struct {
	databaseConfig config.DatabaseConfig
	envConfig      config.EnvConfig
	bookingConfig  config.BookingConfig
}

func NewConfigProvider() ConfigProvider {
	envConfig := config.NewEnvConfig()
	databaseConfig := config.NewDatabaseConfig(
		envConfig.GetDatabaseHost(), envConfig.GetDatabaseUser(), envConfig.GetDatabasePassword(),
		envConfig.GetDatabaseName(), envConfig.GetDatabasePort(), envConfig.GetDatabaseSSLMode(),
		envConfig.GetDatabasePoolSize(),
	)
	bookingConfig := config.NewBookingConfig(
		envConfig.GetHoldTTL(), envConfig.GetSweepInterval(), envConfig.IsSweeperEnabled(),
		envConfig.GetPaymentTimeout(), envConfig.GetMockPayLatency(), envConfig.GetIdempotencyTTL(),
	)
	return NewConfigProviderWith(envConfig, databaseConfig, bookingConfig)
}

// NewConfigProviderWith lets integration tests supply their own database and
// booking rules (for example a 2-second hold).
func NewConfigProviderWith(envConfig config.EnvConfig, databaseConfig config.DatabaseConfig, bookingConfig config.BookingConfig) ConfigProvider {
	return &configProvider{
		databaseConfig: databaseConfig,
		envConfig:      envConfig,
		bookingConfig:  bookingConfig,
	}
}

func (c *configProvider) ProvideDatabaseConfig() config.DatabaseConfig {
	return c.databaseConfig
}

func (c *configProvider) ProvideEnvConfig() config.EnvConfig {
	return c.envConfig
}

func (c *configProvider) ProvideBookingConfig() config.BookingConfig {
	return c.bookingConfig
}
