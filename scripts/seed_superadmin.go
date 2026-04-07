package main

import (
	"context"
	"fmt"
	"log"

	"github.com/shukisha/backend/internal/admin"
	"github.com/shukisha/backend/internal/platform/db"
)

func main() {
	database, err := db.Init()
	if err != nil {
		log.Fatal(err)
	}

	// Auto-Migrate Schema
	database.AutoMigrate(&admin.AdminUser{}, &admin.AdminCredential{})

	adminSvc := admin.NewAdminService(database)

	ctx := context.Background()

	// Check if superadmin exists
	var existing admin.AdminCredential
	err = database.Where("email = ?", "superadmin@shukisha.com").First(&existing).Error
	if err == nil {
		fmt.Println("Superadmin already exists.")
		return
	}

	fmt.Println("Seeding Superadmin...")
	
	// Create Superadmin
	newAdmin, err := adminSvc.CreateAdmin(ctx, "superadmin@shukisha.com", "Main Director", "SUPER_ADMIN", "Global HQ", "admin123")
	if err != nil {
		log.Fatalf("Failed to create superadmin: %v", err)
	}

	fmt.Printf("Superadmin created: %s (ID: %s)\n", "superadmin@shukisha.com", newAdmin.ID)
	fmt.Println("Password: admin123")
}
