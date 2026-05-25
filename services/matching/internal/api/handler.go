package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

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

type RiderRequest struct {
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	Postcode string   `json:"postcode,omitempty"`
}

type NearbyRidersResponse struct {
	Riders []NearbyRiderJSON `json:"riders"`
}

type NearbyRiderJSON struct {
	RiderID   string  `json:"rider_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	DistanceM float64 `json:"distance_m"`
}

type NearbyDriversResponse struct {
	Drivers []NearbyDriverJSON `json:"drivers"`
}

type NearbyDriverJSON struct {
	DriverID  string  `json:"driver_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	DistanceM float64 `json:"distance_m"`
}

func NewServer(repo domain.LocationRepository, matcher *service.MatcherService) *http.ServeMux {
	server := &MatchingServer{
		repo:    repo,
		matcher: matcher,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /drivers/{id}/location", server.handleUpdateLocation)
	mux.HandleFunc("POST /match", server.handleMatch)

	mux.HandleFunc("POST /riders/{id}/request", server.handleRiderRequest)
	mux.HandleFunc("DELETE /riders/{id}/request", server.handleRiderCancel)
	mux.HandleFunc("GET /drivers/{id}/nearby-riders", server.handleNearbyRiders)
	mux.HandleFunc("GET /riders/{id}/nearby-drivers", server.handleNearbyDrivers)

	return mux
}

// resolveLocation abstracts postcode and coordinates resolution
func resolveLocation(ctx context.Context, latPtr, lngPtr *float64, postcode string) (domain.Location, int, string) {
	if latPtr != nil && lngPtr != nil && *latPtr >= -90.0 && *latPtr <= 90.0 && *lngPtr >= -180.0 && *lngPtr <= 180.0 {
		return domain.Location{Lat: *latPtr, Lng: *lngPtr}, 0, ""
	} else if postcode != "" {
		geoLoc, err := geocode.LookupOutward(ctx, postcode)
		if err != nil {
			if errors.Is(err, geocode.ErrInvalidPostcode) {
				return domain.Location{}, http.StatusBadRequest, "invalid postcode"
			}
			if errors.Is(err, geocode.ErrPostcodeNotFound) {
				return domain.Location{}, http.StatusBadRequest, "postcode not found"
			}
			return domain.Location{}, http.StatusBadRequest, "failed to resolve postcode"
		}
		return domain.Location{Lat: geoLoc.Lat, Lng: geoLoc.Lng}, 0, ""
	}
	return domain.Location{}, http.StatusBadRequest, "missing location: provide postcode or lat/lng"
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

	loc, errCode, errMsg := resolveLocation(r.Context(), req.Lat, req.Lng, req.Postcode)
	if errCode != 0 {
		http.Error(w, errMsg, errCode)
		return
	}

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

	riderLoc, errCode, errMsg := resolveLocation(r.Context(), req.Lat, req.Lng, req.Postcode)
	if errCode != 0 {
		http.Error(w, errMsg, errCode)
		return
	}

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

func (s *MatchingServer) handleRiderRequest(w http.ResponseWriter, r *http.Request) {
	riderID := r.PathValue("id")
	if riderID == "" {
		http.Error(w, "missing rider id", http.StatusBadRequest)
		return
	}

	var req RiderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	loc, errCode, errMsg := resolveLocation(r.Context(), req.Lat, req.Lng, req.Postcode)
	if errCode != 0 {
		http.Error(w, errMsg, errCode)
		return
	}

	if err := s.repo.UpsertPendingRider(r.Context(), riderID, loc); err != nil {
		http.Error(w, "failed to upsert pending rider", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *MatchingServer) handleRiderCancel(w http.ResponseWriter, r *http.Request) {
	riderID := r.PathValue("id")
	if riderID == "" {
		http.Error(w, "missing rider id", http.StatusBadRequest)
		return
	}

	if err := s.repo.DeletePendingRider(r.Context(), riderID); err != nil {
		http.Error(w, "failed to delete pending rider request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *MatchingServer) handleNearbyRiders(w http.ResponseWriter, r *http.Request) {
	driverID := r.PathValue("id")
	if driverID == "" {
		http.Error(w, "missing driver id", http.StatusBadRequest)
		return
	}

	radiusStr := r.URL.Query().Get("radius_m")
	radiusMeters := 5000.0
	if radiusStr != "" {
		var err error
		radiusMeters, err = strconv.ParseFloat(radiusStr, 64)
		if err != nil || radiusMeters <= 0 || radiusMeters > 50000 {
			http.Error(w, "invalid radius_m parameter", http.StatusBadRequest)
			return
		}
	}

	driverLoc, err := s.repo.GetDriverLocation(r.Context(), driverID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "driver location unknown", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get driver location", http.StatusInternalServerError)
		return
	}

	hits, err := s.repo.FindNearbyPendingRiders(r.Context(), driverLoc, radiusMeters, 10*time.Minute)
	if err != nil {
		http.Error(w, "failed to query nearby riders", http.StatusInternalServerError)
		return
	}

	ridersJSON := make([]NearbyRiderJSON, 0, len(hits))
	for _, h := range hits {
		ridersJSON = append(ridersJSON, NearbyRiderJSON{
			RiderID:   h.RiderID,
			Lat:       h.Location.Lat,
			Lng:       h.Location.Lng,
			DistanceM: h.DistanceMeters,
		})
	}

	resp := NearbyRidersResponse{Riders: ridersJSON}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *MatchingServer) handleNearbyDrivers(w http.ResponseWriter, r *http.Request) {
	riderID := r.PathValue("id")
	if riderID == "" {
		http.Error(w, "missing rider id", http.StatusBadRequest)
		return
	}

	radiusStr := r.URL.Query().Get("radius_m")
	radiusMeters := 5000.0
	if radiusStr != "" {
		var err error
		radiusMeters, err = strconv.ParseFloat(radiusStr, 64)
		if err != nil || radiusMeters <= 0 || radiusMeters > 50000 {
			http.Error(w, "invalid radius_m parameter", http.StatusBadRequest)
			return
		}
	}

	riderLoc, err := s.repo.GetPendingRider(r.Context(), riderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "rider has no active request", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get pending rider", http.StatusInternalServerError)
		return
	}

	hits, err := s.repo.FindNearbyDriversWithDistance(r.Context(), riderLoc, radiusMeters)
	if err != nil {
		http.Error(w, "failed to query nearby drivers", http.StatusInternalServerError)
		return
	}

	driversJSON := make([]NearbyDriverJSON, 0, len(hits))
	for _, h := range hits {
		driversJSON = append(driversJSON, NearbyDriverJSON{
			DriverID:  h.DriverID,
			Lat:       h.Location.Lat,
			Lng:       h.Location.Lng,
			DistanceM: h.DistanceMeters,
		})
	}

	resp := NearbyDriversResponse{Drivers: driversJSON}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
