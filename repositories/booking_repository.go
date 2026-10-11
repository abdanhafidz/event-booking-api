package repositories

import (
	"context"
	"time"

	entity "abdanhafidz.com/event-booking-api/models/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LockedBooking is a booking held under FOR UPDATE, with its time checks
// evaluated against the database clock.
type LockedBooking struct {
	entity.Booking `gorm:"embedded"`
	HoldPassed     bool
	EventStarted   bool
}

type ExpiredBooking struct {
	TicketTypeId uuid.UUID
	Quantity     int
}

type BookingRepository interface {
	Create(ctx context.Context, booking entity.Booking, hold time.Duration) (entity.Booking, error)
	GetForUser(ctx context.Context, id, userId uuid.UUID) (entity.Booking, error)
	LockForUpdate(ctx context.Context, id uuid.UUID) (LockedBooking, error)
	TransitionStatus(ctx context.Context, id uuid.UUID, from, to string) (entity.Booking, bool, error)
	ExpireDue(ctx context.Context, limit int) ([]ExpiredBooking, error)
}

type bookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) Create(ctx context.Context, b entity.Booking, hold time.Duration) (entity.Booking, error) {
	var rows []entity.Booking
	err := conn(ctx, r.db).Raw(`
		INSERT INTO bookings (id, user_id, event_id, ticket_type_id, quantity, unit_price, total_amount,
		                      currency, status, expires_at, created_at, updated_at)
		VALUES (gen_random_uuid(), ?, ?, ?, ?, ?, ?, ?, ?, now() + make_interval(secs => ?), now(), now())
		RETURNING *`,
		b.UserId, b.EventId, b.TicketTypeId, b.Quantity, b.UnitPrice, b.TotalAmount,
		b.Currency, entity.BookingStatusPendingPayment, hold.Seconds(),
	).Scan(&rows).Error
	if err != nil {
		return entity.Booking{}, err
	}
	return rows[0], nil
}

func (r *bookingRepository) GetForUser(ctx context.Context, id, userId uuid.UUID) (entity.Booking, error) {
	var b entity.Booking
	err := conn(ctx, r.db).First(&b, "id = ? AND user_id = ?", id, userId).Error
	return b, err
}

func (r *bookingRepository) LockForUpdate(ctx context.Context, id uuid.UUID) (LockedBooking, error) {
	var rows []LockedBooking
	err := conn(ctx, r.db).Raw(`
		SELECT b.*, b.expires_at <= now() AS hold_passed, e.starts_at <= now() AS event_started
		FROM bookings b JOIN events e ON e.id = b.event_id
		WHERE b.id = ?
		FOR UPDATE OF b`, id).Scan(&rows).Error
	if err != nil {
		return LockedBooking{}, err
	}
	if len(rows) == 0 {
		return LockedBooking{}, gorm.ErrRecordNotFound
	}
	return rows[0], nil
}

// TransitionStatus is a compare-and-set: it only moves a booking that is still
// in the expected status, and reports false if another request got there first.
func (r *bookingRepository) TransitionStatus(ctx context.Context, id uuid.UUID, from, to string) (entity.Booking, bool, error) {
	var rows []entity.Booking
	err := conn(ctx, r.db).Raw(`
		UPDATE bookings SET status = ?, updated_at = now()
		WHERE id = ? AND status = ?
		RETURNING *`, to, id, from).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return entity.Booking{}, false, err
	}
	return rows[0], true, nil
}

// ExpireDue moves up to limit overdue PENDING_PAYMENT bookings to EXPIRED.
// SKIP LOCKED leaves alone bookings that a pay or cancel request is holding,
// and makes the sweeper safe to run on several instances.
func (r *bookingRepository) ExpireDue(ctx context.Context, limit int) ([]ExpiredBooking, error) {
	var rows []ExpiredBooking
	err := conn(ctx, r.db).Raw(`
		WITH due AS (
			SELECT id FROM bookings
			WHERE status = ? AND expires_at <= now()
			ORDER BY expires_at
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		UPDATE bookings b SET status = ?, updated_at = now()
		FROM due WHERE b.id = due.id
		RETURNING b.ticket_type_id, b.quantity`,
		entity.BookingStatusPendingPayment, limit, entity.BookingStatusExpired,
	).Scan(&rows).Error
	return rows, err
}
