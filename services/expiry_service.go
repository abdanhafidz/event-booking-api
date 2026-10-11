package services

import (
	"context"
	"log/slog"
	"time"

	"abdanhafidz.com/event-booking-api/repositories"
	"github.com/google/uuid"
)

const sweepBatchSize = 200

// ExpiryService releases tickets held by unpaid bookings once their hold has
// passed (BR-04).
type ExpiryService interface {
	SweepOnce(ctx context.Context) (int, error)
	Start(ctx context.Context, interval time.Duration)
}

type expiryService struct {
	transactor     repositories.Transactor
	bookingRepo    repositories.BookingRepository
	ticketTypeRepo repositories.TicketTypeRepository
}

func NewExpiryService(
	transactor repositories.Transactor,
	bookingRepo repositories.BookingRepository,
	ticketTypeRepo repositories.TicketTypeRepository,
) ExpiryService {
	return &expiryService{transactor: transactor, bookingRepo: bookingRepo, ticketTypeRepo: ticketTypeRepo}
}

// SweepOnce expires overdue bookings in batches until none are left, and
// returns how many it expired.
func (s *expiryService) SweepOnce(ctx context.Context) (int, error) {
	total := 0
	for {
		var batch int
		err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
			expired, err := s.bookingRepo.ExpireDue(ctx, sweepBatchSize)
			if err != nil {
				return err
			}
			batch = len(expired)
			release := make(map[uuid.UUID]int)
			for _, b := range expired {
				release[b.TicketTypeId] += b.Quantity
			}
			return s.ticketTypeRepo.Release(ctx, release)
		})
		if err != nil {
			return total, err
		}
		total += batch
		if batch < sweepBatchSize {
			return total, nil
		}
	}
}

func (s *expiryService) Start(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := s.SweepOnce(ctx)
				if err != nil {
					slog.Error("expiry sweep failed", "error", err.Error())
				} else if n > 0 {
					slog.Info("expired bookings", "count", n)
				}
			}
		}
	}()
}
