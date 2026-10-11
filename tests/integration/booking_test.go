package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBookingCreatesPendingHold(t *testing.T) { // IT-BK-01
	a := newApp(t)
	tt, u := a.ticketType(100), a.user()

	res := a.createBooking(u, tt, 2, key())
	expectCode(t, res, 201, "")
	b := res.booking(t)
	expectEqual(t, "status", "PENDING_PAYMENT", b.Status)
	expectEqual(t, "unit_price", int64(150000), b.UnitPrice)
	expectEqual(t, "total_amount", int64(300000), b.TotalAmount)
	hold := b.ExpiresAt.Sub(b.CreatedAt)
	if hold < 599*time.Second || hold > 601*time.Second {
		t.Fatalf("hold: want 600s ±1s, got %v", hold)
	}
	expectEqual(t, "available", 98, a.available(tt))
}

func TestBookingRejectsInvalidQuantity(t *testing.T) { // IT-BK-02
	a := newApp(t)
	tt, u := a.ticketType(100), a.user()

	bodies := []string{
		`{"ticket_type_id":"` + tt.String() + `","quantity":0}`,
		`{"ticket_type_id":"` + tt.String() + `","quantity":7}`,
		`{"ticket_type_id":"` + tt.String() + `","quantity":1.5}`,
		`{"ticket_type_id":"` + tt.String() + `","quantity":"2"}`,
		`{"ticket_type_id":"` + tt.String() + `"}`,
	}
	for _, body := range bodies {
		res := a.request("POST", "/v1/bookings", u, key(), body)
		expectCode(t, res, 400, "VALIDATION_ERROR")
		expectEqual(t, "errors.field for "+body, any("quantity"), res.Errors["field"])
	}
	expectEqual(t, "available", 100, a.available(tt))
	expectEqual(t, "bookings", 0, a.bookingsFor(tt))
}

func TestBookingUnknownTicketType(t *testing.T) { // IT-BK-03
	a := newApp(t)
	expectCode(t, a.createBooking(a.user(), uuid.New(), 1, key()), 404, "TICKET_TYPE_NOT_FOUND")
}

func TestBookingSalesClosedAfterEventStart(t *testing.T) { // IT-BK-04
	a := newApp(t)
	_, ids := a.event(-time.Second, 10)
	expectCode(t, a.createBooking(a.user(), ids[0], 1, key()), 409, "SALES_CLOSED")
	expectEqual(t, "available", 10, a.available(ids[0]))
}

func TestBookingOwnershipAndIdentity(t *testing.T) { // IT-BK-05
	a := newApp(t)
	tt := a.ticketType(100)
	owner, other := a.user(), a.user()
	b := a.book(owner, tt, 2)

	expectCode(t, a.request("GET", "/v1/bookings/"+b.Id.String(), other, "", nil), 404, "BOOKING_NOT_FOUND")
	expectCode(t, a.pay(other, b.Id, "tok_success", key()), 404, "BOOKING_NOT_FOUND")
	expectCode(t, a.cancel(other, b.Id), 404, "BOOKING_NOT_FOUND")
	expectEqual(t, "status", "PENDING_PAYMENT", a.status(b.Id))

	expectCode(t, a.request("GET", "/v1/bookings/"+b.Id.String(), uuid.Nil, "", nil), 401, "UNAUTHENTICATED")
	expectCode(t, a.request("GET", "/v1/bookings/"+b.Id.String(), uuid.New(), "", nil), 401, "UNAUTHENTICATED")

	res := a.request("GET", "/v1/bookings/"+b.Id.String(), owner, "", nil)
	expectCode(t, res, 200, "")
	expectEqual(t, "id", b.Id, res.booking(t).Id)
}

func TestAvailabilityLastTickets(t *testing.T) { // IT-AV-01
	a := newApp(t)
	tt := a.ticketType(5)
	a.book(a.user(), tt, 5)

	res := a.createBooking(a.user(), tt, 1, key())
	expectCode(t, res, 409, "INSUFFICIENT_TICKETS")
	expectEqual(t, "errors.available", any(float64(0)), res.Errors["available"])
}

func TestAvailabilityNoPartialReservation(t *testing.T) { // IT-AV-02
	a := newApp(t)
	tt := a.ticketType(5)
	a.book(a.user(), tt, 2)

	res := a.createBooking(a.user(), tt, 4, key())
	expectCode(t, res, 409, "INSUFFICIENT_TICKETS")
	expectEqual(t, "errors.available", any(float64(3)), res.Errors["available"])
	expectEqual(t, "available", 3, a.available(tt))
}

func TestEventShowsLiveAvailability(t *testing.T) {
	a := newApp(t)
	ev, ids := a.event(7*24*time.Hour, 10, 20)
	a.book(a.user(), ids[0], 3)

	res := a.request("GET", "/v1/events/"+ev.String(), uuid.Nil, "", nil)
	expectCode(t, res, 200, "")
	expectCode(t, a.request("GET", "/v1/events/"+uuid.NewString(), uuid.Nil, "", nil), 404, "EVENT_NOT_FOUND")
	expectEqual(t, "available", 7, a.available(ids[0]))
}
