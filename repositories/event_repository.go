package repositories

import (
	"context"

	entity "abdanhafidz.com/event-booking-api/models/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventRepository interface {
	GetWithTicketTypes(ctx context.Context, id uuid.UUID) (entity.Event, error)
}

type eventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) GetWithTicketTypes(ctx context.Context, id uuid.UUID) (entity.Event, error) {
	var event entity.Event
	err := conn(ctx, r.db).
		Preload("TicketTypes", func(db *gorm.DB) *gorm.DB { return db.Order("name") }).
		First(&event, "id = ?", id).Error
	return event, err
}
