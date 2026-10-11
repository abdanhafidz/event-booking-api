package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (User) TableName() string { return "users" }

type Event struct {
	Id          uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string       `gorm:"not null" json:"name"`
	StartsAt    time.Time    `gorm:"not null;index" json:"starts_at"`
	CreatedAt   time.Time    `json:"created_at"`
	TicketTypes []TicketType `gorm:"foreignKey:EventId" json:"ticket_types,omitempty"`
}

func (Event) TableName() string { return "events" }

// TicketType holds the stock counter. The CHECK on available is the last line
// of defence against overselling (BR-10); the service never relies on it alone.
type TicketType struct {
	Id        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EventId   uuid.UUID `gorm:"type:uuid;not null;index" json:"event_id"`
	Name      string    `gorm:"not null" json:"name"`
	Price     int64     `gorm:"not null;check:chk_ticket_types_price,price >= 0" json:"price"`
	Capacity  int       `gorm:"not null;check:chk_ticket_types_capacity,capacity > 0" json:"capacity"`
	Available int       `gorm:"not null;check:chk_ticket_types_available,available BETWEEN 0 AND capacity" json:"available"`
}

func (TicketType) TableName() string { return "ticket_types" }

type Booking struct {
	Id           uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserId       uuid.UUID   `gorm:"type:uuid;not null;index" json:"user_id"`
	EventId      uuid.UUID   `gorm:"type:uuid;not null" json:"event_id"`
	TicketTypeId uuid.UUID   `gorm:"type:uuid;not null;index" json:"ticket_type_id"`
	Quantity     int         `gorm:"type:smallint;not null;check:chk_bookings_quantity,quantity BETWEEN 1 AND 6" json:"quantity"`
	UnitPrice    int64       `gorm:"not null" json:"unit_price"`
	TotalAmount  int64       `gorm:"not null;check:chk_bookings_total,total_amount = unit_price * quantity" json:"total_amount"`
	Currency     string      `gorm:"type:char(3);not null;default:'IDR'" json:"currency"`
	Status       string      `gorm:"type:text;not null;check:chk_bookings_status,status IN ('PENDING_PAYMENT','CONFIRMED','PAYMENT_FAILED','EXPIRED','CANCELLED')" json:"status"`
	ExpiresAt    time.Time   `gorm:"not null;index:idx_bookings_to_expire,where:status = 'PENDING_PAYMENT'" json:"expires_at"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	User         *User       `gorm:"foreignKey:UserId" json:"-"`
	Event        *Event      `gorm:"foreignKey:EventId" json:"-"`
	TicketType   *TicketType `gorm:"foreignKey:TicketTypeId" json:"-"`
}

func (Booking) TableName() string { return "bookings" }

// Payment records every mock charge attempt. The partial unique index makes a
// second successful payment for one booking impossible (BR-07).
type Payment struct {
	Id        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BookingId uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:one_success_per_booking,where:status = 'SUCCEEDED'" json:"booking_id"`
	Amount    int64     `gorm:"not null" json:"amount"`
	Status    string    `gorm:"type:text;not null;check:chk_payments_status,status IN ('SUCCEEDED','DECLINED','TIMEOUT')" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Booking   *Booking  `gorm:"foreignKey:BookingId" json:"-"`
}

func (Payment) TableName() string { return "payments" }

// IdempotencyKey stores the outcome of a create or pay request so a retry with
// the same key replays it instead of acting again (BR-09).
type IdempotencyKey struct {
	UserId       uuid.UUID `gorm:"type:uuid;primary_key" json:"user_id"`
	Key          string    `gorm:"primary_key" json:"key"`
	RequestHash  string    `gorm:"not null" json:"request_hash"`
	ResponseCode string    `gorm:"not null;default:''" json:"response_code"`
	ResponseBody []byte    `gorm:"type:jsonb" json:"response_body"`
	ExpiresAt    time.Time `gorm:"not null" json:"expires_at"`
}

func (IdempotencyKey) TableName() string { return "idempotency_keys" }

func AllEntities() []interface{} {
	return []interface{}{
		&User{},
		&Event{},
		&TicketType{},
		&Booking{},
		&Payment{},
		&IdempotencyKey{},
		&SeedMarker{},
	}
}

// SeedMarker records that a version of the fixed QA dataset has been loaded,
// so the startup seeder can skip instead of re-seeding.
type SeedMarker struct {
	Version  string    `gorm:"primary_key" json:"version"`
	SeededAt time.Time `gorm:"not null" json:"seeded_at"`
}

func (SeedMarker) TableName() string { return "seed_markers" }
