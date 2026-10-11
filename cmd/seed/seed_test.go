package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"abdanhafidz.com/event-booking-api/config"
	entity "abdanhafidz.com/event-booking-api/models/entity"
)

func TestExportMatchesSeededIds(t *testing.T) {
	dir := t.TempDir()
	if err := Export(dir); err != nil {
		t.Fatal(err)
	}
	props, _ := os.ReadFile(filepath.Join(dir, "event.properties"))
	for _, want := range []string{
		"event_id=" + EventId("load").String(),
		"ticket_type_id_4=" + TicketTypeId("load", 4).String(),
		"flash_ticket_type_id=" + TicketTypeId("flash", 0).String(),
		"flash_tickets=500",
		"smoke_ticket_type_id=" + TicketTypeId("smoke", 0).String(),
	} {
		if !strings.Contains(string(props), want+"\n") {
			t.Errorf("event.properties missing %q", want)
		}
	}
	users, _ := os.ReadFile(filepath.Join(dir, "users.csv"))
	if lines := strings.Count(string(users), "\n"); lines != UserCount {
		t.Errorf("users.csv: want %d lines, got %d", UserCount, lines)
	}
}

// Needs TEST_DATABASE_URL. The second call must skip and change nothing.
func TestEnsureSeedsOnceThenSkips(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	dbConfig := config.NewDatabaseConfigFromDSN(dsn, 5)
	if err := dbConfig.AutoMigrateAll(entity.AllEntities()...); err != nil {
		t.Fatal(err)
	}
	db := dbConfig.GetInstance()
	db.Exec("DELETE FROM seed_markers WHERE version = ?", DatasetVersion)

	if _, err := Ensure(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var available int
	db.Raw("SELECT available FROM ticket_types WHERE id = ?", TicketTypeId("smoke", 0)).Scan(&available)
	db.Exec("UPDATE ticket_types SET available = available - 1 WHERE id = ?", TicketTypeId("smoke", 0))

	seeded, err := Ensure(context.Background(), db)
	if err != nil || seeded {
		t.Fatalf("second run: want skip, got seeded=%v err=%v", seeded, err)
	}
	var after int
	db.Raw("SELECT available FROM ticket_types WHERE id = ?", TicketTypeId("smoke", 0)).Scan(&after)
	if after != available-1 {
		t.Fatalf("skip must not touch existing data: available %d, want %d", after, available-1)
	}
	db.Exec("UPDATE ticket_types SET available = available + 1 WHERE id = ?", TicketTypeId("smoke", 0))
}
