package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/shukisha/backend/internal/platform/cloudinary"
	"github.com/shukisha/backend/internal/platform/config"
)

type AdminHandler struct {
	service *AdminService
}

func NewAdminHandler(service *AdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminID, ok := r.Context().Value(AdminIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing admin ID", http.StatusUnauthorized)
		return
	}

	me, err := h.service.GetCurrentAdmin(r.Context(), adminID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(me)
}

func (h *AdminHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *AdminHandler) UpdateDriverStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ManagementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	active := req.Action == "APPROVE" || req.Action == "ACTIVATE"
	if err := h.service.UpdateDriverStatus(r.Context(), req.EntityID, active); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Success", "action": req.Action})
}

func (h *AdminHandler) GetParcels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	routeID := r.URL.Query().Get("route_id")
	driverID := r.URL.Query().Get("driver_id")
	limit := parseInt(r.URL.Query().Get("limit"), 50)
	offset := parseInt(r.URL.Query().Get("offset"), 0)
	slaHours := parseInt(r.URL.Query().Get("sla_hours"), 8)

	if routeID != "" && !isUUID(routeID) {
		http.Error(w, "Invalid route_id (must be UUID)", http.StatusBadRequest)
		return
	}
	if driverID != "" && !isUUID(driverID) {
		http.Error(w, "Invalid driver_id (must be UUID)", http.StatusBadRequest)
		return
	}

	ps, err := h.service.GetParcels(r.Context(), GetParcelsParams{
		Status:   status,
		Query:    q,
		RouteID:  routeID,
		DriverID: driverID,
		Limit:    limit,
		Offset:   offset,
		SLAHours: slaHours,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ps)
}

func (h *AdminHandler) GetParcelExceptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	routeID := r.URL.Query().Get("route_id")
	driverID := r.URL.Query().Get("driver_id")
	slaHours := parseInt(r.URL.Query().Get("sla_hours"), 8)
	limit := parseInt(r.URL.Query().Get("limit"), 50)
	offset := parseInt(r.URL.Query().Get("offset"), 0)
	onlyUnassigned := r.URL.Query().Get("only_unassigned") == "1"

	if routeID != "" && !isUUID(routeID) {
		http.Error(w, "Invalid route_id (must be UUID)", http.StatusBadRequest)
		return
	}
	if driverID != "" && !isUUID(driverID) {
		http.Error(w, "Invalid driver_id (must be UUID)", http.StatusBadRequest)
		return
	}

	ps, err := h.service.GetParcelExceptions(r.Context(), GetParcelExceptionsParams{
		Query:          q,
		Status:         status,
		RouteID:        routeID,
		DriverID:       driverID,
		OnlyUnassigned: onlyUnassigned,
		SLAHours:       slaHours,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ps)
}

func (h *AdminHandler) ListClaims(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := r.URL.Query().Get("status")
	claimType := r.URL.Query().Get("type")
	limit := parseInt(r.URL.Query().Get("limit"), 50)
	offset := parseInt(r.URL.Query().Get("offset"), 0)

	claims, err := h.service.ListClaims(r.Context(), ListClaimsParams{
		Status: status,
		Type:   claimType,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

func (h *AdminHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := r.URL.Query().Get("status")
	to := r.URL.Query().Get("to")
	parcelID := r.URL.Query().Get("parcel_id")
	limit := parseInt(r.URL.Query().Get("limit"), 50)
	offset := parseInt(r.URL.Query().Get("offset"), 0)

	if parcelID != "" && !isUUID(parcelID) {
		http.Error(w, "Invalid parcel_id (must be UUID)", http.StatusBadRequest)
		return
	}

	jobs, err := h.service.ListNotificationJobs(r.Context(), ListNotificationJobsParams{
		Status:   status,
		To:       to,
		ParcelID: parcelID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (h *AdminHandler) RetryNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminID, ok := r.Context().Value(AdminIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing admin ID", http.StatusUnauthorized)
		return
	}

	jobID, ok := extractID(r.URL.Path, "/v1/admin/notifications/", "/retry")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(jobID) {
		http.Error(w, "Invalid notification ID (must be UUID)", http.StatusBadRequest)
		return
	}

	if err := h.service.RetryNotificationJob(r.Context(), jobID, adminID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "Queued"})
}

func (h *AdminHandler) ParcelClaims(w http.ResponseWriter, r *http.Request) {
	// /v1/admin/parcels/{id}/claims
	adminID, ok := r.Context().Value(AdminIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing admin ID", http.StatusUnauthorized)
		return
	}

	parcelID, ok := extractID(r.URL.Path, "/v1/admin/parcels/", "/claims")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(parcelID) {
		http.Error(w, "Invalid parcel ID (must be UUID)", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		claims, err := h.service.ListParcelClaims(r.Context(), parcelID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(claims)
	case http.MethodPost:
		var req struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		created, err := h.service.CreateParcelClaim(r.Context(), parcelID, adminID, req.Type, req.Description)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) UpdateClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminID, ok := r.Context().Value(AdminIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing admin ID", http.StatusUnauthorized)
		return
	}

	claimID, ok := extractID(r.URL.Path, "/v1/admin/claims/", "")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(claimID) {
		http.Error(w, "Invalid claim ID (must be UUID)", http.StatusBadRequest)
		return
	}

	var req struct {
		Status          string `json:"status"`
		ResolutionNotes string `json:"resolution_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateClaimStatus(r.Context(), claimID, adminID, req.Status, req.ResolutionNotes); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "Updated"})
}

func (h *AdminHandler) ClaimAttachments(w http.ResponseWriter, r *http.Request) {
	adminID, ok := r.Context().Value(AdminIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing admin ID", http.StatusUnauthorized)
		return
	}

	claimID, ok := extractID(r.URL.Path, "/v1/admin/claims/", "/attachments")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(claimID) {
		http.Error(w, "Invalid claim ID (must be UUID)", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		metas, err := h.service.ListClaimAttachments(r.Context(), claimID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metas)
	case http.MethodPost:
		// Limit request to 3MB (MVP evidence uploads only).
		r.Body = http.MaxBytesReader(w, r.Body, 3<<20)
		if err := r.ParseMultipartForm(3 << 20); err != nil {
			http.Error(w, "Invalid multipart form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Read up to 2MB actual file bytes.
		const maxFileSize = 2 << 20
		data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusBadRequest)
			return
		}
		if len(data) > maxFileSize {
			http.Error(w, "File too large (max 2MB)", http.StatusBadRequest)
			return
		}

		mimeType := header.Header.Get("Content-Type")
		meta, err := h.service.AddClaimAttachment(r.Context(), claimID, adminID, header.Filename, mimeType, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(meta)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) DownloadClaimAttachment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, ok := r.Context().Value(AdminIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing admin ID", http.StatusUnauthorized)
		return
	}

	attachmentID, ok := extractID(r.URL.Path, "/v1/admin/claims/attachments/", "/download")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(attachmentID) {
		http.Error(w, "Invalid attachment ID (must be UUID)", http.StatusBadRequest)
		return
	}

	a, err := h.service.GetClaimAttachment(r.Context(), attachmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	cloudName := config.GetEnv("CLOUDINARY_CLOUD_NAME", "")
	apiSecret := config.GetEnv("CLOUDINARY_API_SECRET", "")
	signedURL, err := cloudinary.AuthenticatedURL(cloudName, apiSecret, a.CloudinaryResourceType, a.CloudinaryVersion, a.CloudinaryPublicID, a.CloudinaryFormat)
	if err != nil {
		http.Error(w, "Attachment delivery not configured", http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, signedURL, nil)
	if err != nil {
		http.Error(w, "Failed to build download request", http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch attachment", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		http.Error(w, "Attachment not available", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", a.MimeType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+a.FileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}

func (h *AdminHandler) GetParcelDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parcelID, ok := extractID(r.URL.Path, "/v1/admin/parcels/", "")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(parcelID) {
		http.Error(w, "Invalid parcel ID (must be UUID)", http.StatusBadRequest)
		return
	}

	detail, err := h.service.GetParcelDetail(r.Context(), parcelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *AdminHandler) GetParcelEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parcelID, ok := extractID(r.URL.Path, "/v1/admin/parcels/", "/events")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(parcelID) {
		http.Error(w, "Invalid parcel ID (must be UUID)", http.StatusBadRequest)
		return
	}

	events, err := h.service.GetParcelEvents(r.Context(), parcelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (h *AdminHandler) AddParcelNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminID, ok := r.Context().Value(AdminIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing admin ID", http.StatusUnauthorized)
		return
	}

	parcelID, ok := extractID(r.URL.Path, "/v1/admin/parcels/", "/notes")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(parcelID) {
		http.Error(w, "Invalid parcel ID (must be UUID)", http.StatusBadRequest)
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.AddParcelNote(r.Context(), parcelID, adminID, req.Note); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "Noted"})
}

func (h *AdminHandler) NotifyDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminID, ok := r.Context().Value(AdminIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing admin ID", http.StatusUnauthorized)
		return
	}

	parcelID, ok := extractID(r.URL.Path, "/v1/admin/parcels/", "/notify-driver")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(parcelID) {
		http.Error(w, "Invalid parcel ID (must be UUID)", http.StatusBadRequest)
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.service.NotifyParcelDriver(r.Context(), parcelID, adminID, req.Message); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Notified"})
}

func (h *AdminHandler) OverrideParcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminID, ok := r.Context().Value(AdminIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized: missing admin ID", http.StatusUnauthorized)
		return
	}

	parcelID, ok := extractID(r.URL.Path, "/v1/admin/parcels/", "/override")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !isUUID(parcelID) {
		http.Error(w, "Invalid parcel ID (must be UUID)", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.OverrideParcel(r.Context(), parcelID, adminID, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Overridden"})
}

func extractID(path, prefix, suffix string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	idPart := strings.TrimPrefix(path, prefix)
	if idPart == "" {
		return "", false
	}
	if suffix != "" {
		if !strings.HasSuffix(idPart, suffix) {
			return "", false
		}
		idPart = strings.TrimSuffix(idPart, suffix)
	}
	idPart = strings.TrimSuffix(idPart, "/")
	if idPart == "" || strings.Contains(idPart, "/") {
		return "", false
	}
	return idPart, true
}

func parseInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	var n int
	for _, c := range raw {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func isUUID(raw string) bool {
	_, err := uuid.Parse(strings.TrimSpace(raw))
	return err == nil
}

// Transport Admins

func (h *AdminHandler) GetLiveFleet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fleet, err := h.service.GetLiveFleet(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fleet)
}

func (h *AdminHandler) PingDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// e.g. /v1/admin/drivers/{id}/ping
	path := r.URL.Path
	driverID := path[len("/v1/admin/drivers/"):strings.LastIndex(path, "/ping")]

	if err := h.service.PingDriver(r.Context(), driverID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Pinged"})
}

func (h *AdminHandler) GetRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rs, err := h.service.GetRoutes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rs)
}

func (h *AdminHandler) GetStops(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ss, err := h.service.GetStops(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ss)
}

func (h *AdminHandler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name        string  `json:"name"`
		ShortName   string  `json:"short_name"`
		Description string  `json:"description"`
		BaseFare    float64 `json:"base_fare"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	route, err := h.service.CreateRoute(r.Context(), req.Name, req.ShortName, req.Description, req.BaseFare)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

func (h *AdminHandler) CreateStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name     string `json:"name"`
		PointWKT string `json:"point_wkt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	stop, err := h.service.CreateStop(r.Context(), req.Name, req.PointWKT)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stop)
}

// Finance Admins

func (h *AdminHandler) GetUnpaidLedger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := h.service.GetUnpaidLedger(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *AdminHandler) ExecutePayouts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DriverIDs []string `json:"driver_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.service.ExecutePayouts(r.Context(), req.DriverIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Payouts Executed"})
}

// Support Admins

func (h *AdminHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query().Get("q")
	users, err := h.service.SearchUsers(r.Context(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *AdminHandler) RescueAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// e.g. /v1/admin/users/{id}/rescue-auth
	path := r.URL.Path
	userID := path[len("/v1/admin/users/"):strings.LastIndex(path, "/rescue-auth")]

	pin, err := h.service.RescueAuth(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"rescue_pin": pin})
}

// Audit / Overseer Admins

func (h *AdminHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logs, err := h.service.GetAuditLogs(r.Context(), 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// MANAGEMENT HANDLERS

func (h *AdminHandler) ListAdmins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	admins, err := h.service.ListAdmins(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(admins)
}

func (h *AdminHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email      string `json:"email"`
		Name       string `json:"name"`
		Role       string `json:"role"`
		Department string `json:"department"`
		Password   string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	admin, err := h.service.CreateAdmin(r.Context(), req.Email, req.Name, req.Role, req.Department, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(admin)
}

func (h *AdminHandler) UpdateAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// /v1/admin/management/{id}
	path := r.URL.Path
	adminID := path[len("/v1/admin/management/"):]

	var req struct {
		Role       string `json:"role"`
		Department string `json:"department"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateAdmin(r.Context(), adminID, req.Role, req.Department, req.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Updated"})
}
