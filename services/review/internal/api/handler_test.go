package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"haillion/services/review/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockReviewRepository struct {
	createFunc          func(ctx context.Context, r *domain.Review) error
	getAverageScoreFunc func(ctx context.Context, subjectID string) (float64, error)
}

func (m *mockReviewRepository) Create(ctx context.Context, r *domain.Review) error {
	return m.createFunc(ctx, r)
}

func (m *mockReviewRepository) GetAverageScore(ctx context.Context, subjectID string) (float64, error) {
	return m.getAverageScoreFunc(ctx, subjectID)
}

func TestHandler_CreateReview(t *testing.T) {
	t.Run("successful review creation", func(t *testing.T) {
		repo := &mockReviewRepository{
			createFunc: func(ctx context.Context, r *domain.Review) error {
				assert.NotEmpty(t, r.ID)
				assert.Equal(t, "trip-123", r.TripID)
				assert.Equal(t, "user-reviewer", r.ReviewerID)
				assert.Equal(t, "user-subject", r.SubjectID)
				assert.Equal(t, 5, r.Score)
				assert.Equal(t, "Super!", r.Comment)
				return nil
			},
		}

		mux := NewServer(repo)
		body, err := json.Marshal(CreateReviewRequest{
			TripID:    "trip-123",
			SubjectID: "user-subject",
			Score:     5,
			Comment:   "Super!",
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/reviews", bytes.NewReader(body))
		req.Header.Set("X-User-ID", "user-reviewer")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp ReviewResponse
		err = json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "trip-123", resp.TripID)
		assert.Equal(t, "user-reviewer", resp.ReviewerID)
		assert.Equal(t, "user-subject", resp.SubjectID)
		assert.Equal(t, 5, resp.Score)
		assert.Equal(t, "Super!", resp.Comment)
	})

	t.Run("missing X-User-ID header (unauthorized)", func(t *testing.T) {
		repo := &mockReviewRepository{}
		mux := NewServer(repo)
		body, err := json.Marshal(CreateReviewRequest{
			TripID:    "trip-123",
			SubjectID: "user-subject",
			Score:     5,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/reviews", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid score ranges (0 and 6 boundary tests)", func(t *testing.T) {
		repo := &mockReviewRepository{}
		mux := NewServer(repo)

		scores := []int{0, 6}
		for _, score := range scores {
			body, err := json.Marshal(CreateReviewRequest{
				TripID:    "trip-123",
				SubjectID: "user-subject",
				Score:     score,
			})
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/reviews", bytes.NewReader(body))
			req.Header.Set("X-User-ID", "user-reviewer")
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "score must be between 1 and 5")
		}
	})

	t.Run("conflict error (duplicate review)", func(t *testing.T) {
		repo := &mockReviewRepository{
			createFunc: func(ctx context.Context, r *domain.Review) error {
				return domain.ErrDuplicateReview
			},
		}

		mux := NewServer(repo)
		body, err := json.Marshal(CreateReviewRequest{
			TripID:    "trip-123",
			SubjectID: "user-subject",
			Score:     4,
		})
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/reviews", bytes.NewReader(body))
		req.Header.Set("X-User-ID", "user-reviewer")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "review already exists for this trip")
	})
}

func TestHandler_GetRating(t *testing.T) {
	t.Run("successful rating retrieve", func(t *testing.T) {
		repo := &mockReviewRepository{
			getAverageScoreFunc: func(ctx context.Context, subjectID string) (float64, error) {
				assert.Equal(t, "user-subject-123", subjectID)
				return 4.2, nil
			},
		}

		mux := NewServer(repo)
		req := httptest.NewRequest("GET", "/users/user-subject-123/rating", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var resp RatingResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.InDelta(t, 4.2, resp.Average, 0.001)
	})
}
