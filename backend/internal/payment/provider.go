package payment

import "google-ai-proxy/internal/db"

// PaymentResult is returned by CreatePayment with the redirect URL.
type PaymentResult struct {
	PaymentURL string // Full URL to redirect user for payment
}

// QueryResult is returned by QueryOrder for order status queries.
type QueryResult struct {
	TradeNo     string // Provider's trade number
	TradeStatus string // e.g. TRADE_SUCCESS
}

// PaymentProvider abstracts a payment gateway.
type PaymentProvider interface {
	Name() string
	CreatePayment(order *db.PaymentOrder) (*PaymentResult, error)
	VerifyNotify(params map[string]string) (bool, error)
	QueryOrder(order *db.PaymentOrder) (*QueryResult, error)
}
