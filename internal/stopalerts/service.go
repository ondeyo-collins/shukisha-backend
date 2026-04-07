package stopalerts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StopAlertService struct {
	db *gorm.DB
}

func NewStopAlertService(database *gorm.DB) *StopAlertService {
	return &StopAlertService{db: database}
}

// CreateAlert stores a new stop alert for a passenger
func (s *StopAlertService) CreateAlert(ctx context.Context, userID uuid.UUID, routeID, stopID string, driverID *string) (*StopAlert, error) {
	rID, _ := uuid.Parse(routeID)
	sID, _ := uuid.Parse(stopID)
	
	alert := StopAlert{
		UserID:  userID,
		RouteID: rID,
		StopID:  sID,
		Status:  "ARMED",
	}

	if driverID != nil && *driverID != "" {
		dID, _ := uuid.Parse(*driverID)
		alert.DriverID = &dID
	}

	if err := s.db.WithContext(ctx).Create(&alert).Error; err != nil {
		return nil, err
	}
	return &alert, nil
}

// GetDriverAlerts finds all Armed alerts on the driver's active route
func (s *StopAlertService) GetDriverAlerts(ctx context.Context, driverID uuid.UUID) ([]StopAlert, error) {
	var session struct {
		RouteID uuid.UUID
	}
	if err := s.db.Table("driver_sessions").Where("driver_id = ? AND status = ?", driverID, "ONLINE").Select("route_id").Scan(&session).Error; err != nil {
		return nil, errors.New("no active driver session found")
	}

	var alerts []StopAlert
	err := s.db.Preload("Stop").Where("route_id = ? AND status = ?", session.RouteID, "ARMED").Find(&alerts).Error
	return alerts, err
}

// AckAlert acknowledges an alert for the driver
func (s *StopAlertService) AckAlert(ctx context.Context, driverID, alertID uuid.UUID) error {
	result := s.db.Model(&StopAlert{}).Where("id = ? AND status = ?", alertID, "ARMED").Updates(map[string]interface{}{
		"status":    "ACKNOWLEDGED",
		"driver_id": driverID,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("alert not found or already acknowledged")
	}
	return nil
}

// GetActiveAlerts returns armed alerts for a user
func (s *StopAlertService) GetActiveAlerts(ctx context.Context, userID uuid.UUID) ([]StopAlert, error) {
	var alerts []StopAlert
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND status IN ('ARMED', 'APPROACHING')", userID).
		Preload("Stop").
		Find(&alerts).Error
	return alerts, err
}

// ProcessProximity checks online drivers near targeted stops for ARMED alerts
func (s *StopAlertService) ProcessProximity(ctx context.Context) error {
	var alerts []StopAlert
	// Find all ARMED alerts and join the stop location
	if err := s.db.WithContext(ctx).Where("status = 'ARMED'").Preload("Stop").Find(&alerts).Error; err != nil {
		return err
	}

	for _, alert := range alerts {
		if alert.Stop.ID == uuid.Nil {
			continue // Avoid issues if Stop record is missing
		}

		// Query for online drivers on this route near the stop location
		var count int64
		err := s.db.Table("driver_sessions").
			Select("count(*)").
			Where("status = 'ONLINE' AND route_id = ?", alert.RouteID).
			Where("EXISTS (SELECT 1 FROM driver_sessions ds WHERE ds.driver_id = driver_sessions.driver_id AND ST_DWithin(ST_GeomFromText(ST_AsText(location), 4326), ST_GeomFromText(?, 4326), ?))", alert.Stop.Location, alert.TriggerRadius).
			Count(&count).Error

		if err == nil && count > 0 {
			now := time.Now()
			s.db.Model(&alert).Updates(map[string]interface{}{
				"status":      "APPROACHING",
				"triggered_at": &now,
			})
			fmt.Printf("ALERT TRIGGERED: User %s for Stop %s\n", alert.UserID, alert.Stop.Name)
		}
	}

	return nil
}
