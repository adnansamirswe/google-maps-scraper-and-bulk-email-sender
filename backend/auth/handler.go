package auth

import (
	"encoding/json"
	"net/http"
)

// Handler manages authentication endpoints.
type Handler struct {
	password  string
	jwtSecret string
}

// NewHandler creates a new auth handler.
func NewHandler(password, jwtSecret string) *Handler {
	return &Handler{
		password:  password,
		jwtSecret: jwtSecret,
	}
}

type loginRequest struct {
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Login handles POST /api/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.Password != h.password {
		renderJSON(w, http.StatusUnauthorized, errorResponse{Error: "wrong password"})
		return
	}

	token, expiresAt, err := GenerateToken(h.jwtSecret)
	if err != nil {
		renderJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to generate token"})
		return
	}

	renderJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

func renderJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(data)
}
