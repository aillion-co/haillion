package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayRouter_RoutingAndCORS(t *testing.T) {
	// Spin up mock downstream service
	mockDownstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users/register", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"created"}`))
	}))
	defer mockDownstream.Close()

	// Initialize Gateway
	serviceURLs := map[string]string{
		"/api/identity": mockDownstream.URL,
	}

	router, err := NewGatewayRouter(serviceURLs)
	require.NoError(t, err)

	t.Run("successful route mapping and path prefix trimming", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/identity/users/register", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, `{"status":"created"}`, rec.Body.String())
	})

	t.Run("CORS OPTIONS preflight handling", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/identity/users/register", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
	})

	t.Run("unmapped route returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/unmapped-route", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "gateway: route not found")
	})
}

func TestGatewayRouter_LongestPrefixMatching(t *testing.T) {
	mockSpecific := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("specific"))
	}))
	defer mockSpecific.Close()

	mockGeneral := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("general"))
	}))
	defer mockGeneral.Close()

	serviceURLs := map[string]string{
		"/":             mockGeneral.URL,
		"/api/identity": mockSpecific.URL,
	}

	router, err := NewGatewayRouter(serviceURLs)
	require.NoError(t, err)

	t.Run("matches specific prefix over root prefix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/identity/users", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "specific", rec.Body.String())
	})

	t.Run("matches root prefix for other routes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/any-other-route", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "general", rec.Body.String())
	})
}
