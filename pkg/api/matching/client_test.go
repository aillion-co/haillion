package matching

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("empty URL returns error", func(t *testing.T) {
		client, err := NewClient("", nil)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "base URL cannot be empty")
	})

	t.Run("valid URL returns client", func(t *testing.T) {
		client, err := NewClient("http://localhost:8080", nil)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})
}

func TestMatchingClient_RequestRide(t *testing.T) {
	t.Run("successful ride request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/riders/rider-123/request", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var req RiderRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Equal(t, "SW1A", req.Postcode)
			assert.Nil(t, req.Lat)
			assert.Nil(t, req.Lng)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		err = client.RequestRide(context.Background(), "rider-123", "SW1A")
		assert.NoError(t, err)
	})

	t.Run("non-2xx status code error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`invalid postcode lookup failure`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		err = client.RequestRide(context.Background(), "rider-123", "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 400")
		assert.Contains(t, err.Error(), "invalid postcode lookup failure")
	})

	t.Run("request timeout cancellation safety", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		err = client.RequestRide(ctx, "rider-123", "SW1A")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), context.DeadlineExceeded.Error())
	})
}

func TestMatchingClient_CancelRide(t *testing.T) {
	t.Run("successful ride cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "DELETE", r.Method)
			assert.Equal(t, "/riders/rider-123/request", r.URL.Path)

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		err = client.CancelRide(context.Background(), "rider-123")
		assert.NoError(t, err)
	})

	t.Run("non-2xx status code error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`failed to delete pending rider request`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		err = client.CancelRide(context.Background(), "rider-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
		assert.Contains(t, err.Error(), "failed to delete pending rider request")
	})
}

func TestMatchingClient_NearbyRiders(t *testing.T) {
	t.Run("successful query returning riders", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/drivers/driver-123/nearby-riders", r.URL.Path)
			assert.Equal(t, "2500.000000", r.URL.Query().Get("radius_m"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"riders": [
					{
						"rider_id": "rider-a",
						"lat": 51.5080,
						"lng": -0.1270,
						"distance_m": 150.0
					}
				]
			}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		riders, err := client.NearbyRiders(context.Background(), "driver-123", 2500)
		assert.NoError(t, err)
		require.Len(t, riders, 1)
		assert.Equal(t, "rider-a", riders[0].RiderID)
		assert.InDelta(t, 51.5080, riders[0].Lat, 0.001)
		assert.InDelta(t, -0.1270, riders[0].Lng, 0.001)
		assert.InDelta(t, 150.0, riders[0].DistanceM, 0.001)
	})

	t.Run("non-2xx status code error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`driver location unknown`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		_, err = client.NearbyRiders(context.Background(), "driver-123", 2500)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 404")
		assert.Contains(t, err.Error(), "driver location unknown")
	})
}

func TestMatchingClient_NearbyDrivers(t *testing.T) {
	t.Run("successful query returning drivers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/riders/rider-123/nearby-drivers", r.URL.Path)
			assert.Equal(t, "5000.000000", r.URL.Query().Get("radius_m"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"drivers": [
					{
						"driver_id": "driver-a",
						"lat": 51.5080,
						"lng": -0.1270,
						"distance_m": 150.0
					}
				]
			}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		drivers, err := client.NearbyDrivers(context.Background(), "rider-123", 5000)
		assert.NoError(t, err)
		require.Len(t, drivers, 1)
		assert.Equal(t, "driver-a", drivers[0].DriverID)
		assert.InDelta(t, 51.5080, drivers[0].Lat, 0.001)
		assert.InDelta(t, -0.1270, drivers[0].Lng, 0.001)
		assert.InDelta(t, 150.0, drivers[0].DistanceM, 0.001)
	})

	t.Run("non-2xx status code error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`rider has no active request`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, nil)
		assert.NoError(t, err)

		_, err = client.NearbyDrivers(context.Background(), "rider-123", 5000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 404")
		assert.Contains(t, err.Error(), "rider has no active request")
	})
}
