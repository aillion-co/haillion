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
