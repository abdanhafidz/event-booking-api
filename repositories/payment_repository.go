package repositories

import (
	"context"

	entity "abdanhafidz.com/event-booking-api/models/entity"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment entity.Payment) (entity.Payment, error)
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, payment entity.Payment) (entity.Payment, error) {
	if err := conn(ctx, r.db).Create(&payment).Error; err != nil {
		return entity.Payment{}, err
	}
	return payment, nil
}
