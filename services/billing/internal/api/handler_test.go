package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"haillion/services/billing/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPaymentRepository struct {
	createFunc   func(ctx context.Context, p *domain.Payment) error
	markPaidFunc func(ctx context.Context, id string) error
	getByIDFunc  func(ctx context.Context, id string) (*domain.Payment, error)
}

func (m *mockPaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	return m.createFunc(ctx, p)
}

func (m *mockPaymentRepository) MarkPaid(ctx context.Context, id string) error {
	return m.markPaidFunc(ctx, id)
}

func (m *mockPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	return m.getByIDFunc(ctx, id)
}

func TestHandler_CreatePayment(t *testing.T) {
	t.Run("successful payment creation", func(t *testing.T) {
		repo := &mockPaymentRepository{
			createFunc: func(ctx context.Context, p *domain.Payment) error {
				assert.NotEmpty(t, p.ID)
				assert.Equal(t, "trip-abc", p.TripID)
				assert.Equal(t, int64(1500), p.Amount)
				p.State = domain.PaymentStatePending
				return nil
			},
		}

		mux := NewServer(repo)
		body, err := json.Marshal(CreatePaymentRequest{TripID: "trip-abc", Amount: 1500})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/payments", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp PaymentResponse
		err = json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "trip-abc", resp.TripID)
		assert.Equal(t, int64(1500), resp.Amount)
		assert.Equal(t, domain.PaymentStatePending, resp.State)
	})

	t.Run("invalid request fields", func(t *testing.T) {
		repo := &mockPaymentRepository{}
		mux := NewServer(repo)
		body, err := json.Marshal(CreatePaymentRequest{Amount: -100}) // negative and empty ID
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/payments", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_ProcessPayment(t *testing.T) {
	t.Run("successful processing", func(t *testing.T) {
		repo := &mockPaymentRepository{
			markPaidFunc: func(ctx context.Context, id string) error {
				assert.Equal(t, "pay-123", id)
				return nil
			},
		}

		mux := NewServer(repo)
		req := httptest.NewRequest("POST", "/payments/pay-123/process", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("payment not found", func(t *testing.T) {
		repo := &mockPaymentRepository{
			markPaidFunc: func(ctx context.Context, id string) error {
				return domain.ErrPaymentNotFound
			},
		}

		mux := NewServer(repo)
		req := httptest.NewRequest("POST", "/payments/unknown/process", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHandler_GetSurge(t *testing.T) {
	t.Run("successful calculation", func(t *testing.T) {
		repo := &mockPaymentRepository{}
		mux := NewServer(repo)

		req := httptest.NewRequest("GET", "/surge?riders=10&drivers=5", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp SurgeResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.InDelta(t, 1.2, resp.Multiplier, 0.001)
	})

	t.Run("invalid query params", func(t *testing.T) {
		repo := &mockPaymentRepository{}
		mux := NewServer(repo)

		req := httptest.NewRequest("GET", "/surge?riders=-1&drivers=abc", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_GetFareEstimate(t *testing.T) {
	repo := &mockPaymentRepository{}
	mux := NewServer(repo)

	t.Run("successful fare estimate with surge", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/fare/estimate?from=SW1A&to=EC1A&riders=10&drivers=5", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp domain.FareBreakdown
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)

		// SW1A centroid: Lat: 51.502, Lng: -0.13386
		// EC1A centroid: Lat: 51.5202, Lng: -0.104412
		// Distance is around 1.78 miles.
		assert.InDelta(t, 1.78, resp.DistanceMiles, 0.01)
		assert.Equal(t, int64(178), resp.BasePence)
		assert.InDelta(t, 1.2, resp.SurgeMultiplier, 0.01)
		// total = round(183 * 1.2) = round(219.6) = 220. Minimum is 250, so total is 250 and minimum is applied.
		assert.Equal(t, int64(250), resp.TotalPence)
		assert.True(t, resp.MinimumApplied)
	})

	t.Run("missing query parameters", func(t *testing.T) {
		tests := []struct {
			name string
			url  string
		}{
			{"missing from", "/fare/estimate?to=EC1A"},
			{"missing to", "/fare/estimate?from=SW1A"},
			{"missing both", "/fare/estimate"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tc.url, nil)
				rec := httptest.NewRecorder()

				mux.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Contains(t, rec.Body.String(), "missing query parameter: from and to required")
			})
		}
	})

	t.Run("invalid postcode format", func(t *testing.T) {
		tests := []struct {
			name string
			url  string
		}{
			{"from invalid", "/fare/estimate?from=123&to=EC1A"},
			{"to invalid", "/fare/estimate?from=SW1A&to=SW1A1"},
			{"both invalid", "/fare/estimate?from=123&to=SW1A1"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tc.url, nil)
				rec := httptest.NewRecorder()

				mux.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Contains(t, rec.Body.String(), "invalid postcode")
			})
		}
	})

	t.Run("postcode not found", func(t *testing.T) {
		tests := []struct {
			name string
			url  string
		}{
			{"from unknown", "/fare/estimate?from=ZZ99&to=EC1A"},
			{"to unknown", "/fare/estimate?from=SW1A&to=ZZ99"},
			{"both unknown", "/fare/estimate?from=ZZ99&to=AA1"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tc.url, nil)
				rec := httptest.NewRecorder()

				mux.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusNotFound, rec.Code)
				assert.Contains(t, rec.Body.String(), "postcode not found")
			})
		}
	})

	t.Run("invalid riders/drivers parameter", func(t *testing.T) {
		tests := []struct {
			name string
			url  string
		}{
			{"riders negative", "/fare/estimate?from=SW1A&to=EC1A&riders=-1"},
			{"riders unparseable", "/fare/estimate?from=SW1A&to=EC1A&riders=abc"},
			{"drivers negative", "/fare/estimate?from=SW1A&to=EC1A&drivers=-5"},
			{"drivers unparseable", "/fare/estimate?from=SW1A&to=EC1A&drivers=xyz"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tc.url, nil)
				rec := httptest.NewRecorder()

				mux.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Contains(t, rec.Body.String(), "invalid riders/drivers parameter")
			})
		}
	})

	t.Run("defaults for riders and drivers if absent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/fare/estimate?from=SW1A&to=EC1A", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp domain.FareBreakdown
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)

		assert.Equal(t, 1.0, resp.SurgeMultiplier)
	})

	t.Run("same centroid", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/fare/estimate?from=SW1A&to=SW1A", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp domain.FareBreakdown
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)

		assert.Equal(t, 0.0, resp.DistanceMiles)
		assert.Equal(t, int64(0), resp.BasePence)
		assert.Equal(t, int64(250), resp.TotalPence)
		assert.True(t, resp.MinimumApplied)
	})

	t.Run("decision table combinations", func(t *testing.T) {
		type testCase struct {
			name         string
			from         string
			to           string
			expectedCode int
			expectedBody string
		}

		combinations := []testCase{
			{"OK x OK", "SW1A", "EC1A", http.StatusOK, ""},
			{"OK x Unknown", "SW1A", "ZZ99", http.StatusNotFound, "postcode not found"},
			{"OK x Malformed", "SW1A", "123", http.StatusBadRequest, "invalid postcode"},

			{"Unknown x OK", "ZZ99", "EC1A", http.StatusNotFound, "postcode not found"},
			{"Unknown x Unknown", "ZZ99", "AA1", http.StatusNotFound, "postcode not found"},
			{"Unknown x Malformed", "ZZ99", "123", http.StatusNotFound, "postcode not found"},

			{"Malformed x OK", "123", "EC1A", http.StatusBadRequest, "invalid postcode"},
			{"Malformed x Unknown", "123", "ZZ99", http.StatusBadRequest, "invalid postcode"},
			{"Malformed x Malformed", "123", "SW1A1", http.StatusBadRequest, "invalid postcode"},
		}

		for _, tc := range combinations {
			t.Run(tc.name, func(t *testing.T) {
				url := "/fare/estimate?from=" + tc.from + "&to=" + tc.to
				req := httptest.NewRequest("GET", url, nil)
				rec := httptest.NewRecorder()

				mux.ServeHTTP(rec, req)

				assert.Equal(t, tc.expectedCode, rec.Code)
				if tc.expectedBody != "" {
					assert.Contains(t, rec.Body.String(), tc.expectedBody)
				}
			})
		}
	})
}
