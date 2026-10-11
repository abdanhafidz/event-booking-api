package integration

import (
	"testing"
	"time"
)

func TestPaySuccessConfirms(t *testing.T) { // IT-PY-01
	a := newApp(t)
	tt, u := a.ticketType(100), a.user()
	b := a.book(u, tt, 2)

	res := a.pay(u, b.Id, "tok_success", key())
	expectCode(t, res, 200, "")
	expectEqual(t, "status", "CONFIRMED", res.booking(t).Status)
	expectEqual(t, "succeeded payments", 1, a.payments(b.Id, "SUCCEEDED"))
	expectEqual(t, "available", 98, a.available(tt))
}

func TestPayDeclineFailsAndReleases(t *testing.T) { // IT-PY-02
	a := newApp(t)
	tt, u := a.ticketType(100), a.user()
	b := a.book(u, tt, 2)

	res := a.pay(u, b.Id, "tok_decline", key())
	expectCode(t, res, 402, "PAYMENT_DECLINED")
	expectEqual(t, "errors.booking_status", any("PAYMENT_FAILED"), res.Errors["booking_status"])
	expectEqual(t, "status", "PAYMENT_FAILED", a.status(b.Id))
	expectEqual(t, "available", 100, a.available(tt))
}

func TestPayTimeoutFailsAndReleases(t *testing.T) { // IT-PY-03
	a := newApp(t)
	tt, u := a.ticketType(100), a.user()
	b := a.book(u, tt, 2)

	res := a.pay(u, b.Id, "tok_timeout", key())
	expectCode(t, res, 402, "PAYMENT_TIMEOUT")
	if res.Duration < 2*time.Second || res.Duration > 3*time.Second {
		t.Fatalf("timeout should fire after the 2s client timeout, took %v", res.Duration)
	}
	expectEqual(t, "status", "PAYMENT_FAILED", a.status(b.Id))
	expectEqual(t, "succeeded payments", 0, a.payments(b.Id, "SUCCEEDED"))
	expectEqual(t, "available", 100, a.available(tt))
}

func TestPayAfterHoldExpires(t *testing.T) { // IT-PY-04
	a := newApp(t, withHold(2*time.Second))
	tt, u := a.ticketType(100), a.user()
	b := a.book(u, tt, 2)

	time.Sleep(3 * time.Second) // sweeper is off: the pay request must notice expiry itself
	expectCode(t, a.pay(u, b.Id, "tok_success", key()), 409, "BOOKING_EXPIRED")
	expectEqual(t, "status", "EXPIRED", a.status(b.Id))
	expectEqual(t, "available", 100, a.available(tt))
	expectEqual(t, "payment attempts", 0, a.payments(b.Id, "SUCCEEDED")+a.payments(b.Id, "DECLINED"))
}

func TestPayNonPendingBooking(t *testing.T) { // IT-PY-05
	a := newApp(t)
	tt, u := a.ticketType(100), a.user()

	confirmed := a.book(u, tt, 1)
	expectCode(t, a.pay(u, confirmed.Id, "tok_success", key()), 200, "")
	res := a.pay(u, confirmed.Id, "tok_success", key())
	expectCode(t, res, 409, "BOOKING_NOT_PAYABLE")
	expectEqual(t, "errors.status", any("CONFIRMED"), res.Errors["status"])
	expectEqual(t, "succeeded payments", 1, a.payments(confirmed.Id, "SUCCEEDED"))

	failed := a.book(u, tt, 1)
	expectCode(t, a.pay(u, failed.Id, "tok_decline", key()), 402, "PAYMENT_DECLINED")
	expectCode(t, a.pay(u, failed.Id, "tok_success", key()), 409, "BOOKING_NOT_PAYABLE")
	expectEqual(t, "succeeded payments", 0, a.payments(failed.Id, "SUCCEEDED"))
}

func TestCancelPending(t *testing.T) { // IT-CN-01
	a := newApp(t)
	tt, u := a.ticketType(100), a.user()
	b := a.book(u, tt, 2)

	res := a.cancel(u, b.Id)
	expectCode(t, res, 200, "")
	expectEqual(t, "status", "CANCELLED", res.booking(t).Status)
	expectEqual(t, "available", 100, a.available(tt))
}

func TestCancelConfirmed(t *testing.T) { // IT-CN-02
	a := newApp(t)
	tt, u := a.ticketType(100), a.user()
	b := a.book(u, tt, 2)
	expectCode(t, a.pay(u, b.Id, "tok_success", key()), 200, "")

	expectCode(t, a.cancel(u, b.Id), 200, "")
	expectEqual(t, "status", "CANCELLED", a.status(b.Id))
	expectEqual(t, "available", 100, a.available(tt))
}

func TestCancelNotAllowed(t *testing.T) { // IT-CN-03
	a := newApp(t, withHold(time.Second))
	ev, ids := a.event(7*24*time.Hour, 100)
	tt, u := ids[0], a.user()

	cancelled := a.book(u, tt, 1)
	expectCode(t, a.cancel(u, cancelled.Id), 200, "")

	failed := a.book(u, tt, 1)
	expectCode(t, a.pay(u, failed.Id, "tok_decline", key()), 402, "PAYMENT_DECLINED")

	expired := a.book(u, tt, 1)
	time.Sleep(1500 * time.Millisecond)
	a.sweep()
	expectEqual(t, "expired status", "EXPIRED", a.status(expired.Id))

	for _, b := range []booking{cancelled, failed, expired} {
		expectCode(t, a.cancel(u, b.Id), 409, "BOOKING_NOT_CANCELLABLE")
	}

	// A confirmed booking whose event has already started.
	started := a.book(u, tt, 1)
	expectCode(t, a.pay(u, started.Id, "tok_success", key()), 200, "")
	a.exec(`UPDATE events SET starts_at = now() - interval '1 second' WHERE id = ?`, ev)
	expectCode(t, a.cancel(u, started.Id), 409, "BOOKING_NOT_CANCELLABLE")
	expectEqual(t, "available", 99, a.available(tt))
}

func TestSweeperExpiresUnpaidBookings(t *testing.T) { // IT-EX-01
	a := newApp(t, withHold(2*time.Second), withSweeper(5*time.Second))
	tt, u := a.ticketType(100), a.user()
	pending := a.book(u, tt, 2)
	confirmed := a.book(u, tt, 1)
	expectCode(t, a.pay(u, confirmed.Id, "tok_success", key()), 200, "")

	deadline := pending.ExpiresAt.Add(15 * time.Second)
	for a.status(pending.Id) != "EXPIRED" {
		if time.Now().After(deadline) {
			t.Fatalf("booking not expired within 15s of expires_at")
		}
		time.Sleep(500 * time.Millisecond)
	}
	expectEqual(t, "available", 99, a.available(tt))
	expectEqual(t, "confirmed untouched", "CONFIRMED", a.status(confirmed.Id))
}
