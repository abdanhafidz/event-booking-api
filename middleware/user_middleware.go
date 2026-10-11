package middleware

import (
	"sync"

	http_error "abdanhafidz.com/event-booking-api/models/error"
	"abdanhafidz.com/event-booking-api/repositories"
	"abdanhafidz.com/event-booking-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserMiddleware resolves the caller from X-User-Id. There is no real auth in
// this project (PRD A1); the header only has to name a seeded user.
type UserMiddleware interface {
	RequireUser(ctx *gin.Context)
}

type userMiddleware struct {
	userRepo repositories.UserRepository
	// Users are never deleted, so a positive lookup is cached for the life of
	// the process. This saves one database round trip per request under load.
	known sync.Map
}

func NewUserMiddleware(userRepo repositories.UserRepository) UserMiddleware {
	return &userMiddleware{userRepo: userRepo}
}

func (m *userMiddleware) RequireUser(c *gin.Context) {
	userId, err := uuid.Parse(c.GetHeader("X-User-Id"))
	if err != nil {
		utils.ResponseFAILED[any](c, nil, http_error.UNAUTHENTICATED)
		c.Abort()
		return
	}
	if _, ok := m.known.Load(userId); !ok {
		exists, err := m.userRepo.Exists(c.Request.Context(), userId)
		if err != nil {
			utils.ResponseFAILED[any](c, nil, err)
			c.Abort()
			return
		}
		if !exists {
			utils.ResponseFAILED[any](c, nil, http_error.UNAUTHENTICATED)
			c.Abort()
			return
		}
		m.known.Store(userId, struct{}{})
	}
	c.Set("user_id", userId)
	c.Next()
}
