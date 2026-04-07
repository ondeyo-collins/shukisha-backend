package admin

import (
	"context"
	"errors"

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

func (s *AdminService) GetParcels(ctx context.Context, status string) ([]parcels.Parcel, error) {
	var results []parcels.Parcel
	query := s.db.Model(&parcels.Parcel{}).Preload("TrackingEvents").Preload("Payment")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("created_at desc").Find(&results).Error
	return results, err
}

func (s *AdminService) OverrideParcel(ctx context.Context, parcelID string, reason string) error {
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
		return tx.Create(&event).Error
	})
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
