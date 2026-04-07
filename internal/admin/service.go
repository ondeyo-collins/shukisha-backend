package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shukisha/backend/internal/drivers"
	"github.com/shukisha/backend/internal/identity"
	"github.com/shukisha/backend/internal/ledger"
	"github.com/shukisha/backend/internal/parcels"
	"github.com/shukisha/backend/internal/routes"
	"github.com/shukisha/backend/internal/stopalerts"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminService struct {
	db *gorm.DB
}

func NewAdminService(database *gorm.DB) *AdminService {
	return &AdminService{db: database}
}

type AdminPaymentSummary struct {
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

type AdminParcelSummary struct {
	ID            uuid.UUID            `json:"id"`
	SenderID      uuid.UUID            `json:"sender_id"`
	DriverID      *uuid.UUID           `json:"driver_id"`
	ReceiverPhone string               `json:"receiver_phone"`
	Description   string               `json:"description"`
	DeclaredValue float64              `json:"declared_value"`
	Status        string               `json:"status"`
	RouteID       uuid.UUID            `json:"route_id"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Payment       *AdminPaymentSummary `json:"payment,omitempty"`
	LastEventAt   *time.Time           `json:"last_event_at,omitempty"`
	LastEventType string               `json:"last_event_type,omitempty"`
	AgeMinutes    int                  `json:"age_minutes"`
	SlaBreached   bool                 `json:"sla_breached"`
}

type AdminParcelDetail struct {
	Parcel         AdminParcelSummary            `json:"parcel"`
	TrackingEvents []parcels.ParcelTrackingEvent `json:"tracking_events"`
	Notes          []parcels.ParcelNote          `json:"notes"`
}

func parcelToAdminSummary(p parcels.Parcel) AdminParcelSummary {
	var pay *AdminPaymentSummary
	if p.Payment != nil {
		pay = &AdminPaymentSummary{
			Amount: p.Payment.Amount,
			Status: p.Payment.Status,
		}
	}
	return AdminParcelSummary{
		ID:            p.ID,
		SenderID:      p.SenderID,
		DriverID:      p.DriverID,
		ReceiverPhone: p.ReceiverPhone,
		Description:   p.Description,
		DeclaredValue: p.DeclaredValue,
		Status:        p.Status,
		RouteID:       p.RouteID,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		Payment:       pay,
		AgeMinutes:    int(time.Since(p.CreatedAt).Minutes()),
	}
}

func (s *AdminService) GetStats(ctx context.Context) (*DashboardStats, error) {
	var stats DashboardStats

	s.db.Model(&identity.User{}).Count(&stats.TotalUsers)
	s.db.Model(&drivers.DriverSession{}).Where("status = ?", "ONLINE").Count(&stats.ActiveDrivers)
	s.db.Model(&parcels.Parcel{}).Where("status NOT IN (?)", []string{"DELIVERED", "CANCELLED"}).Count(&stats.ActiveParcels)
	s.db.Model(&stopalerts.StopAlert{}).Where("status = ?", "ARMED").Count(&stats.ActiveAlerts)
	s.db.Model(&routes.Route{}).Count(&stats.TotalRoutes)

	s.db.Model(&parcels.Payment{}).Where("status = ?", "SUCCESS").Select("COALESCE(SUM(amount), 0)").Scan(&stats.TotalRevenue)

	return &stats, nil
}

func (s *AdminService) UpdateDriverStatus(ctx context.Context, driverID string, active bool) error {
	status := "INACTIVE"
	if active {
		status = "ACTIVE"
	}
	return s.db.Model(&drivers.DriverProfile{}).Where("id = ?", driverID).Update("status", status).Error
}

type GetParcelsParams struct {
	Status   string
	Query    string
	RouteID  string
	DriverID string
	Limit    int
	Offset   int
	SLAHours int
}

func (s *AdminService) GetParcels(ctx context.Context, params GetParcelsParams) ([]AdminParcelSummary, error) {
	var results []parcels.Parcel
	query := s.db.WithContext(ctx).Model(&parcels.Parcel{}).Preload("Payment")
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.RouteID != "" {
		query = query.Where("route_id = ?", params.RouteID)
	}
	if params.DriverID != "" {
		query = query.Where("driver_id = ?", params.DriverID)
	}
	if q := strings.TrimSpace(params.Query); q != "" {
		query = query.Where("CAST(id AS TEXT) ILIKE ? OR receiver_phone ILIKE ?", q+"%", "%"+q+"%")
	}
	limit := params.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	err := query.Order("created_at desc").Limit(limit).Offset(offset).Find(&results).Error
	if err != nil {
		return nil, err
	}

	// Load last tracking event per parcel (best-effort; does not fail the list response).
	type lastEventRow struct {
		ParcelID  uuid.UUID `gorm:"column:parcel_id"`
		EventType string    `gorm:"column:event_type"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	lastByParcel := map[uuid.UUID]lastEventRow{}
	if len(results) > 0 {
		ids := make([]uuid.UUID, 0, len(results))
		for _, p := range results {
			ids = append(ids, p.ID)
		}

		var rows []lastEventRow
		_ = s.db.WithContext(ctx).Raw(`
			SELECT DISTINCT ON (parcel_id) parcel_id, event_type, created_at
			FROM parcel_tracking_events
			WHERE parcel_id IN ?
			ORDER BY parcel_id, created_at DESC
		`, ids).Scan(&rows).Error

		for _, r := range rows {
			lastByParcel[r.ParcelID] = r
		}
	}

	slaHours := params.SLAHours
	if slaHours <= 0 {
		slaHours = 8
	}

	out := make([]AdminParcelSummary, 0, len(results))
	for _, p := range results {
		summary := parcelToAdminSummary(p)
		lastActivityAt := p.CreatedAt
		if !p.UpdatedAt.IsZero() {
			lastActivityAt = p.UpdatedAt
		}
		if le, ok := lastByParcel[p.ID]; ok {
			summary.LastEventAt = &le.CreatedAt
			summary.LastEventType = le.EventType
			lastActivityAt = le.CreatedAt
		}
		summary.SlaBreached = time.Since(lastActivityAt) >= time.Duration(slaHours)*time.Hour && p.Status != "DELIVERED"
		out = append(out, summary)
	}
	return out, nil
}

type GetParcelExceptionsParams struct {
	Query          string
	Status         string
	RouteID        string
	DriverID       string
	OnlyUnassigned bool
	SLAHours       int
	Limit          int
	Offset         int
}

func (s *AdminService) GetParcelExceptions(ctx context.Context, params GetParcelExceptionsParams) ([]AdminParcelSummary, error) {
	slaHours := params.SLAHours
	if slaHours <= 0 {
		slaHours = 8
	}
	limit := params.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	cutoff := time.Now().Add(-time.Duration(slaHours) * time.Hour)

	type row struct {
		ID            uuid.UUID  `gorm:"column:id"`
		SenderID      uuid.UUID  `gorm:"column:sender_id"`
		DriverID      *uuid.UUID `gorm:"column:driver_id"`
		ReceiverPhone string     `gorm:"column:receiver_phone"`
		Description   string     `gorm:"column:description"`
		DeclaredValue float64    `gorm:"column:declared_value"`
		Status        string     `gorm:"column:status"`
		RouteID       uuid.UUID  `gorm:"column:route_id"`
		CreatedAt     time.Time  `gorm:"column:created_at"`
		UpdatedAt     time.Time  `gorm:"column:updated_at"`
		PaymentAmount *float64   `gorm:"column:payment_amount"`
		PaymentStatus *string    `gorm:"column:payment_status"`
		LastEventAt   *time.Time `gorm:"column:last_event_at"`
		LastEventType *string    `gorm:"column:last_event_type"`
	}

	where := []string{
		"p.status <> 'DELIVERED'",
		"COALESCE(le.created_at, p.updated_at, p.created_at) <= ?",
	}
	args := []any{cutoff}

	if st := strings.TrimSpace(params.Status); st != "" {
		where = append(where, "p.status = ?")
		args = append(args, st)
	}
	if rid := strings.TrimSpace(params.RouteID); rid != "" {
		where = append(where, "p.route_id = ?")
		args = append(args, rid)
	}
	if did := strings.TrimSpace(params.DriverID); did != "" {
		where = append(where, "p.driver_id = ?")
		args = append(args, did)
	}
	if params.OnlyUnassigned {
		where = append(where, "p.driver_id IS NULL")
	}
	if q := strings.TrimSpace(params.Query); q != "" {
		where = append(where, "(CAST(p.id AS TEXT) ILIKE ? OR p.receiver_phone ILIKE ?)")
		args = append(args, q+"%", "%"+q+"%")
	}

	sql := `
		WITH last_events AS (
			SELECT DISTINCT ON (parcel_id) parcel_id, event_type, created_at
			FROM parcel_tracking_events
			ORDER BY parcel_id, created_at DESC
		)
		SELECT
			p.id, p.sender_id, p.driver_id, p.receiver_phone, p.description, p.declared_value, p.status, p.route_id, p.created_at, p.updated_at,
			pay.amount AS payment_amount, pay.status AS payment_status,
			le.created_at AS last_event_at, le.event_type AS last_event_type
		FROM parcels p
		LEFT JOIN payments pay ON pay.parcel_id = p.id
		LEFT JOIN last_events le ON le.parcel_id = p.id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY COALESCE(le.created_at, p.updated_at, p.created_at) ASC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	var rows []row
	if err := s.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]AdminParcelSummary, 0, len(rows))
	for _, r := range rows {
		var pay *AdminPaymentSummary
		if r.PaymentStatus != nil {
			amt := 0.0
			if r.PaymentAmount != nil {
				amt = *r.PaymentAmount
			}
			pay = &AdminPaymentSummary{Amount: amt, Status: *r.PaymentStatus}
		}

		summary := AdminParcelSummary{
			ID:            r.ID,
			SenderID:      r.SenderID,
			DriverID:      r.DriverID,
			ReceiverPhone: r.ReceiverPhone,
			Description:   r.Description,
			DeclaredValue: r.DeclaredValue,
			Status:        r.Status,
			RouteID:       r.RouteID,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
			Payment:       pay,
			LastEventAt:   r.LastEventAt,
			AgeMinutes:    int(time.Since(r.CreatedAt).Minutes()),
			SlaBreached:   true,
		}
		if r.LastEventType != nil {
			summary.LastEventType = *r.LastEventType
		}
		out = append(out, summary)
	}
	return out, nil
}

func (s *AdminService) GetParcelDetail(ctx context.Context, parcelID string) (*AdminParcelDetail, error) {
	var parcel parcels.Parcel
	if err := s.db.WithContext(ctx).
		Preload("Payment").
		Preload("TrackingEvents", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		First(&parcel, "id = ?", parcelID).Error; err != nil {
		return nil, err
	}

	var notes []parcels.ParcelNote
	_ = s.db.WithContext(ctx).Where("parcel_id = ?", parcel.ID).Order("created_at desc").Find(&notes).Error

	return &AdminParcelDetail{
		Parcel:         parcelToAdminSummary(parcel),
		TrackingEvents: parcel.TrackingEvents,
		Notes:          notes,
	}, nil
}

func (s *AdminService) GetParcelEvents(ctx context.Context, parcelID string) ([]parcels.ParcelTrackingEvent, error) {
	var events []parcels.ParcelTrackingEvent
	err := s.db.WithContext(ctx).Where("parcel_id = ?", parcelID).Order("created_at asc").Find(&events).Error
	return events, err
}

func (s *AdminService) AddParcelNote(ctx context.Context, parcelID string, adminID uuid.UUID, note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return errors.New("note is required")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var parcel parcels.Parcel
		if err := tx.First(&parcel, "id = ?", parcelID).Error; err != nil {
			return err
		}

		newNote := parcels.ParcelNote{
			ParcelID: parcel.ID,
			AdminID:  adminID,
			Note:     note,
		}
		if err := tx.Create(&newNote).Error; err != nil {
			return err
		}

		return tx.Create(&AuditEvent{
			AdminID:  adminID,
			Action:   "PARCEL_NOTE",
			EntityID: parcel.ID.String(),
			Details:  note,
		}).Error
	})
}

func (s *AdminService) NotifyParcelDriver(ctx context.Context, parcelID string, adminID uuid.UUID, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "Driver notified by admin"
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var parcel parcels.Parcel
		if err := tx.First(&parcel, "id = ?", parcelID).Error; err != nil {
			return err
		}
		if parcel.DriverID == nil {
			return errors.New("parcel has no assigned driver")
		}

		// MVP: record an operational event. Actual notification delivery (push/SMS) comes later.
		if err := tx.Create(&parcels.ParcelTrackingEvent{
			ParcelID:  parcel.ID,
			EventType: "DRIVER_NOTIFIED_BY_ADMIN",
			Remarks:   message,
		}).Error; err != nil {
			return err
		}

		return tx.Create(&AuditEvent{
			AdminID:  adminID,
			Action:   "PARCEL_NOTIFY_DRIVER",
			EntityID: parcel.ID.String(),
			Details:  message,
		}).Error
	})
}

func (s *AdminService) OverrideParcel(ctx context.Context, parcelID string, adminID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("reason is required")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var parcel parcels.Parcel
		if err := tx.First(&parcel, "id = ?", parcelID).Error; err != nil {
			return err
		}

		parcel.Status = "DELIVERED"
		if err := tx.Save(&parcel).Error; err != nil {
			return err
		}

		event := parcels.ParcelTrackingEvent{
			ParcelID:  parcel.ID,
			EventType: "DELIVERED_OVERRIDE",
			Remarks:   "Admin Override: " + reason,
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}

		return tx.Create(&AuditEvent{
			AdminID:  adminID,
			Action:   "PARCEL_OVERRIDE_DELIVERED",
			EntityID: parcel.ID.String(),
			Details:  reason,
		}).Error
	})
}

func (s *AdminService) CreateParcelClaim(ctx context.Context, parcelID string, adminID uuid.UUID, claimType string, description string) (*parcels.ParcelClaim, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return nil, errors.New("description is required")
	}
	claimType = strings.TrimSpace(strings.ToUpper(claimType))
	switch claimType {
	case string(parcels.ParcelClaimTypeLost),
		string(parcels.ParcelClaimTypeDamaged),
		string(parcels.ParcelClaimTypeDelayed),
		string(parcels.ParcelClaimTypeWrongRecipient):
	default:
		return nil, errors.New("invalid claim type")
	}

	var created parcels.ParcelClaim
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var parcel parcels.Parcel
		if err := tx.First(&parcel, "id = ?", parcelID).Error; err != nil {
			return err
		}

		created = parcels.ParcelClaim{
			ParcelID:         parcel.ID,
			CreatedByAdminID: adminID,
			Type:             parcels.ParcelClaimType(claimType),
			Status:           parcels.ParcelClaimStatusOpen,
			Description:      description,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}

		return tx.Create(&AuditEvent{
			AdminID:  adminID,
			Action:   "PARCEL_CLAIM_CREATE",
			EntityID: created.ID.String(),
			Details:  "parcel_id=" + parcel.ID.String() + " type=" + claimType,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *AdminService) ListParcelClaims(ctx context.Context, parcelID string) ([]parcels.ParcelClaim, error) {
	var claims []parcels.ParcelClaim
	err := s.db.WithContext(ctx).Where("parcel_id = ?", parcelID).Order("created_at desc").Find(&claims).Error
	return claims, err
}

type ListClaimsParams struct {
	Status string
	Type   string
	Limit  int
	Offset int
}

func (s *AdminService) ListClaims(ctx context.Context, params ListClaimsParams) ([]parcels.ParcelClaim, error) {
	var claims []parcels.ParcelClaim
	query := s.db.WithContext(ctx).Model(&parcels.ParcelClaim{})
	if params.Status != "" {
		query = query.Where("status = ?", strings.ToUpper(params.Status))
	}
	if params.Type != "" {
		query = query.Where("type = ?", strings.ToUpper(params.Type))
	}
	limit := params.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}
	err := query.Order("created_at desc").Limit(limit).Offset(offset).Find(&claims).Error
	return claims, err
}

func (s *AdminService) UpdateClaimStatus(ctx context.Context, claimID string, adminID uuid.UUID, status string, resolutionNotes string) error {
	status = strings.TrimSpace(strings.ToUpper(status))
	switch status {
	case string(parcels.ParcelClaimStatusOpen),
		string(parcels.ParcelClaimStatusInReview),
		string(parcels.ParcelClaimStatusResolved),
		string(parcels.ParcelClaimStatusRejected):
	default:
		return errors.New("invalid claim status")
	}

	resolutionNotes = strings.TrimSpace(resolutionNotes)
	return s.db.Transaction(func(tx *gorm.DB) error {
		var claim parcels.ParcelClaim
		if err := tx.First(&claim, "id = ?", claimID).Error; err != nil {
			return err
		}

		if err := tx.Model(&parcels.ParcelClaim{}).
			Where("id = ?", claim.ID).
			Updates(map[string]any{"status": parcels.ParcelClaimStatus(status), "resolution_notes": resolutionNotes}).Error; err != nil {
			return err
		}

		return tx.Create(&AuditEvent{
			AdminID:  adminID,
			Action:   "PARCEL_CLAIM_UPDATE",
			EntityID: claim.ID.String(),
			Details:  "status=" + status,
		}).Error
	})
}

type ClaimAttachmentMeta struct {
	ID        uuid.UUID `json:"id"`
	ClaimID   uuid.UUID `json:"claim_id"`
	FileName  string    `json:"file_name"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *AdminService) ListClaimAttachments(ctx context.Context, claimID string) ([]ClaimAttachmentMeta, error) {
	var metas []ClaimAttachmentMeta
	err := s.db.WithContext(ctx).
		Model(&parcels.ParcelClaimAttachment{}).
		Select("id, claim_id, file_name, mime_type, size_bytes, created_at").
		Where("claim_id = ?", claimID).
		Order("created_at desc").
		Scan(&metas).Error
	return metas, err
}

func (s *AdminService) AddClaimAttachment(ctx context.Context, claimID string, adminID uuid.UUID, fileName string, mimeType string, content []byte) (*ClaimAttachmentMeta, error) {
	fileName = strings.TrimSpace(fileName)
	mimeType = strings.TrimSpace(mimeType)
	if fileName == "" {
		return nil, errors.New("file_name is required")
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if len(content) == 0 {
		return nil, errors.New("file content is required")
	}

	var created parcels.ParcelClaimAttachment
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var claim parcels.ParcelClaim
		if err := tx.First(&claim, "id = ?", claimID).Error; err != nil {
			return err
		}

		created = parcels.ParcelClaimAttachment{
			ClaimID:           claim.ID,
			UploadedByAdminID: adminID,
			FileName:          fileName,
			MimeType:          mimeType,
			SizeBytes:         int64(len(content)),
			Content:           content,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}

		return tx.Create(&AuditEvent{
			AdminID:  adminID,
			Action:   "PARCEL_CLAIM_ATTACHMENT_ADD",
			EntityID: created.ID.String(),
			Details:  "claim_id=" + claim.ID.String() + " file=" + fileName,
		}).Error
	})
	if err != nil {
		return nil, err
	}

	return &ClaimAttachmentMeta{
		ID:        created.ID,
		ClaimID:   created.ClaimID,
		FileName:  created.FileName,
		MimeType:  created.MimeType,
		SizeBytes: created.SizeBytes,
		CreatedAt: created.CreatedAt,
	}, nil
}

func (s *AdminService) GetClaimAttachment(ctx context.Context, attachmentID string) (*parcels.ParcelClaimAttachment, error) {
	var a parcels.ParcelClaimAttachment
	if err := s.db.WithContext(ctx).First(&a, "id = ?", attachmentID).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Transport Admin Logic
func (s *AdminService) GetLiveFleet(ctx context.Context) ([]drivers.DriverSession, error) {
	var sessions []drivers.DriverSession
	err := s.db.Where("status = ?", "ONLINE").Preload("DriverProfile").Order("last_heartbeat desc").Find(&sessions).Error
	return sessions, err
}

func (s *AdminService) PingDriver(ctx context.Context, driverID string) error {
	// In a complete implementation, this would send an FCM push notification or SMS
	// to "wake up" the driver's device. For MVP we record success.
	return nil
}

func (s *AdminService) GetRoutes(ctx context.Context) ([]routes.Route, error) {
	var rs []routes.Route
	err := s.db.Preload("Stops").Order("created_at desc").Find(&rs).Error
	return rs, err
}

func (s *AdminService) GetStops(ctx context.Context) ([]routes.Stop, error) {
	var ss []routes.Stop
	err := s.db.Order("name asc").Find(&ss).Error
	return ss, err
}

func (s *AdminService) CreateRoute(ctx context.Context, name, shortName, desc string, fare float64) (*routes.Route, error) {
	route := routes.Route{
		Name:        name,
		ShortName:   shortName,
		Description: desc,
		BaseFare:    fare,
	}
	err := s.db.Create(&route).Error
	return &route, err
}

func (s *AdminService) CreateStop(ctx context.Context, name, pointWKT string) (*routes.Stop, error) {
	stop := routes.Stop{
		Name:     name,
		Location: pointWKT, // e.g., "POINT(lng lat)"
	}
	err := s.db.Create(&stop).Error
	return &stop, err
}

// Finance Admin Logic
func (s *AdminService) GetUnpaidLedger(ctx context.Context) ([]ledger.LedgerEntry, error) {
	var entries []ledger.LedgerEntry
	err := s.db.Where("status = ?", "UNPAID").Order("created_at asc").Find(&entries).Error
	return entries, err
}

func (s *AdminService) ExecutePayouts(ctx context.Context, driverIDs []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, driverID := range driverIDs {
			var total float64
			if err := tx.Model(&ledger.LedgerEntry{}).Where("driver_id = ? AND status = ?", driverID, "UNPAID").Select("COALESCE(SUM(amount), 0)").Scan(&total).Error; err != nil {
				return err
			}

			if total > 0 {
				if err := tx.Model(&ledger.LedgerEntry{}).Where("driver_id = ? AND status = ?", driverID, "UNPAID").Update("status", "PAID").Error; err != nil {
					return err
				}

				parsedID, err := uuid.Parse(driverID)
				if err != nil {
					continue
				}

				batch := ledger.PayoutBatch{
					DriverID:     parsedID,
					TotalAmount:  total,
					MPesaReceipt: "MOCK_B2C_" + uuid.NewString()[:8],
				}
				if err := tx.Create(&batch).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// Support Admin Logic
func (s *AdminService) SearchUsers(ctx context.Context, query string) ([]identity.User, error) {
	var users []identity.User
	err := s.db.Where("phone_number LIKE ? OR full_name ILIKE ?", "%"+query+"%", "%"+query+"%").Limit(50).Find(&users).Error
	return users, err
}

func (s *AdminService) RescueAuth(ctx context.Context, userID string) (string, error) {
	pin := "123456" // Mock fixed pin for MVC manual bypass
	err := s.db.Model(&identity.User{}).Where("id = ?", userID).Update("is_active", true).Error
	return pin, err
}

// Audit & Overseer Logic
func (s *AdminService) LogAction(ctx context.Context, adminID uuid.UUID, action, entityID, details string) error {
	event := AuditEvent{
		AdminID:  adminID,
		Action:   action,
		EntityID: entityID,
		Details:  details,
	}
	return s.db.Create(&event).Error
}

func (s *AdminService) GetAuditLogs(ctx context.Context, limit int) ([]AuditEvent, error) {
	var logs []AuditEvent
	err := s.db.Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}

// MANAGEMENT

func (s *AdminService) ListAdmins(ctx context.Context) ([]AdminUserDetail, error) {
	var details []AdminUserDetail
	err := s.db.Table("admin_users").
		Select("admin_users.id, admin_users.user_id, admin_credentials.email, admin_credentials.name, admin_users.role, admin_users.department, admin_users.status, admin_users.created_at").
		Joins("left join admin_credentials on admin_credentials.admin_user_id = admin_users.id").
		Scan(&details).Error
	return details, err
}

func (s *AdminService) CreateAdmin(ctx context.Context, email, name, role, department, password string) (*AdminUser, error) {
	var adminUser AdminUser
	if !IsCanonicalAdminRole(role) {
		return nil, errors.New("invalid admin role")
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create Identity User (placeholder phone)
		idUser := identity.User{
			PhoneNumber: "ADMIN_" + uuid.NewString()[:8],
			FullName:    name,
		}
		if err := tx.Create(&idUser).Error; err != nil {
			return err
		}

		// 2. Create Admin User
		adminUser = AdminUser{
			UserID:     idUser.ID,
			Role:       role,
			Department: department,
			Status:     "ACTIVE",
		}
		if err := tx.Create(&adminUser).Error; err != nil {
			return err
		}

		// 3. Create Admin Credentials with hashed password
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		cred := AdminCredential{
			AdminUserID:  adminUser.ID,
			Email:        email,
			Name:         name,
			PasswordHash: string(hash),
		}
		return tx.Create(&cred).Error
	})

	return &adminUser, err
}

func (s *AdminService) UpdateAdmin(ctx context.Context, id, role, department, status string) error {
	if !IsCanonicalAdminRole(role) {
		return errors.New("invalid admin role")
	}
	if !IsCanonicalAdminStatus(status) {
		return errors.New("invalid admin status")
	}
	return s.db.Model(&AdminUser{}).Where("id = ?", id).Updates(map[string]interface{}{
		"role":       role,
		"department": department,
		"status":     status,
	}).Error
}
