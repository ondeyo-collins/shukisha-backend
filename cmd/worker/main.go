package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shukisha/backend/internal/parcels"
	"github.com/shukisha/backend/internal/platform/cache"
	"github.com/shukisha/backend/internal/platform/db"
	"github.com/shukisha/backend/internal/platform/sms"
	"github.com/shukisha/backend/internal/stopalerts"
)

func main() {
	fmt.Println("Shukisha Worker starting...")

	// Initialize Database
	if _, err := db.Init(); err != nil {
		log.Fatalf("Critical: Could not initialize database: %v", err)
	}

	// Initialize Cache
	if _, err := cache.Init(); err != nil {
		log.Fatalf("Critical: Could not initialize cache: %v", err)
	}

	// Initialize Alert Service
	alertService := stopalerts.NewStopAlertService(db.DB)
	smsProvider := sms.NewTwilioProvider()
	notifyService := parcels.NewNotificationService(db.DB, smsProvider)

	ctx := context.Background()

	// Proximity Check Loop
	proximityTicker := time.NewTicker(5 * time.Second)
	notifyTicker := time.NewTicker(10 * time.Second)
	defer proximityTicker.Stop()
	defer notifyTicker.Stop()

	for {
		select {
		case <-proximityTicker.C:
			fmt.Println("Worker processing proximity checks...")
			if err := alertService.ProcessProximity(ctx); err != nil {
				fmt.Printf("Error processing proximity: %v\n", err)
			}
		case <-notifyTicker.C:
			if err := notifyService.ProcessOnce(ctx, 20); err != nil {
				fmt.Printf("Error processing notifications: %v\n", err)
			}
		}
	}
}
