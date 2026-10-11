package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"abdanhafidz.com/event-booking-api/config"
	dto "abdanhafidz.com/event-booking-api/models/dto"
	entity "abdanhafidz.com/event-booking-api/models/entity"
	http_error "abdanhafidz.com/event-booking-api/models/error"
	"abdanhafidz.com/event-booking-api/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookingService interface {
	// Create and Pay also report whether the result was replayed from an
	// earlier request with the same Idempotency-Key.
	Create(ctx context.Context, userId uuid.UUID, idempotencyKey string, req dto.CreateBookingRequest) (dto.BookingResponse, bool, error)
	Get(ctx context.Context, userId, bookingId uuid.UUID) (dto.BookingResponse, error)
	Pay(ctx context.Context, userId, bookingId uuid.UUID, idempotencyKey string, req dto.PayBookingRequest) (dto.BookingResponse, bool, error)
	Cancel(ctx context.Context, userId, bookingId uuid.UUID) (dto.BookingResponse, error)
}

type bookingService struct {
	transactor      repositories.Transactor
	bookingRepo     repositories.BookingRepository
	ticketTypeRepo  repositories.TicketTypeRepository
	paymentRepo     repositories.PaymentRepository
	idempotencyRepo repositories.IdempotencyRepository
	paymentGateway  PaymentGatewayService
	bookingConfig   config.BookingConfig
}

func NewBookingService(
	transactor repositories.Transactor,
	bookingRepo repositories.BookingRepository,
	ticketTypeRepo repositories.TicketTypeRepository,
	paymentRepo repositories.PaymentRepository,
	idempotencyRepo repositories.IdempotencyRepository,
	paymentGateway PaymentGatewayService,
	bookingConfig config.BookingConfig,
) BookingService {
	return &bookingService{
		transactor:      transactor,
		bookingRepo:     bookingRepo,
		ticketTypeRepo:  ticketTypeRepo,
		paymentRepo:     paymentRepo,
		idempotencyRepo: idempotencyRepo,
		paymentGateway:  paymentGateway,
		bookingConfig:   bookingConfig,
	}
}

func (s *bookingService) Create(ctx context.Context, userId uuid.UUID, key string, req dto.CreateBookingRequest) (dto.BookingResponse, bool, error) {
	return s.idempotent(ctx, userId, key, "POST /v1/bookings", req, func(ctx context.Context) (dto.BookingResponse, error) {
		tt, err := s.ticketTypeRepo.GetForSale(ctx, req.TicketTypeId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.BookingResponse{}, http_error.TICKET_TYPE_NOT_FOUND
		}
		if err != nil {
			return dto.BookingResponse{}, err
		}
		if !tt.OnSale {
			return dto.BookingResponse{}, http_error.SALES_CLOSED
		}

		reserved, err := s.ticketTypeRepo.Reserve(ctx, tt.Id, req.Quantity)
		if err != nil {
			return dto.BookingResponse{}, err
		}
		if !reserved {
			available, err := s.ticketTypeRepo.GetAvailable(ctx, tt.Id)
			if err != nil {
				return dto.BookingResponse{}, err
			}
			return dto.BookingResponse{}, http_error.INSUFFICIENT_TICKETS.WithDetails(map[string]any{"available": available})
		}

		booking, err := s.bookingRepo.Create(ctx, entity.Booking{
			UserId:       userId,
			EventId:      tt.EventId,
			TicketTypeId: tt.Id,
			Quantity:     req.Quantity,
			UnitPrice:    tt.Price,
			TotalAmount:  tt.Price * int64(req.Quantity),
			Currency:     entity.DefaultCurrency,
		}, s.bookingConfig.GetHoldTTL())
		if err != nil {
			return dto.BookingResponse{}, err
		}
		return dto.NewBookingResponse(booking), nil
	})
}

func (s *bookingService) Get(ctx context.Context, userId, bookingId uuid.UUID) (dto.BookingResponse, error) {
	booking, err := s.bookingRepo.GetForUser(ctx, bookingId, userId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.BookingResponse{}, http_error.BOOKING_NOT_FOUND
	}
	if err != nil {
		return dto.BookingResponse{}, err
	}
	return dto.NewBookingResponse(booking), nil
}

// Pay runs in one transaction with the booking row locked, so a second pay or
// a cancel for the same booking waits and then sees the new status. The mock
// is called inside the transaction; that is safe only because the mock never
// charges on timeout (see the RFC trade-offs).
func (s *bookingService) Pay(ctx context.Context, userId, bookingId uuid.UUID, key string, req dto.PayBookingRequest) (dto.BookingResponse, bool, error) {
	route := "POST /v1/bookings/" + bookingId.String() + "/pay"
	return s.idempotent(ctx, userId, key, route, req, func(ctx context.Context) (dto.BookingResponse, error) {
		locked, err := s.lockOwned(ctx, userId, bookingId)
		if err != nil {
			return dto.BookingResponse{}, err
		}

		if locked.Status == entity.BookingStatusExpired {
			return dto.BookingResponse{}, http_error.BOOKING_EXPIRED
		}
		if locked.Status == entity.BookingStatusPendingPayment && locked.HoldPassed {
			if err := s.transitionAndRelease(ctx, locked.Booking, entity.BookingStatusExpired); err != nil {
				return dto.BookingResponse{}, err
			}
			return dto.BookingResponse{}, http_error.BOOKING_EXPIRED
		}
		if locked.Status != entity.BookingStatusPendingPayment {
			return dto.BookingResponse{}, http_error.BOOKING_NOT_PAYABLE.WithDetails(map[string]any{"status": locked.Status})
		}

		result, err := s.paymentGateway.Charge(ctx, locked.TotalAmount, req.PaymentToken)
		if err != nil {
			return dto.BookingResponse{}, err
		}
		if _, err := s.paymentRepo.Create(ctx, entity.Payment{
			BookingId: locked.Id,
			Amount:    locked.TotalAmount,
			Status:    result,
		}); err != nil {
			return dto.BookingResponse{}, err
		}

		if result == entity.PaymentStatusSucceeded {
			confirmed, ok, err := s.bookingRepo.TransitionStatus(ctx, locked.Id, entity.BookingStatusPendingPayment, entity.BookingStatusConfirmed)
			if err != nil {
				return dto.BookingResponse{}, err
			}
			if !ok {
				return dto.BookingResponse{}, http_error.INTERNAL_SERVER_ERROR
			}
			return dto.NewBookingResponse(confirmed), nil
		}

		if err := s.transitionAndRelease(ctx, locked.Booking, entity.BookingStatusPaymentFailed); err != nil {
			return dto.BookingResponse{}, err
		}
		details := map[string]any{"booking_status": entity.BookingStatusPaymentFailed}
		if result == entity.PaymentStatusTimeout {
			return dto.BookingResponse{}, http_error.PAYMENT_TIMEOUT.WithDetails(details)
		}
		return dto.BookingResponse{}, http_error.PAYMENT_DECLINED.WithDetails(details)
	})
}

func (s *bookingService) Cancel(ctx context.Context, userId, bookingId uuid.UUID) (dto.BookingResponse, error) {
	var res dto.BookingResponse
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		locked, err := s.lockOwned(ctx, userId, bookingId)
		if err != nil {
			return err
		}
		cancellable := locked.Status == entity.BookingStatusPendingPayment || locked.Status == entity.BookingStatusConfirmed
		if !cancellable || locked.EventStarted {
			return http_error.BOOKING_NOT_CANCELLABLE.WithDetails(map[string]any{"status": locked.Status})
		}
		cancelled, ok, err := s.bookingRepo.TransitionStatus(ctx, locked.Id, locked.Status, entity.BookingStatusCancelled)
		if err != nil {
			return err
		}
		if !ok {
			return http_error.INTERNAL_SERVER_ERROR
		}
		if err := s.ticketTypeRepo.Release(ctx, map[uuid.UUID]int{locked.TicketTypeId: locked.Quantity}); err != nil {
			return err
		}
		res = dto.NewBookingResponse(cancelled)
		return nil
	})
	return res, err
}

func (s *bookingService) lockOwned(ctx context.Context, userId, bookingId uuid.UUID) (repositories.LockedBooking, error) {
	locked, err := s.bookingRepo.LockForUpdate(ctx, bookingId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return locked, http_error.BOOKING_NOT_FOUND
	}
	if err != nil {
		return locked, err
	}
	// Someone else's booking is reported as missing so its ID is not confirmed (BR-11).
	if locked.UserId != userId {
		return locked, http_error.BOOKING_NOT_FOUND
	}
	return locked, nil
}

// transitionAndRelease moves a pending booking to a final status and returns
// its tickets in the same transaction.
func (s *bookingService) transitionAndRelease(ctx context.Context, b entity.Booking, to string) error {
	_, ok, err := s.bookingRepo.TransitionStatus(ctx, b.Id, entity.BookingStatusPendingPayment, to)
	if err != nil || !ok {
		return err
	}
	return s.ticketTypeRepo.Release(ctx, map[uuid.UUID]int{b.TicketTypeId: b.Quantity})
}

const outcomeOK = "OK"

// idempotent runs fn exactly once per (user, key). The key claim, the business
// writes and the stored outcome commit together. Business errors are stored
// and committed like successes, so a retry gets the same answer; unexpected
// errors roll everything back, freeing the key for a retry.
func (s *bookingService) idempotent(
	ctx context.Context, userId uuid.UUID, key, route string, req any,
	fn func(ctx context.Context) (dto.BookingResponse, error),
) (dto.BookingResponse, bool, error) {
	hash, err := requestHash(route, req)
	if err != nil {
		return dto.BookingResponse{}, false, err
	}

	var (
		result   dto.BookingResponse
		outcome  error
		replayed bool
	)
	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		claimed, err := s.idempotencyRepo.Claim(ctx, userId, key, hash, s.bookingConfig.GetIdempotencyTTL())
		if err != nil {
			return err
		}
		if !claimed {
			stored, err := s.idempotencyRepo.Get(ctx, userId, key)
			if err != nil {
				return err
			}
			if stored.RequestHash != hash {
				outcome = http_error.IDEMPOTENCY_KEY_REUSED
				return nil
			}
			replayed = true
			result, outcome, err = decodeOutcome(stored)
			return err
		}

		result, outcome = fn(ctx)
		if outcome != nil {
			if _, isBusiness := http_error.AsAppError(outcome); !isBusiness {
				return outcome
			}
		}
		code, body, err := encodeOutcome(result, outcome)
		if err != nil {
			return err
		}
		return s.idempotencyRepo.SaveResponse(ctx, userId, key, code, body)
	})
	if err != nil {
		return dto.BookingResponse{}, false, err
	}
	return result, replayed, outcome
}

func requestHash(route string, req any) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte(route+"\n"), body...))
	return hex.EncodeToString(sum[:]), nil
}

func encodeOutcome(result dto.BookingResponse, outcome error) (string, []byte, error) {
	if outcome == nil {
		body, err := json.Marshal(result)
		return outcomeOK, body, err
	}
	appErr, _ := http_error.AsAppError(outcome)
	body, err := json.Marshal(appErr.Details)
	return appErr.Code, body, err
}

func decodeOutcome(stored entity.IdempotencyKey) (dto.BookingResponse, error, error) {
	var result dto.BookingResponse
	if stored.ResponseCode == outcomeOK {
		err := json.Unmarshal(stored.ResponseBody, &result)
		return result, nil, err
	}
	var details map[string]any
	if len(stored.ResponseBody) > 0 {
		if err := json.Unmarshal(stored.ResponseBody, &details); err != nil {
			return result, nil, err
		}
	}
	return result, http_error.Lookup(stored.ResponseCode).WithDetails(details), nil
}
