package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aillion/services/trip/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTripRepository struct {
	createFunc      func(ctx context.Context, t *domain.Trip) error
	updateStateFunc func(ctx context.Context, id string, driverID *string, newState domain.State, expectedCurrent domain.State) error
	getByIDFunc     func(ctx context.Context, id string) (*domain.Trip, error)
}

func (m *mockTripRepository) Create(ctx context.Context, t *domain.Trip) error {
	return m.createFunc(ctx, t)
}

func (m *mockTripRepository) UpdateState(ctx context.Context, id string, driverID *string, newState domain.State, expectedCurrent domain.State) error {
	return m.updateStateFunc(ctx, id, driverID, newState, expectedCurrent)
}

func (m *mockTripRepository) GetByID(ctx context.Context, id string) (*domain.Trip, error) {
	return m.getByIDFunc(ctx, id)
}

func TestHandler_CreateTrip(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		repo := &mockTripRepository{
			createFunc: func(ctx context.Context, trip *domain.Trip) error {
				assert.NotEmpty(t, trip.ID)
				assert.Equal(t, "rider-123", trip.RiderID)
				trip.State = domain.StateRequested
				return nil
			},
		}

		mux := NewServer(repo)
		body, err := json.Marshal(CreateTripRequest{RiderID: "rider-123", Lat: 37.77, Lng: -122.41})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/trips", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp TripResponse
		err = json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "rider-123", resp.RiderID)
		assert.Equal(t, domain.StateRequested, resp.State)
	})

	t.Run("missing rider id", func(t *testing.T) {
		repo := &mockTripRepository{}
		mux := NewServer(repo)
		body, err := json.Marshal(CreateTripRequest{Lat: 37.77, Lng: -122.41})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/trips", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_AcceptTrip(t *testing.T) {
	t.Run("successful acceptance", func(t *testing.T) {
		driverID := "driver-999"
		repo := &mockTripRepository{
			updateStateFunc: func(ctx context.Context, id string, dID *string, newState, expectedCurrent domain.State) error {
				assert.Equal(t, "trip-123", id)
				require.NotNil(t, dID)
				assert.Equal(t, driverID, *dID)
				assert.Equal(t, domain.StateAccepted, newState)
				assert.Equal(t, domain.StateRequested, expectedCurrent)
				return nil
			},
		}

		mux := NewServer(repo)
		body, err := json.Marshal(AcceptTripRequest{DriverID: driverID})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/trips/trip-123/accept", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("conflict error (invalid transition)", func(t *testing.T) {
		repo := &mockTripRepository{
			updateStateFunc: func(ctx context.Context, id string, dID *string, newState, expectedCurrent domain.State) error {
				return domain.ErrInvalidStateTransition
			},
		}

		mux := NewServer(repo)
		body, err := json.Marshal(AcceptTripRequest{DriverID: "driver-999"})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/trips/trip-123/accept", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid state transition or concurrent acceptance")
	})
}

func TestHandler_GetTrip(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		driverID := "driver-777"
		expectedTrip := &domain.Trip{
			ID:       "trip-111",
			RiderID:  "rider-222",
			DriverID: &driverID,
			State:    domain.StateInProgress,
		}

		repo := &mockTripRepository{
			getByIDFunc: func(ctx context.Context, id string) (*domain.Trip, error) {
				assert.Equal(t, "trip-111", id)
				return expectedTrip, nil
			},
		}

		mux := NewServer(repo)
		req := httptest.NewRequest("GET", "/trips/trip-111", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp TripResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expectedTrip.ID, resp.ID)
		assert.Equal(t, expectedTrip.RiderID, resp.RiderID)
		require.NotNil(t, resp.DriverID)
		assert.Equal(t, driverID, *resp.DriverID)
		assert.Equal(t, domain.StateInProgress, resp.State)
	})
}
