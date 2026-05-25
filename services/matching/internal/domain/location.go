package domain

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Location struct {
	Lat float64
	Lng float64
}

type DriverLocation struct {
	DriverID string
	Location Location
}

type PendingRiderHit struct {
	RiderID        string
	Location       Location
	DistanceMeters float64
}

type DriverLocationHit struct {
	DriverID       string
	Location       Location
	DistanceMeters float64
}

type LocationRepository interface {
	UpdateLocation(ctx context.Context, driverID string, loc Location) error
	FindNearbyDrivers(ctx context.Context, riderLoc Location, radiusMeters float64) ([]DriverLocation, error)

	UpsertPendingRider(ctx context.Context, riderID string, loc Location) error
	DeletePendingRider(ctx context.Context, riderID string) error
	GetPendingRider(ctx context.Context, riderID string) (Location, error)    // returns ErrNotFound
	GetDriverLocation(ctx context.Context, driverID string) (Location, error) // returns ErrNotFound
	FindNearbyPendingRiders(ctx context.Context, center Location, radiusMeters float64, maxAge time.Duration) ([]PendingRiderHit, error)
	FindNearbyDriversWithDistance(ctx context.Context, center Location, radiusMeters float64) ([]DriverLocationHit, error)
}
