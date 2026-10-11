package integration

import (
	"testing"
	"time"
)

// Run these with -count=20 in CI: a race that fails one run in twenty is still a bug.

func TestConcurrentBookingsNeverOversell(t *testing.T) { // IT-CC-01
	a := newApp(t)
	tt := a.ticketType(50)
	users := a.users(200)

	results := parallel(200, func(i int) response { return a.createBooking(users[i], tt, 1, key()) })
	counts := countCodes(results)
	expectEqual(t, "201 Created", 50, counts["Created"])
	expectEqual(t, "INSUFFICIENT_TICKETS", 150, counts["INSUFFICIENT_TICKETS"])
	expectEqual(t, "available", 0, a.available(tt))
}

func TestConcurrentMixedQuantities(t *testing.T) { // IT-CC-02
	a := newApp(t)
	tt := a.ticketType(10)
	users := a.users(20)

	results := parallel(20, func(i int) response { return a.createBooking(users[i], tt, i%3+1, key()) })
	booked := 0
	for i, r := range results {
		switch {
		case r.Code == 201:
			booked += i%3 + 1
		case r.ErrCode != "INSUFFICIENT_TICKETS":
			t.Fatalf("unexpected response %d %s", r.Code, r.RawBody)
		}
	}
	if booked > 10 {
		t.Fatalf("oversold: %d of 10", booked)
	}
	expectEqual(t, "available", 10-booked, a.available(tt))
}

func TestConcurrentPayAndCancel(t *testing.T) { // IT-CC-03
	a := newApp(t, withLatency(200*time.Millisecond))
	tt := a.ticketType(100)
	users := a.users(50)
	bookings := make([]booking, 50)
	for i, u := range users {
		bookings[i] = a.book(u, tt, 1)
	}

	results := parallel(100, func(i int) response {
		b, u := bookings[i/2], users[i/2]
		if i%2 == 0 {
			return a.pay(u, b.Id, "tok_success", key())
		}
		return a.cancel(u, b.Id)
	})

	for i, b := range bookings {
		pay, cancel := results[2*i], results[2*i+1]
		expectCode(t, cancel, 200, "") // cancel works on both pending and confirmed bookings
		if pay.Code != 200 && pay.ErrCode != "BOOKING_NOT_PAYABLE" {
			t.Fatalf("pay: unexpected %d %s", pay.Code, pay.RawBody)
		}
		expectEqual(t, "final status", "CANCELLED", a.status(b.Id))
		if n := a.payments(b.Id, "SUCCEEDED"); n > 1 {
			t.Fatalf("booking %s paid %d times", b.Id, n)
		}
	}
	expectEqual(t, "available", 100, a.available(tt))
}

func TestConcurrentPaysWithDifferentKeys(t *testing.T) { // IT-CC-04
	a := newApp(t, withLatency(500*time.Millisecond))
	tt, u := a.ticketType(100), a.user()
	b := a.book(u, tt, 1)

	results := parallel(5, func(int) response { return a.pay(u, b.Id, "tok_success", key()) })
	counts := countCodes(results)
	expectEqual(t, "200 OK", 1, counts["OK"])
	expectEqual(t, "BOOKING_NOT_PAYABLE", 4, counts["BOOKING_NOT_PAYABLE"])
	expectEqual(t, "succeeded payments", 1, a.payments(b.Id, "SUCCEEDED"))
}
