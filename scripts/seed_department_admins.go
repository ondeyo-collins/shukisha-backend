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

type seedAdmin struct {
	email      string
	name       string
	role       string
	department string
}

func main() {
	database, err := db.Init()
	if err != nil {
		log.Fatal(err)
	}

	// Ensure core tables exist.
	if err := database.AutoMigrate(&identity.User{}, &admin.AdminUser{}, &admin.AdminCredential{}); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if password == "" {
		password = "admin123"
		fmt.Println("Warning: SEED_ADMIN_PASSWORD not set; using default password: admin123")
	}

	toSeed := []seedAdmin{
		{email: "logistics@shukisha.com", name: "Logistics Admin", role: admin.RoleLogistics, department: "Logistics"},
		{email: "transport@shukisha.com", name: "Transport Admin", role: admin.RoleTransport, department: "Transport"},
		{email: "finance@shukisha.com", name: "Finance Admin", role: admin.RoleFinance, department: "Finance"},
		{email: "support@shukisha.com", name: "Support Admin", role: admin.RoleSupport, department: "Support"},
	}

	adminSvc := admin.NewAdminService(database)
	ctx := context.Background()

	for _, a := range toSeed {
		if !admin.IsCanonicalAdminRole(a.role) {
			log.Fatalf("invalid seed role %q for %s", a.role, a.email)
		}

		var existing admin.AdminCredential
		if err := database.WithContext(ctx).Where("email = ?", a.email).First(&existing).Error; err == nil {
			fmt.Printf("Exists: %s (%s)\n", a.email, a.role)
			continue
		}

		newAdmin, err := adminSvc.CreateAdmin(ctx, a.email, a.name, a.role, a.department, password)
		if err != nil {
			log.Fatalf("failed to create %s: %v", a.email, err)
		}

		fmt.Printf("Created: %s (ID: %s, role: %s)\n", a.email, newAdmin.ID, newAdmin.Role)
	}

	fmt.Println("Done.")
}

