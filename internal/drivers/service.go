package drivers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type DriverService struct {
	db    *gorm.DB
	cache *redis.Client
}

func NewDriverService(database *gorm.DB, cacheClient *redis.Client) *DriverService {
	return &DriverService{
		db:    database,
		cache: cacheClient,
	}
}

// Onboard handles driver profile creation
func (s *DriverService) Onboard(ctx context.Context, driverID uuid.UUID, plate, sacco string) error {
	profile := DriverProfile{
		ID:                 driverID,
		VehiclePlate:       plate,
		SaccoName:          sacco,
		VerificationStatus: "PENDING",
	}
	return s.db.WithContext(ctx).Save(&profile).Error
}

// StartSession marks a driver as online on a specific route
func (s *DriverService) StartSession(ctx context.Context, driverID uuid.UUID, routeID uuid.UUID) (*DriverSession, error) {
	session := DriverSession{
		DriverID:      driverID,
		RouteID:       routeID,
		Status:        "ONLINE",
		LastHeartbeat: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// Heartbeat updates the driver's live location in Redis and Postgres
func (s *DriverService) Heartbeat(ctx context.Context, driverID uuid.UUID, lat, lng float64) error {
	// 1. Update Real-time position in Redis (GEOADD)
	// We use a global key for all online drivers to allow proximity queries later
	err := s.cache.GeoAdd(ctx, "driver_locations", &redis.GeoLocation{
		Name:      driverID.String(),
		Longitude: lng,
		Latitude:  lat,
	}).Err()
	if err != nil {
		return fmt.Errorf("redis geoadd failed: %w", err)
	}

	// 2. Update status in Postgres (LastHeartbeat)
	return s.db.WithContext(ctx).
		Model(&DriverSession{}).
		Where("driver_id = ? AND status = 'ONLINE'", driverID).
		Update("last_heartbeat", time.Now()).Error
}

// StopSession closes the active driver session
func (s *DriverService) StopSession(ctx context.Context, driverID uuid.UUID) error {
	// Remove from Redis live locations
	s.cache.ZRem(ctx, "driver_locations", driverID.String())

	// Close session in DB
	return s.db.WithContext(ctx).
		Model(&DriverSession{}).
		Where("driver_id = ? AND status = 'ONLINE'", driverID).
		Update("status", "CLOSED").Error
}
