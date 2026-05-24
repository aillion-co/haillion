package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"haillion/services/trip/internal/domain"

	"github.com/google/uuid"
)

type TripServer struct {
	repo domain.TripRepository
}

type CreateTripRequest struct {
	RiderID string  `json:"rider_id"`
	Lat     float64 `json:"lat"` // retained for the api contract, stubbed for MVP
	Lng     float64 `json:"lng"` // retained for the api contract, stubbed for MVP
}

type AcceptTripRequest struct {
	DriverID string `json:"driver_id"`
}

type TripResponse struct {
	ID       string       `json:"id"`
	RiderID  string       `json:"rider_id"`
	DriverID *string      `json:"driver_id,omitempty"`
	State    domain.State `json:"state"`
}

func NewServer(repo domain.TripRepository) *http.ServeMux {
	server := &TripServer{repo: repo}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /trips", server.handleCreateTrip)
	mux.HandleFunc("POST /trips/{id}/accept", server.handleAcceptTrip)
	mux.HandleFunc("POST /trips/{id}/start", server.handleStartTrip)
	mux.HandleFunc("POST /trips/{id}/complete", server.handleCompleteTrip)
	mux.HandleFunc("GET /trips/{id}", server.handleGetTrip)

	return mux
}

func (s *TripServer) handleCreateTrip(w http.ResponseWriter, r *http.Request) {
	var req CreateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RiderID == "" {
		http.Error(w, "missing rider id", http.StatusBadRequest)
		return
	}

	t := &domain.Trip{
		ID:      uuid.New().String(),
		RiderID: req.RiderID,
	}

	if err := s.repo.Create(r.Context(), t); err != nil {
		http.Error(w, "failed to create trip", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(TripResponse{
		ID:       t.ID,
		RiderID:  t.RiderID,
		DriverID: t.DriverID,
		State:    t.State,
	})
}

func (s *TripServer) handleAcceptTrip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing trip id", http.StatusBadRequest)
		return
	}

	var req AcceptTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.DriverID == "" {
		http.Error(w, "missing driver id", http.StatusBadRequest)
		return
	}

	err := s.repo.UpdateState(r.Context(), id, &req.DriverID, domain.StateAccepted, domain.StateRequested)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidStateTransition) {
			http.Error(w, "invalid state transition or concurrent acceptance", http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrTripNotFound) {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *TripServer) handleStartTrip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing trip id", http.StatusBadRequest)
		return
	}

	// Fetch current trip to retain current driver
	t, err := s.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTripNotFound) {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = s.repo.UpdateState(r.Context(), id, t.DriverID, domain.StateInProgress, domain.StateAccepted)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidStateTransition) {
			http.Error(w, "invalid state transition", http.StatusConflict)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *TripServer) handleCompleteTrip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing trip id", http.StatusBadRequest)
		return
	}

	// Fetch current trip to retain current driver
	t, err := s.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTripNotFound) {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = s.repo.UpdateState(r.Context(), id, t.DriverID, domain.StateCompleted, domain.StateInProgress)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidStateTransition) {
			http.Error(w, "invalid state transition", http.StatusConflict)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *TripServer) handleGetTrip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing trip id", http.StatusBadRequest)
		return
	}

	t, err := s.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTripNotFound) {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(TripResponse{
		ID:       t.ID,
		RiderID:  t.RiderID,
		DriverID: t.DriverID,
		State:    t.State,
	})
}
