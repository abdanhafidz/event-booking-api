package config

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestDSNQuotingSurvivesEmptyAndSpecialPasswords(t *testing.T) {
	for _, password := range []string{"", "with space", `it's\tricky`} {
		dsn := "host=" + quoteDSN("localhost") + " password=" + quoteDSN(password) + " dbname=" + quoteDSN("booking_test")
		cfg, err := pgx.ParseConfig(dsn)
		if err != nil {
			t.Fatalf("%q: %v", password, err)
		}
		if cfg.Password != password || cfg.Database != "booking_test" {
			t.Fatalf("%q: got password %q, database %q", password, cfg.Password, cfg.Database)
		}
	}
}
