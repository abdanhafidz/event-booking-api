package models

const (
	BookingStatusPendingPayment = "PENDING_PAYMENT"
	BookingStatusConfirmed      = "CONFIRMED"
	BookingStatusPaymentFailed  = "PAYMENT_FAILED"
	BookingStatusExpired        = "EXPIRED"
	BookingStatusCancelled      = "CANCELLED"
)

const (
	PaymentStatusSucceeded = "SUCCEEDED"
	PaymentStatusDeclined  = "DECLINED"
	PaymentStatusTimeout   = "TIMEOUT"
)

// Mock payment tokens. Any unknown token is treated as a decline.
const (
	PaymentTokenSuccess = "tok_success"
	PaymentTokenDecline = "tok_decline"
	PaymentTokenTimeout = "tok_timeout"
)

const DefaultCurrency = "IDR"
