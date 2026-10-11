package controllers

import (
	"encoding/json"
	"errors"
	"regexp"

	http_error "abdanhafidz.com/event-booking-api/models/error"
	"abdanhafidz.com/event-booking-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	uuid "github.com/google/uuid"
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ParseUserId reads the caller set by the user middleware.
func ParseUserId(ctx *gin.Context) uuid.UUID {
	id, _ := ctx.Get("user_id")
	userId, _ := id.(uuid.UUID)
	return userId
}

// ParseUUIDParam parses a path parameter. A malformed ID cannot match any row,
// so it is reported with the resource's not-found error.
func ParseUUIDParam(ctx *gin.Context, name string, notFound error) (uuid.UUID, bool) {
	id, err := uuid.Parse(ctx.Param(name))
	if err != nil {
		utils.ResponseFAILED[any](ctx, nil, notFound)
		ctx.Abort()
		return uuid.UUID{}, false
	}
	return id, true
}

// RequireIdempotencyKey enforces BR-09 before the body is read.
func RequireIdempotencyKey(ctx *gin.Context) (string, bool) {
	key := ctx.GetHeader("Idempotency-Key")
	if key == "" {
		utils.ResponseFAILED[any](ctx, nil, http_error.IDEMPOTENCY_KEY_REQUIRED)
		ctx.Abort()
		return "", false
	}
	if !idempotencyKeyPattern.MatchString(key) {
		utils.ResponseFAILED[any](ctx, nil, http_error.VALIDATION_ERROR.WithDetails(map[string]any{
			"field": "Idempotency-Key", "reason": "1-64 characters from A-Z a-z 0-9 _ -",
		}))
		ctx.Abort()
		return "", false
	}
	return key, true
}

// RequestJSON binds and validates the body. On failure it writes a
// VALIDATION_ERROR naming the field and aborts; callers must check IsAborted.
func RequestJSON[TRequest any](ctx *gin.Context) TRequest {
	var request TRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		utils.ResponseFAILED[any](ctx, nil, http_error.VALIDATION_ERROR.WithDetails(validationDetails(err)))
		ctx.Abort()
	}
	return request
}

func validationDetails(err error) map[string]any {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) && len(verrs) > 0 {
		return map[string]any{"field": verrs[0].Field(), "reason": verrs[0].Tag()}
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return map[string]any{"field": typeErr.Field, "reason": "must be " + typeErr.Type.String()}
	}
	return map[string]any{"reason": "malformed JSON body"}
}

func ResponseJSON[TResponse any, TMetaData any](ctx *gin.Context, metaData TMetaData, res TResponse, err error) {
	utils.SendResponse(ctx, metaData, res, err)
}

func markReplayed(ctx *gin.Context, replayed bool) {
	if replayed {
		ctx.Header("Idempotent-Replayed", "true")
	}
}
