package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aillion/services/identity/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepository struct {
	createFunc  func(ctx context.Context, u *domain.User) error
	getByIDFunc func(ctx context.Context, id string) (*domain.User, error)
}

func (m *mockUserRepository) Create(ctx context.Context, u *domain.User) error {
	return m.createFunc(ctx, u)
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return m.getByIDFunc(ctx, id)
}

func TestHandler_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		repo := &mockUserRepository{
			createFunc: func(ctx context.Context, u *domain.User) error {
				assert.Equal(t, "user-1", u.ID)
				assert.Equal(t, "rider@example.com", u.Email)
				assert.Equal(t, domain.RoleRider, u.Role)
				return nil
			},
		}

		mux := NewServer(repo)
		reqBody, err := json.Marshal(RegisterRequest{
			ID:    "user-1",
			Email: "rider@example.com",
			Role:  domain.RoleRider,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/users/register", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp UserResponse
		err = json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, "user-1", resp.ID)
		assert.Equal(t, "rider@example.com", resp.Email)
		assert.Equal(t, domain.RoleRider, resp.Role)
	})

	t.Run("conflict error (duplicate email)", func(t *testing.T) {
		repo := &mockUserRepository{
			createFunc: func(ctx context.Context, u *domain.User) error {
				return domain.ErrDuplicateEmail
			},
		}

		mux := NewServer(repo)
		reqBody, err := json.Marshal(RegisterRequest{
			ID:    "user-1",
			Email: "duplicate@example.com",
			Role:  domain.RoleDriver,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/users/register", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "email already registered")
	})

	t.Run("bad request (invalid fields)", func(t *testing.T) {
		repo := &mockUserRepository{}
		mux := NewServer(repo)

		tests := []struct {
			name string
			req  RegisterRequest
		}{
			{
				name: "empty id",
				req:  RegisterRequest{Email: "test@example.com", Role: domain.RoleRider},
			},
			{
				name: "empty email",
				req:  RegisterRequest{ID: "id", Role: domain.RoleRider},
			},
			{
				name: "invalid role",
				req:  RegisterRequest{ID: "id", Email: "test@example.com", Role: domain.Role("invalid")},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				reqBody, err := json.Marshal(tc.req)
				require.NoError(t, err)

				req := httptest.NewRequest("POST", "/users/register", bytes.NewReader(reqBody))
				rec := httptest.NewRecorder()

				mux.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Contains(t, rec.Body.String(), "missing or invalid fields")
			})
		}
	})

	t.Run("bad request (invalid json)", func(t *testing.T) {
		repo := &mockUserRepository{}
		mux := NewServer(repo)

		req := httptest.NewRequest("POST", "/users/register", bytes.NewReader([]byte("{invalid-json}")))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid request body")
	})
}

func TestHandler_GetUser(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		expectedUser := &domain.User{
			ID:    "user-123",
			Email: "driver@example.com",
			Role:  domain.RoleDriver,
		}

		repo := &mockUserRepository{
			getByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
				assert.Equal(t, "user-123", id)
				return expectedUser, nil
			},
		}

		mux := NewServer(repo)
		req := httptest.NewRequest("GET", "/users/user-123", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp UserResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.ID, resp.ID)
		assert.Equal(t, expectedUser.Email, resp.Email)
		assert.Equal(t, expectedUser.Role, resp.Role)
	})

	t.Run("not found error", func(t *testing.T) {
		repo := &mockUserRepository{
			getByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		mux := NewServer(repo)
		req := httptest.NewRequest("GET", "/users/unknown", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "user not found")
	})
}
