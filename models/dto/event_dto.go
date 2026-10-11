package dto

import (
	"time"

	entity "abdanhafidz.com/event-booking-api/models/entity"
	"github.com/google/uuid"
)

type TicketTypeResponse struct {
	Id        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Price     int64     `json:"price"`
	Capacity  int       `json:"capacity"`
	Available int       `json:"available"`
}

type EventResponse struct {
	Id          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	StartsAt    time.Time            `json:"starts_at"`
	Currency    string               `json:"currency"`
	TicketTypes []TicketTypeResponse `json:"ticket_types"`
}

func NewEventResponse(e entity.Event) EventResponse {
	res := EventResponse{
		Id:          e.Id,
		Name:        e.Name,
		StartsAt:    e.StartsAt.UTC(),
		Currency:    entity.DefaultCurrency,
		TicketTypes: make([]TicketTypeResponse, 0, len(e.TicketTypes)),
	}
	for _, t := range e.TicketTypes {
		res.TicketTypes = append(res.TicketTypes, TicketTypeResponse{
			Id:        t.Id,
			Name:      t.Name,
			Price:     t.Price,
			Capacity:  t.Capacity,
			Available: t.Available,
		})
	}
	return res
}
