package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/shukisha/backend/internal/admin"
	"github.com/shukisha/backend/internal/identity"
	"github.com/shukisha/backend/internal/platform/db"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	var (
		email      = flag.String("email", "", "Admin email (required)")
		name       = flag.String("name", "", "Admin display name (required)")
		role       = flag.String("role", "SUPER_ADMIN", "Admin role (e.g. SUPER_ADMIN, LOGISTICS, TRANSPORT, FINANCE, SUPPORT)")
		department = flag.String("department", "Global HQ", "Admin department label")
		password   = flag.String("password", "", "Admin password (or set ADMIN_PASSWORD env var)")
		reset      = flag.Bool("reset", false, "If the email already exists, reset password and exit")
	)
	flag.Parse()

	if *email == "" || *name == "" {
		log.Fatalf("missing required flags: --email and --name")
	}

	if !admin.IsCanonicalAdminRole(*role) {
		log.Fatalf("invalid role %q (allowed: SUPER_ADMIN, LOGISTICS, TRANSPORT, FINANCE, SUPPORT)", *role)
	}

	pass := *password
	if pass == "" {
		pass = os.Getenv("ADMIN_PASSWORD")
	}
	if pass == "" {
		log.Fatalf("missing password: pass --password or set ADMIN_PASSWORD")
	}

	database, err := db.Init()
	if err != nil {
		log.Fatal(err)
	}

	// Ensure tables exist (CreateAdmin uses identity + admin tables).
	if err := database.AutoMigrate(&identity.User{}, &admin.AdminUser{}, &admin.AdminCredential{}); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	ctx := context.Background()

	var existing admin.AdminCredential
	err = database.WithContext(ctx).Where("email = ?", *email).First(&existing).Error
	if err == nil {
		if !*reset {
			fmt.Printf("Admin credential already exists for %s (use --reset to reset password)\n", *email)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("failed to hash password: %v", err)
		}

		if err := database.WithContext(ctx).Model(&admin.AdminCredential{}).
			Where("email = ?", *email).
			Updates(map[string]any{"password_hash": string(hash), "name": *name}).Error; err != nil {
			log.Fatalf("failed to reset password: %v", err)
		}

		fmt.Printf("Password reset for %s\n", *email)
		return
	}

	adminSvc := admin.NewAdminService(database)
	newAdmin, err := adminSvc.CreateAdmin(ctx, *email, *name, *role, *department, pass)
	if err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}

	fmt.Printf("Admin created: %s (ID: %s, role: %s)\n", *email, newAdmin.ID, newAdmin.Role)
}
