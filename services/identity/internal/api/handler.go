package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"aillion/services/identity/internal/domain"
)

type UserServer struct {
	repo domain.UserRepository
}

type RegisterRequest struct {
	ID    string      `json:"id"`
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
}

type UserResponse struct {
	ID    string      `json:"id"`
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
}

func NewServer(repo domain.UserRepository) *http.ServeMux {
	server := &UserServer{repo: repo}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /users/register", server.handleRegister)
	mux.HandleFunc("GET /users/{id}", server.handleGetUser)

	return mux
}

func (s *UserServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Email == "" || (req.Role != domain.RoleRider && req.Role != domain.RoleDriver) {
		http.Error(w, "missing or invalid fields", http.StatusBadRequest)
		return
	}

	u := &domain.User{
		ID:    req.ID,
		Email: req.Email,
		Role:  req.Role,
	}

	if err := s.repo.Create(r.Context(), u); err != nil {
		if errors.Is(err, domain.ErrDuplicateEmail) {
			http.Error(w, "email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(UserResponse{
		ID:    u.ID,
		Email: u.Email,
		Role:  u.Role,
	})
}

func (s *UserServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing user id", http.StatusBadRequest)
		return
	}

	u, err := s.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(UserResponse{
		ID:    u.ID,
		Email: u.Email,
		Role:  u.Role,
	})
}
