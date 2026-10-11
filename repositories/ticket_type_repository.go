package repositories

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TicketTypeForSale is a ticket type with the event fields a booking needs,
// evaluated against the database clock.
type TicketTypeForSale struct {
	Id      uuid.UUID
	EventId uuid.UUID
	Price   int64
	OnSale  bool
}

type TicketTypeRepository interface {
	GetForSale(ctx context.Context, id uuid.UUID) (TicketTypeForSale, error)
	Reserve(ctx context.Context, id uuid.UUID, quantity int) (bool, error)
	GetAvailable(ctx context.Context, id uuid.UUID) (int, error)
	Release(ctx context.Context, quantities map[uuid.UUID]int) error
}

type ticketTypeRepository struct {
	db *gorm.DB
}

func NewTicketTypeRepository(db *gorm.DB) TicketTypeRepository {
	return &ticketTypeRepository{db: db}
}

func (r *ticketTypeRepository) GetForSale(ctx context.Context, id uuid.UUID) (TicketTypeForSale, error) {
	var rows []TicketTypeForSale
	err := conn(ctx, r.db).Raw(`
		SELECT t.id, t.event_id, t.price, e.starts_at > now() AS on_sale
		FROM ticket_types t JOIN events e ON e.id = t.event_id
		WHERE t.id = ?`, id).Scan(&rows).Error
	if err != nil {
		return TicketTypeForSale{}, err
	}
	if len(rows) == 0 {
		return TicketTypeForSale{}, gorm.ErrRecordNotFound
	}
	return rows[0], nil
}

// Reserve is the overselling guard (BR-03). Concurrent reservations of one
// ticket type queue on its row lock, and each re-checks available >= quantity
// against the committed value, so the last ticket cannot be sold twice.
func (r *ticketTypeRepository) Reserve(ctx context.Context, id uuid.UUID, quantity int) (bool, error) {
	res := conn(ctx, r.db).Exec(
		`UPDATE ticket_types SET available = available - ? WHERE id = ? AND available >= ?`,
		quantity, id, quantity,
	)
	return res.RowsAffected == 1, res.Error
}

func (r *ticketTypeRepository) GetAvailable(ctx context.Context, id uuid.UUID) (int, error) {
	var available int
	err := conn(ctx, r.db).Raw(`SELECT available FROM ticket_types WHERE id = ?`, id).Scan(&available).Error
	return available, err
}

// Release returns stock. Ticket types are updated in ascending ID order so two
// transactions releasing several types never deadlock.
func (r *ticketTypeRepository) Release(ctx context.Context, quantities map[uuid.UUID]int) error {
	ids := make([]uuid.UUID, 0, len(quantities))
	for id := range quantities {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })

	for _, id := range ids {
		err := conn(ctx, r.db).Exec(
			`UPDATE ticket_types SET available = available + ? WHERE id = ?`,
			quantities[id], id,
		).Error
		if err != nil {
			return err
		}
	}
	return nil
}
