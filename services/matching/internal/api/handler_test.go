package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"haillion/services/matching/internal/domain"
	"haillion/services/matching/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLocationRepository struct {
	domain.LocationRepository
	updateLocationFunc    func(ctx context.Context, driverID string, loc domain.Location) error
	findNearbyDriversFunc func(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error)

	upsertPendingRiderFunc            func(ctx context.Context, riderID string, loc domain.Location) error
	deletePendingRiderFunc            func(ctx context.Context, riderID string) error
	getPendingRiderFunc               func(ctx context.Context, riderID string) (domain.Location, error)
	getDriverLocationFunc             func(ctx context.Context, driverID string) (domain.Location, error)
	findNearbyPendingRidersFunc       func(ctx context.Context, center domain.Location, radiusMeters float64, maxAge time.Duration) ([]domain.PendingRiderHit, error)
	findNearbyDriversWithDistanceFunc func(ctx context.Context, center domain.Location, radiusMeters float64) ([]domain.DriverLocationHit, error)
}

func (m *mockLocationRepository) UpdateLocation(ctx context.Context, driverID string, loc domain.Location) error {
	return m.updateLocationFunc(ctx, driverID, loc)
}

func (m *mockLocationRepository) FindNearbyDrivers(ctx context.Context, riderLoc domain.Location, radiusMeters float64) ([]domain.DriverLocation, error) {
	return m.findNearbyDriversFunc(ctx, riderLoc, radiusMeters)
}

func (m *mockLocationRepository) UpsertPendingRider(ctx context.Context, riderID string, loc domain.Location) error {
	return m.upsertPendingRiderFunc(ctx, riderID, loc)
}

func (m *mockLocationRepository) DeletePendingRider(ctx context.Context, riderID string) error {
	return m.deletePendingRiderFunc(ctx, riderID)
}

func (m *mockLocationRepository) GetPendingRider(ctx context.Context, riderID string) (domain.Location, error) {
	return m.getPendingRiderFunc(ctx, riderID)
}

func (m *mockLocationRepository) GetDriverLocation(ctx context.Context, driverID string) (domain.Location, error) {
	return m.getDriverLocationFunc(ctx, driverID)
}

func (m *mockLocationRepository) FindNearbyPendingRiders(ctx context.Context, center domain.Location, radiusMeters float64, maxAge time.Duration) ([]domain.PendingRiderHit, error) {
	return m.findNearbyPendingRidersFunc(ctx, center, radiusMeters, maxAge)
}

func (m *mockLocationRepository) FindNearbyDriversWithDistance(ctx context.Context, center domain.Location, radiusMeters float64) ([]domain.DriverLocationHit, error) {
	return m.findNearbyDriversWithDistanceFunc(ctx, center, radiusMeters)
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

func TestHandler_RiderRequest(t *testing.T) {
	t.Run("successful request with postcode SW1A", func(t *testing.T) {
		repo := &mockLocationRepository{
			upsertPendingRiderFunc: func(ctx context.Context, riderID string, loc domain.Location) error {
				assert.Equal(t, "rider-123", riderID)
				assert.InDelta(t, 51.502, loc.Lat, 0.001)
				assert.InDelta(t, -0.13386, loc.Lng, 0.001)
				return nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))
		body := []byte(`{"postcode":"SW1A"}`)

		req := httptest.NewRequest("POST", "/riders/rider-123/request", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid postcode", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))
		body := []byte(`{"postcode":"ZZ99"}`)

		req := httptest.NewRequest("POST", "/riders/rider-123/request", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "postcode not found")
	})

	t.Run("missing location body", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))
		body := []byte(`{}`)

		req := httptest.NewRequest("POST", "/riders/rider-123/request", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "missing location: provide postcode or lat/lng")
	})
}

func TestHandler_RiderCancel(t *testing.T) {
	t.Run("successful request cancel", func(t *testing.T) {
		repo := &mockLocationRepository{
			deletePendingRiderFunc: func(ctx context.Context, riderID string) error {
				assert.Equal(t, "rider-123", riderID)
				return nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))

		req := httptest.NewRequest("DELETE", "/riders/rider-123/request", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestHandler_NearbyRiders(t *testing.T) {
	t.Run("successful nearby riders returning data", func(t *testing.T) {
		repo := &mockLocationRepository{
			getDriverLocationFunc: func(ctx context.Context, driverID string) (domain.Location, error) {
				assert.Equal(t, "driver-123", driverID)
				return domain.Location{Lat: 51.5074, Lng: -0.1278}, nil
			},
			findNearbyPendingRidersFunc: func(ctx context.Context, center domain.Location, radiusMeters float64, maxAge time.Duration) ([]domain.PendingRiderHit, error) {
				assert.Equal(t, 51.5074, center.Lat)
				assert.Equal(t, 2500.0, radiusMeters)
				assert.Equal(t, 10*time.Minute, maxAge)
				return []domain.PendingRiderHit{
					{
						RiderID:        "rider-a",
						Location:       domain.Location{Lat: 51.5080, Lng: -0.1270},
						DistanceMeters: 150.0,
					},
				}, nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))

		req := httptest.NewRequest("GET", "/drivers/driver-123/nearby-riders?radius_m=2500", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp NearbyRidersResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		require.Len(t, resp.Riders, 1)
		assert.Equal(t, "rider-a", resp.Riders[0].RiderID)
		assert.Equal(t, 150.0, resp.Riders[0].DistanceM)
	})

	t.Run("default radius to 5000", func(t *testing.T) {
		repo := &mockLocationRepository{
			getDriverLocationFunc: func(ctx context.Context, driverID string) (domain.Location, error) {
				return domain.Location{Lat: 51.5074, Lng: -0.1278}, nil
			},
			findNearbyPendingRidersFunc: func(ctx context.Context, center domain.Location, radiusMeters float64, maxAge time.Duration) ([]domain.PendingRiderHit, error) {
				assert.Equal(t, 5000.0, radiusMeters)
				return []domain.PendingRiderHit{}, nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))

		req := httptest.NewRequest("GET", "/drivers/driver-123/nearby-riders", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("driver location unknown returns 404", func(t *testing.T) {
		repo := &mockLocationRepository{
			getDriverLocationFunc: func(ctx context.Context, driverID string) (domain.Location, error) {
				return domain.Location{}, domain.ErrNotFound
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))

		req := httptest.NewRequest("GET", "/drivers/driver-123/nearby-riders", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "driver location unknown")
	})

	t.Run("invalid radius returns 400", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))

		invalidRadii := []string{"-10", "0", "50001", "abc"}
		for _, rad := range invalidRadii {
			req := httptest.NewRequest("GET", "/drivers/driver-123/nearby-riders?radius_m="+rad, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "for radius_m=%s", rad)
		}
	})
}

func TestHandler_NearbyDrivers(t *testing.T) {
	t.Run("successful nearby drivers returning data", func(t *testing.T) {
		repo := &mockLocationRepository{
			getPendingRiderFunc: func(ctx context.Context, riderID string) (domain.Location, error) {
				assert.Equal(t, "rider-123", riderID)
				return domain.Location{Lat: 51.5074, Lng: -0.1278}, nil
			},
			findNearbyDriversWithDistanceFunc: func(ctx context.Context, center domain.Location, radiusMeters float64) ([]domain.DriverLocationHit, error) {
				assert.Equal(t, 51.5074, center.Lat)
				assert.Equal(t, 2500.0, radiusMeters)
				return []domain.DriverLocationHit{
					{
						DriverID:       "driver-a",
						Location:       domain.Location{Lat: 51.5080, Lng: -0.1270},
						DistanceMeters: 150.0,
					},
				}, nil
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))

		req := httptest.NewRequest("GET", "/riders/rider-123/nearby-drivers?radius_m=2500", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp NearbyDriversResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		require.Len(t, resp.Drivers, 1)
		assert.Equal(t, "driver-a", resp.Drivers[0].DriverID)
		assert.Equal(t, 150.0, resp.Drivers[0].DistanceM)
	})

	t.Run("rider has no active request returns 404", func(t *testing.T) {
		repo := &mockLocationRepository{
			getPendingRiderFunc: func(ctx context.Context, riderID string) (domain.Location, error) {
				return domain.Location{}, domain.ErrNotFound
			},
		}

		mux := NewServer(repo, service.NewMatcherService(repo))

		req := httptest.NewRequest("GET", "/riders/rider-123/nearby-drivers", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "rider has no active request")
	})

	t.Run("invalid radius returns 400", func(t *testing.T) {
		repo := &mockLocationRepository{}
		mux := NewServer(repo, service.NewMatcherService(repo))

		invalidRadii := []string{"-10", "0", "50001", "abc"}
		for _, rad := range invalidRadii {
			req := httptest.NewRequest("GET", "/riders/rider-123/nearby-drivers?radius_m="+rad, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "for radius_m=%s", rad)
		}
	})
}
