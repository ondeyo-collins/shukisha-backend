package routes

import (
	"time"

	"github.com/google/uuid"
)

type Route struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string      `gorm:"size:255;not null" json:"name"`
	ShortName   string      `gorm:"size:20" json:"short_name"`
	Description string      `gorm:"type:text" json:"description"`
	BaseFare    float64     `gorm:"type:decimal(10,2);default:0" json:"base_fare"`
	Stops       []Stop      `gorm:"many2many:route_stops;joinForeignKey:route_id;joinReferences:stop_id" json:"stops"`
	CreatedAt   time.Time   `json:"created_at"`
}

type Stop struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	// Geography type as string (WKT format: POINT(lng lat))
	Location  string    `gorm:"type:geography(POINT, 4326)" json:"location"`
	CreatedAt time.Time `json:"created_at"`
}

type RouteStop struct {
	RouteID   uuid.UUID `gorm:"primaryKey"`
	StopID    uuid.UUID `gorm:"primaryKey"`
	StopOrder int       `gorm:"not null"`
}

// TableName overrides the table name for RouteStop join table
func (RouteStop) TableName() string {
	return "route_stops"
}
