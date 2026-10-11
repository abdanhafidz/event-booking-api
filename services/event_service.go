package services

import (
	"context"
	"errors"

	dto "abdanhafidz.com/event-booking-api/models/dto"
	http_error "abdanhafidz.com/event-booking-api/models/error"
	"abdanhafidz.com/event-booking-api/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventService interface {
	Get(ctx context.Context, id uuid.UUID) (dto.EventResponse, error)
}

type eventService struct {
	eventRepo repositories.EventRepository
}

func NewEventService(eventRepo repositories.EventRepository) EventService {
	return &eventService{eventRepo: eventRepo}
}

func (s *eventService) Get(ctx context.Context, id uuid.UUID) (dto.EventResponse, error) {
	event, err := s.eventRepo.GetWithTicketTypes(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.EventResponse{}, http_error.EVENT_NOT_FOUND
	}
	if err != nil {
		return dto.EventResponse{}, err
	}
	return dto.NewEventResponse(event), nil
}
