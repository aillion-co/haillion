package domain

import (
	"context"
	"errors"
)

type State string

const (
	StateRequested  State = "requested"
	StateAccepted   State = "accepted"
	StateInProgress State = "in-progress"
	StateCompleted  State = "completed"
)

// ErrInvalidStateTransition is returned when a state transition is rejected or race condition is hit.
var ErrInvalidStateTransition = errors.New("invalid state transition or concurrent modification")

// ErrTripNotFound is returned when a trip cannot be found.
var ErrTripNotFound = errors.New("trip not found")

type Trip struct {
	ID       string
	RiderID  string
	DriverID *string
	State    State
}

type TripRepository interface {
	Create(ctx context.Context, t *Trip) error
	UpdateState(ctx context.Context, id string, driverID *string, newState State, expectedCurrent State) error
	GetByID(ctx context.Context, id string) (*Trip, error)
}
