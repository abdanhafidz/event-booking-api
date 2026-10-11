package repositories

import (
	"context"
	"time"

	entity "abdanhafidz.com/event-booking-api/models/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IdempotencyRepository interface {
	Claim(ctx context.Context, userId uuid.UUID, key, requestHash string, ttl time.Duration) (bool, error)
	Get(ctx context.Context, userId uuid.UUID, key string) (entity.IdempotencyKey, error)
	SaveResponse(ctx context.Context, userId uuid.UUID, key, code string, body []byte) error
}

type idempotencyRepository struct {
	db *gorm.DB
}

func NewIdempotencyRepository(db *gorm.DB) IdempotencyRepository {
	return &idempotencyRepository{db: db}
}

// Claim inserts the key, or takes over one that has expired. It must be the
// first statement of the business transaction: a second request with the same
// live key blocks on this insert until the first transaction ends, then gets
// false and replays the stored response. That is what serialises duplicates.
func (r *idempotencyRepository) Claim(ctx context.Context, userId uuid.UUID, key, requestHash string, ttl time.Duration) (bool, error) {
	var claimed []uuid.UUID
	err := conn(ctx, r.db).Raw(`
		INSERT INTO idempotency_keys (user_id, key, request_hash, response_code, response_body, expires_at)
		VALUES (?, ?, ?, '', NULL, now() + make_interval(secs => ?))
		ON CONFLICT (user_id, key) DO UPDATE
			SET request_hash = EXCLUDED.request_hash, response_code = '', response_body = NULL,
			    expires_at = EXCLUDED.expires_at
			WHERE idempotency_keys.expires_at <= now()
		RETURNING user_id`,
		userId, key, requestHash, ttl.Seconds(),
	).Scan(&claimed).Error
	return len(claimed) == 1, err
}

func (r *idempotencyRepository) Get(ctx context.Context, userId uuid.UUID, key string) (entity.IdempotencyKey, error) {
	var k entity.IdempotencyKey
	err := conn(ctx, r.db).First(&k, "user_id = ? AND key = ?", userId, key).Error
	return k, err
}

func (r *idempotencyRepository) SaveResponse(ctx context.Context, userId uuid.UUID, key, code string, body []byte) error {
	return conn(ctx, r.db).Exec(
		`UPDATE idempotency_keys SET response_code = ?, response_body = ? WHERE user_id = ? AND key = ?`,
		code, body, userId, key,
	).Error
}
