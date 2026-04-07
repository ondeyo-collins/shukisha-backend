package parcels

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/shukisha/backend/internal/identity"
)

type ParcelHandler struct {
	service *ParcelService
}

func NewParcelHandler(service *ParcelService) *ParcelHandler {
	return &ParcelHandler{service: service}
}

func (h *ParcelHandler) BookParcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(identity.UserIDKey).(uuid.UUID)

	var req BookParcelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	parcel, err := h.service.BookParcel(r.Context(), userID, req.RouteID, req.ReceiverPhone, req.Description, req.DeclaredValue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(parcel)
}

func (h *ParcelHandler) GetParcelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parcelID := r.URL.Path[len("/v1/parcels/"):]
	if parcelID == "" {
		http.Error(w, "Parcel ID required", http.StatusBadRequest)
		return
	}

	var parcel Parcel
	if err := h.service.db.Preload("TrackingEvents").Preload("Payment").First(&parcel, "id = ?", parcelID).Error; err != nil {
		http.Error(w, "Parcel not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(parcel)
}

func (h *ParcelHandler) PayParcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// e.g., /v1/parcels/{id}/pay
	path := r.URL.Path
	parcelID := path[len("/v1/parcels/") : strings.LastIndex(path, "/pay")]

	if err := h.service.PayParcel(r.Context(), parcelID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Payment Initiated"})
}

// PaymentCallback simulates an M-Pesa callback for MVP/testing
func (h *ParcelHandler) PaymentCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type CallbackReq struct {
		ParcelID string `json:"parcel_id"`
		Ref      string `json:"ref"`
	}

	var req CallbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	pID, _ := uuid.Parse(req.ParcelID)
	if err := h.service.ProcessPayment(r.Context(), pID, req.Ref); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Payment Processed"})
}

// DRIVER PARCEL HANDLERS

func (h *ParcelHandler) GetDriverOffers(w http.ResponseWriter, r *http.Request) {
	driverID, ok := r.Context().Value(identity.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	offers, err := h.service.GetDriverOffers(r.Context(), driverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(offers)
}

func (h *ParcelHandler) DriverParcelAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	driverID, ok := r.Context().Value(identity.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// path: /v1/driver/parcels/{id}/{action}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}

	parcelID, err := uuid.Parse(pathParts[4])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	action := pathParts[5]

	var req struct {
		OTP string `json:"otp"`
	}
	if action == "pickup" || action == "deliver" {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	switch action {
	case "accept":
		err = h.service.AcceptOffer(r.Context(), driverID, parcelID)
	case "pickup":
		err = h.service.PickupParcel(r.Context(), driverID, parcelID, req.OTP)
	case "checkpoint":
		err = h.service.Checkpoint(r.Context(), driverID, parcelID)
	case "arrive-dropoff":
		err = h.service.ArriveDropoff(r.Context(), driverID, parcelID)
	case "delivery-attempt":
		err = h.service.DeliveryAttempt(r.Context(), driverID, parcelID)
	case "deliver":
		err = h.service.DeliverParcel(r.Context(), driverID, parcelID, req.OTP)
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Success", "action": action})
}
