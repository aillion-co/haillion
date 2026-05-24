package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"haillion/services/matching/internal/domain"
)

var ErrNoDriversFound = errors.New("no drivers found nearby")

type MatcherService struct {
	repo domain.LocationRepository
}

func NewMatcherService(repo domain.LocationRepository) *MatcherService {
	return &MatcherService{repo: repo}
}

func (s *MatcherService) MatchDriver(ctx context.Context, riderLoc domain.Location, radiusMeters float64) (*domain.DriverLocation, int, error) {
	drivers, err := s.repo.FindNearbyDrivers(ctx, riderLoc, radiusMeters)
	if err != nil {
		return nil, 0, fmt.Errorf("find nearby drivers: %w", err)
	}

	if len(drivers) == 0 {
		return nil, 0, ErrNoDriversFound
	}

	// PostGIS ST_Distance sorted results in ASC order, so the closest is drivers[0]
	closest := drivers[0]

	// Compute basic flat-surface distance in meters as an ETA heuristic
	// Lat/Lng in degrees to meters: 1 degree ~ 111,000 meters
	latRad := riderLoc.Lat * math.Pi / 180.0
	dx := (closest.Location.Lng - riderLoc.Lng) * 111000.0 * math.Cos(latRad)
	dy := (closest.Location.Lat - riderLoc.Lat) * 111000.0
	distanceMeters := math.Hypot(dx, dy)

	// Average driver speed: 15 meters / second (~54 km/h)
	speedMetersPerSecond := 15.0
	etaSeconds := int(math.Round(distanceMeters / speedMetersPerSecond))

	// Enforce a sensible minimum ETA of 30 seconds for realism
	if etaSeconds < 30 {
		etaSeconds = 30
	}

	return &closest, etaSeconds, nil
}
