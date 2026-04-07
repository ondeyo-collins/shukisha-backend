package drivers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/shukisha/backend/internal/identity"
)

type DriverHandler struct {
	service *DriverService
}

func NewDriverHandler(service *DriverService) *DriverHandler {
	return &DriverHandler{service: service}
}

type OnboardingRequest struct {
	VehiclePlate string `json:"vehicle_plate"`
	SaccoName    string `json:"sacco_name"`
}

type SessionStartRequest struct {
	RouteID string `json:"route_id"`
}

func (h *DriverHandler) Onboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	driverID := r.Context().Value(identity.UserIDKey).(uuid.UUID)

	var req OnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.Onboard(r.Context(), driverID, req.VehiclePlate, req.SaccoName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile updated"})
}

func (h *DriverHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	driverID := r.Context().Value(identity.UserIDKey).(uuid.UUID)

	var req SessionStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	routeID, err := uuid.Parse(req.RouteID)
	if err != nil {
		http.Error(w, "Invalid route ID", http.StatusBadRequest)
		return
	}

	session, err := h.service.StartSession(r.Context(), driverID, routeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

func (h *DriverHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	driverID := r.Context().Value(identity.UserIDKey).(uuid.UUID)

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.Heartbeat(r.Context(), driverID, req.Lat, req.Lng); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DriverHandler) StopSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	driverID := r.Context().Value(identity.UserIDKey).(uuid.UUID)

	if err := h.service.StopSession(r.Context(), driverID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Session closed"})
}
