package service

import (
	"context"
	"errors"
	"testing"

	"aillion/services/matching/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLocationRepository struct {
	updateLocationFunc    func(ctx context.Context, driverID string, loc domain.Location) error
	findNearbyDriversFunc func(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error)
}

func (m *mockLocationRepository) UpdateLocation(ctx context.Context, driverID string, loc domain.Location) error {
	return m.updateLocationFunc(ctx, driverID, loc)
}

func (m *mockLocationRepository) FindNearbyDrivers(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
	return m.findNearbyDriversFunc(ctx, riderLoc, radiusMeters)
}

func TestMatcherService_MatchDriver(t *testing.T) {
	t.Run("successful match with logical positive ETA", func(t *testing.T) {
		driverID := "driver-abc"
		repo := &mockLocationRepository{
			findNearbyDriversFunc: func(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
				return []domain.DriverLocation{
					{
						DriverID: driverID,
						Location: domain.Location{Lat: 37.7750, Lng: -122.4190}, // close to rider
					},
				}, nil
			},
		}

		svc := NewMatcherService(repo)
		riderLoc := domain.Location{Lat: 37.7749, Lng: -122.4194}

		dl, eta, err := svc.MatchDriver(context.Background(), riderLoc, 2000.0)
		assert.NoError(t, err)
		require.NotNil(t, dl)
		assert.Equal(t, driverID, dl.DriverID)
		assert.True(t, eta >= 30, "ETA must be at least the realistic minimum of 30 seconds")
	})

	t.Run("no drivers found returns typed error", func(t *testing.T) {
		repo := &mockLocationRepository{
			findNearbyDriversFunc: func(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
				return []domain.DriverLocation{}, nil
			},
		}

		svc := NewMatcherService(repo)
		riderLoc := domain.Location{Lat: 37.7749, Lng: -122.4194}

		dl, eta, err := svc.MatchDriver(context.Background(), riderLoc, 2000.0)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNoDriversFound))
		assert.Nil(t, dl)
		assert.Equal(t, 0, eta)
	})

	t.Run("database error is wrapped and returned", func(t *testing.T) {
		dbErr := errors.New("timeout")
		repo := &mockLocationRepository{
			findNearbyDriversFunc: func(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
				return nil, dbErr
			},
		}

		svc := NewMatcherService(repo)
		riderLoc := domain.Location{Lat: 37.7749, Lng: -122.4194}

		dl, eta, err := svc.MatchDriver(context.Background(), riderLoc, 2000.0)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, dbErr))
		assert.Nil(t, dl)
		assert.Equal(t, 0, eta)
	})
}
