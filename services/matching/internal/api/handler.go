package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"haillion/pkg/geocode"
	"haillion/services/matching/internal/domain"
	"haillion/services/matching/internal/service"
)

type MatchingServer struct {
	repo    domain.LocationRepository
	matcher *service.MatcherService
}

type UpdateLocationRequest struct {
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	Postcode string   `json:"postcode,omitempty"`
}

type MatchRequest struct {
	RiderID  string   `json:"rider_id"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	Postcode string   `json:"postcode,omitempty"`
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

	// Resolution strategy using pointer checks:
	// We check if Lat and Lng pointers are non-nil and their values are in range.
	// If they are present & valid, we use them directly.
	// Otherwise, we fallback to postcode resolution if a postcode is provided.
	// If neither is valid/present, we return a 400 bad request error.
	var lat, lng float64
	if req.Lat != nil && req.Lng != nil && *req.Lat >= -90.0 && *req.Lat <= 90.0 && *req.Lng >= -180.0 && *req.Lng <= 180.0 {
		lat = *req.Lat
		lng = *req.Lng
	} else if req.Postcode != "" {
		geoLoc, err := geocode.LookupOutward(r.Context(), req.Postcode)
		if err != nil {
			if errors.Is(err, geocode.ErrInvalidPostcode) {
				http.Error(w, "invalid postcode", http.StatusBadRequest)
				return
			}
			if errors.Is(err, geocode.ErrPostcodeNotFound) {
				http.Error(w, "postcode not found", http.StatusBadRequest)
				return
			}
			http.Error(w, "failed to resolve postcode", http.StatusBadRequest)
			return
		}
		lat = geoLoc.Lat
		lng = geoLoc.Lng
	} else {
		http.Error(w, "missing location: provide postcode or lat/lng", http.StatusBadRequest)
		return
	}

	loc := domain.Location{Lat: lat, Lng: lng}
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

	// Resolution strategy using pointer checks:
	// We check if Lat and Lng pointers are non-nil and their values are in range.
	// If they are present & valid, we use them directly.
	// Otherwise, we fallback to postcode resolution if a postcode is provided.
	// If neither is valid/present, we return a 400 bad request error.
	var lat, lng float64
	if req.Lat != nil && req.Lng != nil && *req.Lat >= -90.0 && *req.Lat <= 90.0 && *req.Lng >= -180.0 && *req.Lng <= 180.0 {
		lat = *req.Lat
		lng = *req.Lng
	} else if req.Postcode != "" {
		geoLoc, err := geocode.LookupOutward(r.Context(), req.Postcode)
		if err != nil {
			if errors.Is(err, geocode.ErrInvalidPostcode) {
				http.Error(w, "invalid postcode", http.StatusBadRequest)
				return
			}
			if errors.Is(err, geocode.ErrPostcodeNotFound) {
				http.Error(w, "postcode not found", http.StatusBadRequest)
				return
			}
			http.Error(w, "failed to resolve postcode", http.StatusBadRequest)
			return
		}
		lat = geoLoc.Lat
		lng = geoLoc.Lng
	} else {
		http.Error(w, "missing location: provide postcode or lat/lng", http.StatusBadRequest)
		return
	}

	riderLoc := domain.Location{Lat: lat, Lng: lng}
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
