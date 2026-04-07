package routes

import (
	"context"
	"fmt"
	"gorm.io/gorm"
)

type RouteService struct {
	db *gorm.DB
}

func NewRouteService(database *gorm.DB) *RouteService {
	return &RouteService{db: database}
}

// FindNearestStops returns stops within the given radius (meters) using PostGIS
func (s *RouteService) FindNearestStops(ctx context.Context, lat, lng float64, radius int) ([]Stop, error) {
	var stops []Stop
	
	// ST_DWithin is efficient for radius searches on geography types
	// Location is GEOGRAPHY(POINT, 4326), units are in meters
	err := s.db.WithContext(ctx).
		Where("ST_DWithin(location, ST_SetSRID(ST_MakePoint(?, ?), 4326), ?)", lng, lat, radius).
		Order(fmt.Sprintf("location <-> ST_SetSRID(ST_MakePoint(%f, %f), 4326)", lng, lat)).
		Limit(10).
		Find(&stops).Error

	return stops, err
}

// SearchRoutes searches for routes by name or short name
func (s *RouteService) SearchRoutes(ctx context.Context, query string) ([]Route, error) {
	var routes []Route
	err := s.db.WithContext(ctx).
		Where("name ILIKE ? OR short_name ILIKE ?", "%"+query+"%", "%"+query+"%").
		Find(&routes).Error
	return routes, err
}

// GetRouteWithStops retrieves a single route with its stops in the correct order
func (s *RouteService) GetRouteWithStops(ctx context.Context, routeID string) (*Route, error) {
	var route Route
	
	// Complex join to maintain StopOrder
	err := s.db.WithContext(ctx).
		Preload("Stops", func(db *gorm.DB) *gorm.DB {
			return db.Joins("JOIN route_stops ON route_stops.stop_id = stops.id").
				Order("route_stops.stop_order ASC")
		}).
		First(&route, "id = ?", routeID).Error

	if err != nil {
		return nil, err
	}
	return &route, nil
}
