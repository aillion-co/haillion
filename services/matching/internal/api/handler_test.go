package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aillion/services/matching/internal/domain"
	"aillion/services/matching/internal/service"

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

func TestHandler_UpdateLocation(t *testing.T) {
	t.Run("successful location update", func(t *testing.T) {
		repo := &mockLocationRepository{
			updateLocationFunc: func(ctx context.Context, driverID string, loc domain.Location) error {
				assert.Equal(t, "driver-1", driverID)
				assert.Equal(t, 37.7749, loc.Lat)
				assert.Equal(t, -122.4194, loc.Lng)
				return nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))
		body, err := json.Marshal(UpdateLocationRequest{Lat: 37.7749, Lng: -122.4194})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/drivers/driver-1/location", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid latitude", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))
		body, err := json.Marshal(UpdateLocationRequest{Lat: 100.0, Lng: -122.4194})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/drivers/driver-1/location", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid coordinate range")
	})
}

func TestHandler_Match(t *testing.T) {
	t.Run("successful driver match", func(t *testing.T) {
		driverID := "driver-ok"
		repo := &mockLocationRepository{
			findNearbyDriversFunc: func(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
				return []domain.DriverLocation{
					{
						DriverID: driverID,
						Location: domain.Location{Lat: 37.7750, Lng: -122.4190},
					},
				}, nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))
		body, err := json.Marshal(MatchRequest{
			RiderID: "rider-1",
			Lat:     37.7749,
			Lng:     -122.4194,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/match", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp MatchResponse
		err = json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, driverID, resp.DriverID)
		assert.True(t, resp.ETA >= 30)
	})

	t.Run("no available drivers found", func(t *testing.T) {
		repo := &mockLocationRepository{
			findNearbyDriversFunc: func(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
				return []domain.DriverLocation{}, nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))
		body, err := json.Marshal(MatchRequest{
			RiderID: "rider-1",
			Lat:     37.7749,
			Lng:     -122.4194,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/match", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "no available drivers found")
	})
}
