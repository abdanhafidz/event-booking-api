package integration

import (
	"bytes"
	"testing"
	"time"
)

func TestIdempotentCreateReplays(t *testing.T) { // IT-ID-01
	a := newApp(t)
	tt, u, k := a.ticketType(100), a.user(), key()

	first := a.createBooking(u, tt, 2, k)
	second := a.createBooking(u, tt, 2, k)
	expectCode(t, first, 201, "")
	expectCode(t, second, 201, "")
	if !bytes.Equal(first.Data, second.Data) {
		t.Fatalf("replayed body differs:\n%s\n%s", first.Data, second.Data)
	}
	expectEqual(t, "first replayed header", "", first.Header.Get("Idempotent-Replayed"))
	expectEqual(t, "second replayed header", "true", second.Header.Get("Idempotent-Replayed"))
	expectEqual(t, "bookings", 1, a.bookingsFor(tt))
	expectEqual(t, "available", 98, a.available(tt))
}

func TestIdempotencyKeyReusedWithDifferentBody(t *testing.T) { // IT-ID-02
	a := newApp(t)
	tt, u, k := a.ticketType(100), a.user(), key()

	expectCode(t, a.createBooking(u, tt, 2, k), 201, "")
	expectCode(t, a.createBooking(u, tt, 3, k), 422, "IDEMPOTENCY_KEY_REUSED")
	expectEqual(t, "available", 98, a.available(tt))
}

func TestIdempotentPayReplays(t *testing.T) { // IT-ID-03
	a := newApp(t)
	tt, u, k := a.ticketType(100), a.user(), key()
	b := a.book(u, tt, 2)

	first := a.pay(u, b.Id, "tok_success", k)
	second := a.pay(u, b.Id, "tok_success", k)
	expectCode(t, first, 200, "")
	expectCode(t, second, 200, "")
	if !bytes.Equal(first.Data, second.Data) {
		t.Fatalf("replayed body differs:\n%s\n%s", first.Data, second.Data)
	}
	expectEqual(t, "replayed header", "true", second.Header.Get("Idempotent-Replayed"))
	expectEqual(t, "succeeded payments", 1, a.payments(b.Id, "SUCCEEDED"))
}

func TestIdempotencyKeyRequired(t *testing.T) { // IT-ID-04
	a := newApp(t)
	tt, u := a.ticketType(100), a.user()
	b := a.book(u, tt, 1)

	expectCode(t, a.createBooking(u, tt, 1, ""), 400, "IDEMPOTENCY_KEY_REQUIRED")
	expectCode(t, a.pay(u, b.Id, "tok_success", ""), 400, "IDEMPOTENCY_KEY_REQUIRED")
	expectEqual(t, "available", 99, a.available(tt))
}

func TestIdempotentConcurrentDuplicates(t *testing.T) { // IT-ID-05
	a := newApp(t)
	tt, u, k := a.ticketType(100), a.user(), key()

	results := parallel(10, func(int) response { return a.createBooking(u, tt, 2, k) })
	first := results[0].booking(t)
	for _, r := range results {
		expectCode(t, r, 201, "")
		expectEqual(t, "booking id", first.Id, r.booking(t).Id)
	}
	expectEqual(t, "bookings", 1, a.bookingsFor(tt))
	expectEqual(t, "available", 98, a.available(tt))
}

func TestIdempotencyKeyExpires(t *testing.T) {
	a := newApp(t, withIdempotencyTTL(time.Second))
	tt, u, k := a.ticketType(100), a.user(), key()

	first := a.createBooking(u, tt, 1, k)
	time.Sleep(1500 * time.Millisecond)
	second := a.createBooking(u, tt, 1, k)
	expectCode(t, second, 201, "")
	if first.booking(t).Id == second.booking(t).Id {
		t.Fatalf("expired key should create a new booking")
	}
	expectEqual(t, "replayed header", "", second.Header.Get("Idempotent-Replayed"))
}
