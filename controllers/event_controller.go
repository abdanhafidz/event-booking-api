package controllers

import (
	"abdanhafidz.com/event-booking-api/models/dto"
	http_error "abdanhafidz.com/event-booking-api/models/error"
	"abdanhafidz.com/event-booking-api/services"
	"github.com/gin-gonic/gin"
)

// Referenced by the swagger annotations below.
var _ dto.EventResponse

type EventController interface {
	Get(ctx *gin.Context)
}

type eventController struct {
	eventService services.EventService
}

func NewEventController(eventService services.EventService) EventController {
	return &eventController{eventService: eventService}
}

// Get Event godoc
// @Summary      Get an event
// @Description  Event with its ticket types and live availability
// @Tags         Event
// @Produce      json
// @Param        id   path      string  true  "Event ID"
// @Success      200  {object}  dto.SuccessResponse[dto.EventResponse]
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /v1/events/{id} [get]
func (c *eventController) Get(ctx *gin.Context) {
	id, ok := ParseUUIDParam(ctx, "id", http_error.EVENT_NOT_FOUND)
	if !ok {
		return
	}
	res, err := c.eventService.Get(ctx.Request.Context(), id)
	ResponseJSON[any, any](ctx, nil, res, err)
}
