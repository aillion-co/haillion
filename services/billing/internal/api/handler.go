package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"haillion/pkg/geocode"
	"haillion/services/billing/internal/domain"

	"github.com/google/uuid"
)

type BillingServer struct {
	repo domain.PaymentRepository
}

type CreatePaymentRequest struct {
	TripID string `json:"trip_id"`
	Amount int64  `json:"amount"` // in cents
}

type PaymentResponse struct {
	ID     string              `json:"id"`
	TripID string              `json:"trip_id"`
	Amount int64               `json:"amount"`
	State  domain.PaymentState `json:"state"`
}

type SurgeResponse struct {
	Multiplier float64 `json:"multiplier"`
}

func NewServer(repo domain.PaymentRepository) *http.ServeMux {
	server := &BillingServer{repo: repo}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /payments", server.handleCreatePayment)
	mux.HandleFunc("POST /payments/{id}/process", server.handleProcessPayment)
	mux.HandleFunc("GET /surge", server.handleGetSurge)
	mux.HandleFunc("GET /fare/estimate", server.handleGetFareEstimate)

	return mux
}

func (s *BillingServer) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.TripID == "" || req.Amount <= 0 {
		http.Error(w, "missing or invalid fields", http.StatusBadRequest)
		return
	}

	p := &domain.Payment{
		ID:     uuid.New().String(),
		TripID: req.TripID,
		Amount: req.Amount,
	}

	if err := s.repo.Create(r.Context(), p); err != nil {
		http.Error(w, "failed to create payment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(PaymentResponse{
		ID:     p.ID,
		TripID: p.TripID,
		Amount: p.Amount,
		State:  p.State,
	})
}

func (s *BillingServer) handleProcessPayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing payment id", http.StatusBadRequest)
		return
	}

	err := s.repo.MarkPaid(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			http.Error(w, "payment not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *BillingServer) handleGetSurge(w http.ResponseWriter, r *http.Request) {
	ridersQuery := r.URL.Query().Get("riders")
	driversQuery := r.URL.Query().Get("drivers")

	if ridersQuery == "" || driversQuery == "" {
		http.Error(w, "missing query parameters 'riders' or 'drivers'", http.StatusBadRequest)
		return
	}

	riders, err := strconv.Atoi(ridersQuery)
	if err != nil || riders < 0 {
		http.Error(w, "invalid riders parameter", http.StatusBadRequest)
		return
	}

	drivers, err := strconv.Atoi(driversQuery)
	if err != nil || drivers < 0 {
		http.Error(w, "invalid drivers parameter", http.StatusBadRequest)
		return
	}

	multiplier := domain.CalculateSurge(riders, drivers)

	resp := SurgeResponse{Multiplier: multiplier}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *BillingServer) handleGetFareEstimate(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		http.Error(w, "missing query parameter: from and to required", http.StatusBadRequest)
		return
	}

	var riders int
	ridersQuery := r.URL.Query().Get("riders")
	if ridersQuery != "" {
		var err error
		riders, err = strconv.Atoi(ridersQuery)
		if err != nil || riders < 0 {
			http.Error(w, "invalid riders/drivers parameter", http.StatusBadRequest)
			return
		}
	}

	var drivers int
	driversQuery := r.URL.Query().Get("drivers")
	if driversQuery != "" {
		var err error
		drivers, err = strconv.Atoi(driversQuery)
		if err != nil || drivers < 0 {
			http.Error(w, "invalid riders/drivers parameter", http.StatusBadRequest)
			return
		}
	}

	locFrom, err := geocode.LookupOutward(r.Context(), from)
	if err != nil {
		if errors.Is(err, geocode.ErrInvalidPostcode) {
			http.Error(w, "invalid postcode", http.StatusBadRequest)
			return
		}
		if errors.Is(err, geocode.ErrPostcodeNotFound) {
			http.Error(w, "postcode not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	locTo, err := geocode.LookupOutward(r.Context(), to)
	if err != nil {
		if errors.Is(err, geocode.ErrInvalidPostcode) {
			http.Error(w, "invalid postcode", http.StatusBadRequest)
			return
		}
		if errors.Is(err, geocode.ErrPostcodeNotFound) {
			http.Error(w, "postcode not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	distance := geocode.Haversine(locFrom, locTo)
	fare := domain.EstimateFare(distance, riders, drivers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(fare)
}
