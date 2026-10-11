package repositories

import (
	"context"

	entity "abdanhafidz.com/event-booking-api/models/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	err := conn(ctx, r.db).Model(&entity.User{}).Where("id = ?", id).Limit(1).Count(&count).Error
	return count > 0, err
}
