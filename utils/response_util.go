package utils

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"abdanhafidz.com/event-booking-api/models/dto"
	http_error "abdanhafidz.com/event-booking-api/models/error"
	"github.com/gin-gonic/gin"
)

func ResponseOK[Tdata any, TMetaData any](c *gin.Context, metaData TMetaData, data Tdata) {
	respond(c, http.StatusOK, metaData, data)
}

func ResponseCreated[Tdata any, TMetaData any](c *gin.Context, metaData TMetaData, data Tdata) {
	respond(c, http.StatusCreated, metaData, data)
}

func respond[Tdata any, TMetaData any](c *gin.Context, status int, metaData TMetaData, data Tdata) {
	c.JSON(status, dto.SuccessResponse[Tdata]{
		Status:   "success",
		Data:     data,
		Message:  "Request processed successfully",
		MetaData: metaData,
	})
}

// ResponseFAILED maps an error to its HTTP status. Business errors carry their
// own status and code; a request that ran out of time (usually waiting for a
// database connection) is reported as 503; anything else is a 500 and is logged.
func ResponseFAILED[TMetaData any](c *gin.Context, metaData TMetaData, err error) {
	appErr, ok := http_error.AsAppError(err)
	if !ok {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
			appErr = http_error.SERVICE_UNAVAILABLE
			c.Header("Retry-After", "1")
		} else {
			slog.Error("unhandled error", "error", err.Error(), "path", c.FullPath(), "request_id", c.GetString("request_id"))
			appErr = http_error.INTERNAL_SERVER_ERROR
		}
	}
	c.JSON(appErr.Status, dto.ErrorResponse{
		Status:   "error",
		Code:     appErr.Code,
		Errors:   appErr.Details,
		Message:  appErr.Message,
		MetaData: metaData,
	})
}

func SendResponse[Tdata any, TMetaData any](c *gin.Context, metaData TMetaData, data Tdata, err error) {
	if c.IsAborted() {
		return
	}
	if err != nil {
		ResponseFAILED(c, metaData, err)
	} else {
		ResponseOK(c, metaData, data)
	}
	c.Abort()
}
