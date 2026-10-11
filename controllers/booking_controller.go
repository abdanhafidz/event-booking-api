package controllers

import (
	"abdanhafidz.com/event-booking-api/models/dto"
	http_error "abdanhafidz.com/event-booking-api/models/error"
	"abdanhafidz.com/event-booking-api/services"
	"abdanhafidz.com/event-booking-api/utils"
	"github.com/gin-gonic/gin"
)

type BookingController interface {
	Create(ctx *gin.Context)
	Get(ctx *gin.Context)
	Pay(ctx *gin.Context)
	Cancel(ctx *gin.Context)
}

type bookingController struct {
	bookingService services.BookingService
}

func NewBookingController(bookingService services.BookingService) BookingController {
	return &bookingController{bookingService: bookingService}
}

// Create Booking godoc
// @Summary      Reserve tickets
// @Description  Reserves tickets all-or-nothing and starts the payment hold
// @Tags         Booking
// @Accept       json
// @Produce      json
// @Param        X-User-Id        header    string                    true  "Caller user ID"
// @Param        Idempotency-Key  header    string                    true  "Idempotency key"
// @Param        request          body      dto.CreateBookingRequest  true  "Booking request"
// @Success      201  {object}  dto.SuccessResponse[dto.BookingResponse]
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      409  {object}  dto.ErrorResponse
// @Failure      422  {object}  dto.ErrorResponse
// @Router       /v1/bookings [post]
func (c *bookingController) Create(ctx *gin.Context) {
	key, ok := RequireIdempotencyKey(ctx)
	if !ok {
		return
	}
	req := RequestJSON[dto.CreateBookingRequest](ctx)
	if ctx.IsAborted() {
		return
	}
	res, replayed, err := c.bookingService.Create(ctx.Request.Context(), ParseUserId(ctx), key, req)
	markReplayed(ctx, replayed)
	if err != nil {
		ResponseJSON[any, any](ctx, nil, res, err)
		return
	}
	utils.ResponseCreated[any, any](ctx, nil, res)
	ctx.Abort()
}

// Get Booking godoc
// @Summary      Get one of your bookings
// @Tags         Booking
// @Produce      json
// @Param        X-User-Id  header    string  true  "Caller user ID"
// @Param        id         path      string  true  "Booking ID"
// @Success      200  {object}  dto.SuccessResponse[dto.BookingResponse]
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /v1/bookings/{id} [get]
func (c *bookingController) Get(ctx *gin.Context) {
	id, ok := ParseUUIDParam(ctx, "id", http_error.BOOKING_NOT_FOUND)
	if !ok {
		return
	}
	res, err := c.bookingService.Get(ctx.Request.Context(), ParseUserId(ctx), id)
	ResponseJSON[any, any](ctx, nil, res, err)
}

// Pay Booking godoc
// @Summary      Pay for a booking with the mock gateway
// @Description  tok_success confirms; tok_decline and tok_timeout fail the booking and release its tickets
// @Tags         Booking
// @Accept       json
// @Produce      json
// @Param        X-User-Id        header    string                 true  "Caller user ID"
// @Param        Idempotency-Key  header    string                 true  "Idempotency key"
// @Param        id               path      string                 true  "Booking ID"
// @Param        request          body      dto.PayBookingRequest  true  "Payment request"
// @Success      200  {object}  dto.SuccessResponse[dto.BookingResponse]
// @Failure      402  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      409  {object}  dto.ErrorResponse
// @Failure      422  {object}  dto.ErrorResponse
// @Router       /v1/bookings/{id}/pay [post]
func (c *bookingController) Pay(ctx *gin.Context) {
	id, ok := ParseUUIDParam(ctx, "id", http_error.BOOKING_NOT_FOUND)
	if !ok {
		return
	}
	key, ok := RequireIdempotencyKey(ctx)
	if !ok {
		return
	}
	req := RequestJSON[dto.PayBookingRequest](ctx)
	if ctx.IsAborted() {
		return
	}
	res, replayed, err := c.bookingService.Pay(ctx.Request.Context(), ParseUserId(ctx), id, key, req)
	markReplayed(ctx, replayed)
	ResponseJSON[any, any](ctx, nil, res, err)
}

// Cancel Booking godoc
// @Summary      Cancel a pending or confirmed booking before the event starts
// @Tags         Booking
// @Produce      json
// @Param        X-User-Id  header    string  true  "Caller user ID"
// @Param        id         path      string  true  "Booking ID"
// @Success      200  {object}  dto.SuccessResponse[dto.BookingResponse]
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      409  {object}  dto.ErrorResponse
// @Router       /v1/bookings/{id}/cancel [post]
func (c *bookingController) Cancel(ctx *gin.Context) {
	id, ok := ParseUUIDParam(ctx, "id", http_error.BOOKING_NOT_FOUND)
	if !ok {
		return
	}
	res, err := c.bookingService.Cancel(ctx.Request.Context(), ParseUserId(ctx), id)
	ResponseJSON[any, any](ctx, nil, res, err)
}
