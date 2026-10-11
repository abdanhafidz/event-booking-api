package services

import (
	"context"
	"testing"
	"time"

	dto "abdanhafidz.com/event-booking-api/models/dto"
	entity "abdanhafidz.com/event-booking-api/models/entity"
	http_error "abdanhafidz.com/event-booking-api/models/error"
	"github.com/google/uuid"
)

func TestMockGatewayOutcomes(t *testing.T) {
	g := NewMockPaymentGatewayService(0, 50*time.Millisecond)
	cases := map[string]string{
		entity.PaymentTokenSuccess: entity.PaymentStatusSucceeded,
		entity.PaymentTokenDecline: entity.PaymentStatusDeclined,
		"tok_unknown":              entity.PaymentStatusDeclined,
		entity.PaymentTokenTimeout: entity.PaymentStatusTimeout,
	}
	for token, want := range cases {
		got, err := g.Charge(context.Background(), 1000, token)
		if err != nil || got != want {
			t.Errorf("%s: want %s, got %s (err %v)", token, want, got, err)
		}
	}
}

func TestMockGatewayTimeoutIsBoundedByClientTimeout(t *testing.T) {
	g := NewMockPaymentGatewayService(0, 100*time.Millisecond)
	start := time.Now()
	got, _ := g.Charge(context.Background(), 1000, entity.PaymentTokenTimeout)
	if got != entity.PaymentStatusTimeout || time.Since(start) > 500*time.Millisecond {
		t.Fatalf("want TIMEOUT after ~100ms, got %s after %v", got, time.Since(start))
	}
}

func TestMockGatewayHonoursCallerCancellation(t *testing.T) {
	g := NewMockPaymentGatewayService(time.Second, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := g.Charge(ctx, 1000, entity.PaymentTokenSuccess); err == nil {
		t.Fatal("want the caller's context error, got nil")
	}
}

func TestRequestHashDependsOnRouteAndBody(t *testing.T) {
	tt := uuid.New()
	a, _ := requestHash("POST /v1/bookings", dto.CreateBookingRequest{TicketTypeId: tt, Quantity: 2})
	b, _ := requestHash("POST /v1/bookings", dto.CreateBookingRequest{TicketTypeId: tt, Quantity: 2})
	c, _ := requestHash("POST /v1/bookings", dto.CreateBookingRequest{TicketTypeId: tt, Quantity: 3})
	d, _ := requestHash("POST /v1/bookings/x/pay", dto.CreateBookingRequest{TicketTypeId: tt, Quantity: 2})
	if a != b || a == c || a == d {
		t.Fatalf("hash: same=%v differentBody=%v differentRoute=%v", a == b, a != c, a != d)
	}
}

func TestStoredOutcomeRoundTrip(t *testing.T) {
	code, body, err := encodeOutcome(dto.BookingResponse{}, http_error.INSUFFICIENT_TICKETS.WithDetails(map[string]any{"available": 3}))
	if err != nil {
		t.Fatal(err)
	}
	_, outcome, err := decodeOutcome(entity.IdempotencyKey{ResponseCode: code, ResponseBody: body})
	appErr, ok := http_error.AsAppError(outcome)
	if err != nil || !ok || appErr.Code != "INSUFFICIENT_TICKETS" || appErr.Details["available"] != float64(3) {
		t.Fatalf("round trip lost the outcome: %v %v", outcome, err)
	}
}
