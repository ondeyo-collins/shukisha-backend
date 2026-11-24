package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/shukisha/backend/internal/admin"
	"github.com/shukisha/backend/internal/identity"
	"github.com/shukisha/backend/internal/platform/db"
)

func main() {
	// Initialize Database
	database, err := db.Init()
	if err != nil {
		log.Fatalf("Could not initialize database: %v", err)
	}

	// Auto-Migrate Admin Tables
	fmt.Println("Migrating admin tables...")
	err = database.AutoMigrate(&identity.User{}, &admin.AdminUser{}, &admin.AdminCredential{})
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// 1. Create a base Identity User
	user := identity.User{
		PhoneNumber: "+254700000000", // Placeholder phone
	}
	if err := database.Where(identity.User{PhoneNumber: user.PhoneNumber}).FirstOrCreate(&user).Error; err != nil {
		log.Fatalf("Failed to create identity user: %v", err)
	}

	// 2. Create the Admin User profile
	adminUser := admin.AdminUser{
		UserID: user.ID,
		Role:   "SUPER_ADMIN",
	}
	if err := database.Where(admin.AdminUser{UserID: user.ID}).FirstOrCreate(&adminUser).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	// 3. Create the Email+Password Credential
	authSvc := admin.NewAdminAuthService(database)
	email := "admin@shukisha.co.ke"
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if password == "" {
		log.Fatal("SEED_ADMIN_PASSWORD must be set")
	}
	name := "Super Admin"

	err = authSvc.CreateAdminCredential(context.Background(), adminUser.ID, email, name, password)
	if err != nil {
		// If it already exists, just notify
		fmt.Printf("Note: Admin credential for %s might already exist: %v\n", email, err)
	} else {
		fmt.Printf("Successfully created SUPER_ADMIN: %s\n", email)
	}
}
