package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"haillion/services/matching/internal/domain"
	"haillion/services/matching/internal/service"
)

type MatchingServer struct {
	repo    domain.LocationRepository
	matcher *service.MatcherService
}

type UpdateLocationRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type MatchRequest struct {
	RiderID string  `json:"rider_id"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

type MatchResponse struct {
	DriverID string `json:"driver_id"`
	ETA      int    `json:"eta_seconds"`
}

func NewServer(repo domain.LocationRepository, matcher *service.MatcherService) *http.ServeMux {
	server := &MatchingServer{
		repo:    repo,
		matcher: matcher,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /drivers/{id}/location", server.handleUpdateLocation)
	mux.HandleFunc("POST /match", server.handleMatch)

	return mux
}

func (s *MatchingServer) handleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	driverID := r.PathValue("id")
	if driverID == "" {
		http.Error(w, "missing driver id", http.StatusBadRequest)
		return
	}

	var req UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate coordinates range
	if req.Lat < -90.0 || req.Lat > 90.0 || req.Lng < -180.0 || req.Lng > 180.0 {
		http.Error(w, "invalid coordinate range", http.StatusBadRequest)
		return
	}

	loc := domain.Location{Lat: req.Lat, Lng: req.Lng}
	if err := s.repo.UpdateLocation(r.Context(), driverID, loc); err != nil {
		http.Error(w, "failed to update location", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *MatchingServer) handleMatch(w http.ResponseWriter, r *http.Request) {
	var req MatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RiderID == "" {
		http.Error(w, "missing rider id", http.StatusBadRequest)
		return
	}

	if req.Lat < -90.0 || req.Lat > 90.0 || req.Lng < -180.0 || req.Lng > 180.0 {
		http.Error(w, "invalid coordinate range", http.StatusBadRequest)
		return
	}

	riderLoc := domain.Location{Lat: req.Lat, Lng: req.Lng}
	// Default radius of 5000 meters (5km) for the MVP match
	radiusMeters := 5000.0

	dl, eta, err := s.matcher.MatchDriver(r.Context(), riderLoc, radiusMeters)
	if err != nil {
		if errors.Is(err, service.ErrNoDriversFound) {
			http.Error(w, "no available drivers found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := MatchResponse{
		DriverID: dl.DriverID,
		ETA:      eta,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
