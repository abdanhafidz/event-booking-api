package config

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseConfig interface {
	AutoMigrateAll(entities ...interface{}) error
	GetInstance() *gorm.DB
}
type databaseConfig struct {
	db *gorm.DB
}

func NewDatabaseConfig(host, user, password, name, port, sslMode string, poolSize int) DatabaseConfig {
	// Values are quoted: an empty or space-containing password would otherwise
	// swallow the next key (password= dbname=x makes "dbname=x" the password).
	// TimeZone=UTC so every timestamp the API returns is UTC, as the RFC specifies.
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		quoteDSN(host), quoteDSN(user), quoteDSN(password), quoteDSN(name), quoteDSN(port), quoteDSN(sslMode),
	)
	return NewDatabaseConfigFromDSN(dsn, poolSize)
}

func quoteDSN(v string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + "'"
}

func NewDatabaseConfigFromDSN(dsn string, poolSize int) DatabaseConfig {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	db = db.Session(&gorm.Session{
		PrepareStmt: false,
	})

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database handle:", err)
	}
	sqlDB.SetMaxOpenConns(poolSize)
	sqlDB.SetMaxIdleConns(poolSize)

	return &databaseConfig{db: db}
}

func (cfg *databaseConfig) AutoMigrateAll(entities ...interface{}) error {
	return cfg.db.AutoMigrate(entities...)
}

func (cfg *databaseConfig) GetInstance() *gorm.DB {
	return cfg.db
}
