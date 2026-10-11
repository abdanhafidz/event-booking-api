// Command seed loads the fixed QA dataset (users and fixture events).
//
//	seed               ensure the dataset is in the database; skip if this
//	                   version was already seeded (runs at container startup)
//	seed -export DIR   write users.csv and event.properties for JMeter and the
//	                   smoke test, from the deterministic IDs; no database needed
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	entity "abdanhafidz.com/event-booking-api/models/entity"
	"abdanhafidz.com/event-booking-api/provider"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// seedLockKey serialises seeders that start at the same time (two replicas).
const seedLockKey = 7420001

func main() {
	export := flag.String("export", "", "write users.csv and event.properties to this directory and exit")
	flag.Parse()

	if *export != "" {
		if err := Export(*export); err != nil {
			log.Fatalf("[SEED] export failed: %v", err)
		}
		log.Printf("[SEED] exported dataset %s to %s", DatasetVersion, *export)
		return
	}

	cfg := provider.NewConfigProvider()
	if err := provider.Migrate(cfg); err != nil {
		log.Fatalf("[SEED] migrate failed: %v", err)
	}
	seeded, err := Ensure(context.Background(), cfg.ProvideDatabaseConfig().GetInstance())
	if err != nil {
		log.Fatalf("[SEED] seeding failed: %v", err)
	}
	if seeded {
		log.Printf("[SEED] dataset %s seeded: %d users, %d events", DatasetVersion, UserCount, len(fixtureEvents))
	} else {
		log.Printf("[SEED] dataset %s already seeded, skipping", DatasetVersion)
	}
}

// Ensure loads the dataset unless its version marker exists. It reports
// whether it seeded. The marker is written in the same transaction as the
// data, so a failed seed leaves no marker and is retried on the next start.
func Ensure(ctx context.Context, db *gorm.DB) (bool, error) {
	seeded := false
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", seedLockKey).Error; err != nil {
			return err
		}
		var markers int64
		if err := tx.Model(&entity.SeedMarker{}).Where("version = ?", DatasetVersion).Count(&markers).Error; err != nil {
			return err
		}
		if markers > 0 {
			return nil
		}

		// ON CONFLICT DO NOTHING keeps rows from an older dataset version intact.
		// A fresh chain per insert: a reused *gorm.DB would carry Omit/Clauses over.
		insert := func() *gorm.DB { return tx.Clauses(clause.OnConflict{DoNothing: true}) }
		users := make([]entity.User, 0, UserCount)
		for i := 1; i <= UserCount; i++ {
			users = append(users, entity.User{Id: UserId(i), Name: fmt.Sprintf("user-%d", i)})
		}
		if err := insert().CreateInBatches(&users, 1000).Error; err != nil {
			return err
		}

		for _, f := range fixtureEvents {
			event := entity.Event{Id: EventId(f.Key), Name: f.Name, StartsAt: fixtureStartsAt}
			if err := insert().Omit("TicketTypes").Create(&event).Error; err != nil {
				return err
			}
			for i := 0; i < f.TicketTypes; i++ {
				tt := entity.TicketType{
					Id:        TicketTypeId(f.Key, i),
					EventId:   event.Id,
					Name:      fmt.Sprintf("Tier %d", i+1),
					Price:     fixturePrice,
					Capacity:  f.Capacity,
					Available: f.Capacity,
				}
				if err := insert().Create(&tt).Error; err != nil {
					return err
				}
			}
		}

		seeded = true
		return tx.Create(&entity.SeedMarker{Version: DatasetVersion, SeededAt: time.Now().UTC()}).Error
	})
	return seeded, err
}

// Export writes the files the pipeline needs, using the same IDs Ensure seeds.
func Export(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(dir, "users.csv"))
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	for i := 1; i <= UserCount; i++ {
		if err := w.Write([]string{UserId(i).String()}); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	p, err := os.Create(filepath.Join(dir, "event.properties"))
	if err != nil {
		return err
	}
	defer p.Close()
	fmt.Fprintf(p, "dataset_version=%s\n", DatasetVersion)
	for _, ev := range fixtureEvents {
		switch ev.Key {
		case "load":
			fmt.Fprintf(p, "event_id=%s\nticket_type_count=%d\n", EventId(ev.Key), ev.TicketTypes)
			for i := 0; i < ev.TicketTypes; i++ {
				fmt.Fprintf(p, "ticket_type_id_%d=%s\n", i, TicketTypeId(ev.Key, i))
			}
		default:
			fmt.Fprintf(p, "%s_event_id=%s\n%s_ticket_type_id=%s\n%s_tickets=%d\n",
				ev.Key, EventId(ev.Key), ev.Key, TicketTypeId(ev.Key, 0), ev.Key, ev.Capacity)
		}
	}
	return nil
}
