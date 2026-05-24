package domain

import (
	"context"
	"errors"
)

type PaymentState string

const (
	PaymentStatePending PaymentState = "pending"
	PaymentStatePaid    PaymentState = "paid"
)

// ErrPaymentNotFound is returned when a payment cannot be found.
var ErrPaymentNotFound = errors.New("payment not found")

type Payment struct {
	ID     string
	TripID string
	Amount int64 // in cents
	State  PaymentState
}

type PaymentRepository interface {
	Create(ctx context.Context, p *Payment) error
	MarkPaid(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Payment, error)
}
