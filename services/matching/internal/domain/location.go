package domain

import "context"

type Location struct {
	Lat float64
	Lng float64
}

type DriverLocation struct {
	DriverID string
	Location Location
}

type LocationRepository interface {
	UpdateLocation(ctx context.Context, driverID string, loc Location) error
	FindNearbyDrivers(ctx context.Context, riderLoc Location, radiusMeters float64) ([]DriverLocation, error)
}
