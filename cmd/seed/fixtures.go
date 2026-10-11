package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// DatasetVersion identifies the fixture set. Change it when the fixtures below
// change; the startup seeder loads a version only once.
const DatasetVersion = "2026-10-11.v1"

// UserCount covers the largest scenario: PT-05 needs 2,000 distinct users.
const UserCount = 10000

// Every fixture has a deterministic ID (UUIDv5), so the pipeline can derive the
// IDs it needs (seed -export) without connecting to the database.
var namespace = uuid.MustParse("6f0c1b8e-3d2a-4c55-9b7e-1a2b3c4d5e6f")

func fixtureId(name string) uuid.UUID { return uuid.NewSHA1(namespace, []byte(name)) }

func UserId(n int) uuid.UUID { return fixtureId("user/" + strconv.Itoa(n)) }

func EventId(key string) uuid.UUID { return fixtureId("event/" + key) }

func TicketTypeId(key string, i int) uuid.UUID {
	return fixtureId(fmt.Sprintf("event/%s/ticket-type/%d", key, i))
}

type fixtureEvent struct {
	Key         string
	Name        string
	TicketTypes int
	Capacity    int
}

// Fixture events. Stock is reused across runs:
//   - load: 5 × 1,000,000 tickets, enough for many runs of PT-01 to PT-04.
//   - flash: 500 tickets. PT-05 never pays, so its holds expire and the sweeper
//     returns the stock 10 minutes after each run.
//   - smoke: 5 tickets. The smoke test releases everything it books.
var fixtureEvents = []fixtureEvent{
	{Key: "load", Name: "Perf load event", TicketTypes: 5, Capacity: 1_000_000},
	{Key: "flash", Name: "Perf flash sale", TicketTypes: 1, Capacity: 500},
	{Key: "smoke", Name: "Smoke test event", TicketTypes: 1, Capacity: 5},
}

// Far enough ahead that fixture events never stop selling.
var fixtureStartsAt = time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)

const fixturePrice = 150000
