package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"haillion/services/matching/internal/domain"
	"haillion/services/matching/internal/service"

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

func ptr(f float64) *float64 {
	return &f
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
		body, err := json.Marshal(UpdateLocationRequest{Lat: ptr(37.7749), Lng: ptr(-122.4194)})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/drivers/driver-1/location", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid latitude", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))
		body, err := json.Marshal(UpdateLocationRequest{Lat: ptr(100.0), Lng: ptr(-122.4194)})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/drivers/driver-1/location", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "missing location: provide postcode or lat/lng")
	})

	// LOCAL-11 acceptance criteria:
	t.Run("postcode only SW1A successful update", func(t *testing.T) {
		repo := &mockLocationRepository{
			updateLocationFunc: func(ctx context.Context, driverID string, loc domain.Location) error {
				assert.Equal(t, "driver-1", driverID)
				// SW1A centroid is Lat: 51.502, Lng: -0.13386
				assert.InDelta(t, 51.502, loc.Lat, 0.001)
				assert.InDelta(t, -0.13386, loc.Lng, 0.001)
				return nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))
		// Raw JSON body containing only postcode
		body := []byte(`{"postcode":"SW1A"}`)

		req := httptest.NewRequest("POST", "/drivers/driver-1/location", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("prefer explicit lat/lng over postcode", func(t *testing.T) {
		repo := &mockLocationRepository{
			updateLocationFunc: func(ctx context.Context, driverID string, loc domain.Location) error {
				assert.Equal(t, "driver-1", driverID)
				// Explicit coords must be preferred
				assert.Equal(t, 37.7749, loc.Lat)
				assert.Equal(t, -122.4194, loc.Lng)
				return nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))
		body := []byte(`{"lat":37.7749, "lng":-122.4194, "postcode":"SW1A"}`)

		req := httptest.NewRequest("POST", "/drivers/driver-1/location", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("neither valid lat/lng nor postcode provided", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))

		// Raw JSON missing both
		body := []byte(`{}`)

		req := httptest.NewRequest("POST", "/drivers/driver-1/location", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "missing location: provide postcode or lat/lng")
	})

	t.Run("unknown postcode", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))

		body := []byte(`{"postcode":"ZZ99"}`)

		req := httptest.NewRequest("POST", "/drivers/driver-1/location", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "postcode not found")
	})

	t.Run("malformed postcode", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))

		body := []byte(`{"postcode":"123"}`)

		req := httptest.NewRequest("POST", "/drivers/driver-1/location", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid postcode")
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
			Lat:     ptr(37.7749),
			Lng:     ptr(-122.4194),
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
			Lat:     ptr(37.7749),
			Lng:     ptr(-122.4194),
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/match", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "no available drivers found")
	})

	// LOCAL-11 acceptance criteria:
	t.Run("postcode only SW1A successful match", func(t *testing.T) {
		driverID := "driver-sw1a"
		repo := &mockLocationRepository{
			findNearbyDriversFunc: func(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
				// SW1A centroid
				assert.InDelta(t, 51.502, riderLoc.Lat, 0.001)
				assert.InDelta(t, -0.13386, riderLoc.Lng, 0.001)
				return []domain.DriverLocation{
					{
						DriverID: driverID,
						Location: domain.Location{Lat: 51.503, Lng: -0.134},
					},
				}, nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))
		body := []byte(`{"rider_id":"rider-1","postcode":"SW1A"}`)

		req := httptest.NewRequest("POST", "/match", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp MatchResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, driverID, resp.DriverID)
	})

	t.Run("postcode only SW1A no drivers match", func(t *testing.T) {
		repo := &mockLocationRepository{
			findNearbyDriversFunc: func(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
				return []domain.DriverLocation{}, nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))
		body := []byte(`{"rider_id":"rider-1","postcode":"SW1A"}`)

		req := httptest.NewRequest("POST", "/match", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "no available drivers found")
	})

	t.Run("neither valid lat/lng nor postcode provided for match", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))

		body := []byte(`{"rider_id":"rider-1"}`)

		req := httptest.NewRequest("POST", "/match", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "missing location: provide postcode or lat/lng")
	})
}
