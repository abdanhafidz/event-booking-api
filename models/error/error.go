package http_error

import (
	"errors"
	"net/http"
)

// AppError is a business error with a stable code that clients and tests can
// assert on. Copies made by WithDetails still match the original with errors.Is.
type AppError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *AppError) Error() string { return e.Message }

func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	return ok && t.Code == e.Code
}

func (e *AppError) WithDetails(details map[string]any) *AppError {
	c := *e
	c.Details = details
	return &c
}

var registry = map[string]*AppError{}

func newAppError(status int, code, message string) *AppError {
	e := &AppError{Status: status, Code: code, Message: message}
	registry[code] = e
	return e
}

// Lookup returns the catalogue entry for a code, used when replaying a stored
// idempotent response.
func Lookup(code string) *AppError {
	if e, ok := registry[code]; ok {
		return e
	}
	return INTERNAL_SERVER_ERROR
}

var (
	// ================= GENERAL =================
	VALIDATION_ERROR      = newAppError(http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed")
	ROUTE_NOT_FOUND       = newAppError(http.StatusNotFound, "ROUTE_NOT_FOUND", "Route not found")
	SERVICE_UNAVAILABLE   = newAppError(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is overloaded, retry shortly")
	INTERNAL_SERVER_ERROR = newAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")

	// ================= IDENTITY =================
	UNAUTHENTICATED = newAppError(http.StatusUnauthorized, "UNAUTHENTICATED", "Missing or unknown X-User-Id")

	// ================= IDEMPOTENCY =================
	IDEMPOTENCY_KEY_REQUIRED = newAppError(http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key header is required")
	IDEMPOTENCY_KEY_REUSED   = newAppError(http.StatusUnprocessableEntity, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key was already used with a different request")

	// ================= CATALOGUE =================
	EVENT_NOT_FOUND       = newAppError(http.StatusNotFound, "EVENT_NOT_FOUND", "Event not found")
	TICKET_TYPE_NOT_FOUND = newAppError(http.StatusNotFound, "TICKET_TYPE_NOT_FOUND", "Ticket type not found")

	// ================= BOOKING =================
	BOOKING_NOT_FOUND       = newAppError(http.StatusNotFound, "BOOKING_NOT_FOUND", "Booking not found")
	SALES_CLOSED            = newAppError(http.StatusConflict, "SALES_CLOSED", "Ticket sales for this event have closed")
	INSUFFICIENT_TICKETS    = newAppError(http.StatusConflict, "INSUFFICIENT_TICKETS", "Not enough tickets remain")
	BOOKING_EXPIRED         = newAppError(http.StatusConflict, "BOOKING_EXPIRED", "The booking hold has expired")
	BOOKING_NOT_PAYABLE     = newAppError(http.StatusConflict, "BOOKING_NOT_PAYABLE", "The booking cannot be paid in its current status")
	BOOKING_NOT_CANCELLABLE = newAppError(http.StatusConflict, "BOOKING_NOT_CANCELLABLE", "The booking cannot be cancelled in its current status")

	// ================= PAYMENT =================
	PAYMENT_DECLINED = newAppError(http.StatusPaymentRequired, "PAYMENT_DECLINED", "Payment was declined")
	PAYMENT_TIMEOUT  = newAppError(http.StatusPaymentRequired, "PAYMENT_TIMEOUT", "Payment provider did not respond in time")
)

// AsAppError reports whether err is a business error and returns it.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
