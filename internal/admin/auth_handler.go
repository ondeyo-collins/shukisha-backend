package admin

import (
	"encoding/json"
	"log"
	"net/http"
)

type AdminAuthHandler struct {
	service *AdminAuthService
}

func NewAdminAuthHandler(service *AdminAuthService) *AdminAuthHandler {
	return &AdminAuthHandler{service: service}
}

type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AdminLoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

// POST /v1/auth/admin/login
func (h *AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	token, adminUser, cred, err := h.service.LoginWithPassword(r.Context(), req.Email, req.Password)
	if err != nil {
		log.Printf("Admin login failed for %s: %v", req.Email, err)
		// Use same message for both "not found" and "wrong password" to avoid enumeration
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Fail fast on misconfigured/suspended admin accounts
	if adminUser.Status != "ACTIVE" {
		http.Error(w, "Admin account is not active", http.StatusForbidden)
		return
	}
	if !IsCanonicalAdminRole(adminUser.Role) {
		http.Error(w, "Admin role misconfigured; contact SUPER_ADMIN", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AdminLoginResponse{
		AccessToken: token,
		ExpiresIn:   86400,
		Name:        cred.Name,
		Email:       cred.Email,
		Role:        adminUser.Role,
	})
}
