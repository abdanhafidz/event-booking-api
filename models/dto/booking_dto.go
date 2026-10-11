package dto

import (
	"time"

	entity "abdanhafidz.com/event-booking-api/models/entity"
	"github.com/google/uuid"
)

type CreateBookingRequest struct {
	TicketTypeId uuid.UUID `json:"ticket_type_id" binding:"required"`
	Quantity     int       `json:"quantity" binding:"required,min=1,max=6"`
}

type PayBookingRequest struct {
	PaymentToken string `json:"payment_token" binding:"required,max=64"`
}

type BookingResponse struct {
	Id           uuid.UUID `json:"id"`
	EventId      uuid.UUID `json:"event_id"`
	TicketTypeId uuid.UUID `json:"ticket_type_id"`
	Quantity     int       `json:"quantity"`
	UnitPrice    int64     `json:"unit_price"`
	TotalAmount  int64     `json:"total_amount"`
	Currency     string    `json:"currency"`
	Status       string    `json:"status"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewBookingResponse(b entity.Booking) BookingResponse {
	return BookingResponse{
		Id:           b.Id,
		EventId:      b.EventId,
		TicketTypeId: b.TicketTypeId,
		Quantity:     b.Quantity,
		UnitPrice:    b.UnitPrice,
		TotalAmount:  b.TotalAmount,
		Currency:     b.Currency,
		Status:       b.Status,
		ExpiresAt:    b.ExpiresAt.UTC(),
		CreatedAt:    b.CreatedAt.UTC(),
		UpdatedAt:    b.UpdatedAt.UTC(),
	}
}
