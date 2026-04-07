package main

import (
	"fmt"
	"log"

	"github.com/shukisha/backend/internal/admin"
	"github.com/shukisha/backend/internal/platform/db"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	database, err := db.Init()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	email := "admin@shukisha.co.ke"
	password := "CHANGE_ME"

	var cred admin.AdminCredential
	err = database.Where("email = ?", email).First(&cred).Error
	if err != nil {
		fmt.Printf("❌ AdminCredential not found for %s: %v\n", email, err)
		return
	}
	fmt.Printf("✅ Found AdminCredential for %s\n", email)

	err = bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password))
	if err != nil {
		fmt.Printf("❌ Password mismatch: %v\n", err)
	} else {
		fmt.Printf("✅ Password matches!\n")
	}

	var adminUser admin.AdminUser
	err = database.Where("id = ?", cred.AdminUserID).First(&adminUser).Error
	if err != nil {
		fmt.Printf("❌ AdminUser (ID: %s) linked to credential not found: %v\n", cred.AdminUserID, err)
	} else {
		fmt.Printf("✅ Found AdminUser linked to credential. Role: %s, UserID: %s\n", adminUser.Role, adminUser.UserID)
	}
}
