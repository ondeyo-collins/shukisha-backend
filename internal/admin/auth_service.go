package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shukisha/backend/internal/identity"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminAuthService handles email+password login for admin users
type AdminAuthService struct {
	db *gorm.DB
}

func NewAdminAuthService(db *gorm.DB) *AdminAuthService {
	return &AdminAuthService{db: db}
}

// LoginWithPassword verifies email+password and returns a signed admin JWT
func (s *AdminAuthService) LoginWithPassword(ctx context.Context, email, password string) (string, *AdminUser, *AdminCredential, error) {
	var cred AdminCredential
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&cred).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil, nil, errors.New("invalid credentials")
	}
	if err != nil {
		return "", nil, nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password)); err != nil {
		return "", nil, nil, errors.New("invalid credentials")
	}

	// Fetch the linked AdminUser for role
	var adminUser AdminUser
	if err := s.db.WithContext(ctx).Where("id = ?", cred.AdminUserID).First(&adminUser).Error; err != nil {
		return "", nil, nil, fmt.Errorf("admin user not found: %w", err)
	}

	token, err := identity.GenerateAdminToken(adminUser.UserID, cred.Email, cred.Name, adminUser.Role)
	if err != nil {
		return "", nil, nil, err
	}

	return token, &adminUser, &cred, nil
}

// CreateAdminCredential creates a new admin account with hashed password (used for seeding/setup)
func (s *AdminAuthService) CreateAdminCredential(ctx context.Context, adminUserID uuid.UUID, email, name, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	cred := AdminCredential{
		AdminUserID:  adminUserID,
		Email:        email,
		Name:         name,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return s.db.WithContext(ctx).Create(&cred).Error
}
