package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"aillion/services/review/internal/domain"

	"github.com/google/uuid"
)

type ReviewServer struct {
	repo domain.ReviewRepository
}

type CreateReviewRequest struct {
	TripID    string `json:"trip_id"`
	SubjectID string `json:"subject_id"`
	Score     int    `json:"score"`
	Comment   string `json:"comment"`
}

type ReviewResponse struct {
	ID         string `json:"id"`
	TripID     string `json:"trip_id"`
	ReviewerID string `json:"reviewer_id"`
	SubjectID  string `json:"subject_id"`
	Score      int    `json:"score"`
	Comment    string `json:"comment"`
}

type RatingResponse struct {
	Average float64 `json:"average"`
}

func NewServer(repo domain.ReviewRepository) *http.ServeMux {
	server := &ReviewServer{repo: repo}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /reviews", server.handleCreateReview)
	mux.HandleFunc("GET /users/{id}/rating", server.handleGetRating)

	return mux
}

func (s *ReviewServer) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	reviewerID := r.Header.Get("X-User-ID")
	if reviewerID == "" {
		http.Error(w, "unauthorized: missing X-User-ID header", http.StatusUnauthorized)
		return
	}

	var req CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.TripID == "" || req.SubjectID == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	if req.Score < 1 || req.Score > 5 {
		http.Error(w, "score must be between 1 and 5", http.StatusBadRequest)
		return
	}

	rev := &domain.Review{
		ID:         uuid.New().String(),
		TripID:     req.TripID,
		ReviewerID: reviewerID,
		SubjectID:  req.SubjectID,
		Score:      req.Score,
		Comment:    req.Comment,
	}

	if err := s.repo.Create(r.Context(), rev); err != nil {
		if errors.Is(err, domain.ErrInvalidScore) {
			http.Error(w, "invalid score range", http.StatusBadRequest)
			return
		}
		if errors.Is(err, domain.ErrDuplicateReview) {
			http.Error(w, "review already exists for this trip", http.StatusConflict)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ReviewResponse{
		ID:         rev.ID,
		TripID:     rev.TripID,
		ReviewerID: rev.ReviewerID,
		SubjectID:  rev.SubjectID,
		Score:      rev.Score,
		Comment:    rev.Comment,
	})
}

func (s *ReviewServer) handleGetRating(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing subject id", http.StatusBadRequest)
		return
	}

	avg, err := s.repo.GetAverageScore(r.Context(), id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(RatingResponse{Average: avg})
}
