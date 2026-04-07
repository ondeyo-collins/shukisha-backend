package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shukisha/backend/internal/ledger"
	"github.com/shukisha/backend/internal/parcels"
	"github.com/shukisha/backend/internal/platform/db"
	"github.com/shukisha/backend/internal/routes"
)

func main() {
	fmt.Println("Seeding sample transit and logistics data...")

	// Initialize Database
	database, err := db.Init()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 1. Create a Route (Route 111)
	route := routes.Route{
		Name:      "Ngong Road - Nairobi CBD",
		ShortName: "111",
		BaseFare:  450,
	}
	if err := database.Create(&route).Error; err != nil {
		log.Printf("Warning: Could not create route (maybe exists): %v", err)
	}

	// 2. Create Stops
	stops := []routes.Stop{
		{
			Name:     "Kencom/Ambassadeur",
			Location: "POINT(36.8233 -1.2858)",
		},
		{
			Name:     "Adams Arcade",
			Location: "POINT(36.7845 -1.2995)",
		},
	}

	for i := range stops {
		if err := database.Create(&stops[i]).Error; err != nil {
			log.Printf("Warning: Could not create stop %s: %v", stops[i].Name, err)
			continue
		}
		routeStop := routes.RouteStop{
			RouteID:   route.ID,
			StopID:    stops[i].ID,
			StopOrder: i + 1,
		}
		database.Create(&routeStop)
	}

	// 3. Create a Parcel (Logistics testing)
	p := parcels.Parcel{
		SenderID:      uuid.New(),
		Description:   "Logistics Verification Package",
		ReceiverPhone: "0711223344",
		Status:        "PAID",
		RouteID:       route.ID,
		DeclaredValue: 5000,
		CreatedAt:     time.Now().Add(-1 * time.Hour),
	}
	database.Create(&p)

	// 4. Create Ledger Entry (Finance testing)
	driverID := uuid.New()
	le := ledger.LedgerEntry{
		DriverID:    driverID,
		ParcelID:    p.ID,
		Amount:      450.00,
		Description: "Commission for Verification Trip",
		Status:      "UNPAID",
		CreatedAt:   time.Now().Add(-20 * time.Minute),
	}
	database.Create(&le)

	fmt.Println("Seeding completed successfully! 🚍📦💰")
}
