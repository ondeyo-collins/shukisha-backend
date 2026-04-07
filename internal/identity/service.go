package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shukisha/backend/internal/platform/sms"
	"gorm.io/gorm"
)

type IdentityService struct {
	db          *gorm.DB
	smsProvider sms.Provider
}

func NewIdentityService(database *gorm.DB, smsProvider sms.Provider) *IdentityService {
	return &IdentityService{
		db:          database,
		smsProvider: smsProvider,
	}
}

func (s *IdentityService) RefreshToken(ctx context.Context, oldToken string) (string, error) {
	claims, err := VerifyToken(oldToken)
	if err != nil {
		return "", err
	}
	return GenerateToken(claims.UserID)
}

// RequestOTP generates and stores an OTP for a phone number
func (s *IdentityService) RequestOTP(ctx context.Context, phone string) (string, error) {
	otp, err := GenerateOTP()
	if err != nil {
		return "", err
	}

	salt, _ := GenerateSalt()
	hash := HashOTP(otp, salt)

	challenge := OTPChallenge{
		PhoneNumber: phone,
		CodeHash:    hash,
		Salt:        salt,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	if err := s.db.WithContext(ctx).Create(&challenge).Error; err != nil {
		return "", err
	}

	// Call the SMS Provider (Twilio or Mock)
	msg := fmt.Sprintf("Shukisha: Your verification code is %s. Valid for 5 minutes.", otp)
	if err := s.smsProvider.SendSMS(ctx, phone, msg); err != nil {
		fmt.Printf("Warning: Failed to send SMS to %s: %v\n", phone, err)
		// We still return the OTP for development/testing if SMS fails
	}

	return otp, nil
}

// VerifyOTP checks the code and returns the User if successful
func (s *IdentityService) VerifyOTP(ctx context.Context, phone, code string) (*User, error) {
	var challenge OTPChallenge
	err := s.db.WithContext(ctx).
		Where("phone_number = ? AND expires_at > ? AND verified_at IS NULL", phone, time.Now()).
		Order("created_at DESC").
		First(&challenge).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("OTP expired or not found")
		}
		return nil, err
	}

	if !VerifyOTP(code, challenge.Salt, challenge.CodeHash) {
		s.db.Model(&challenge).Update("attempts", challenge.Attempts+1)
		return nil, errors.New("invalid code")
	}

	// Mark challenge as verified
	now := time.Now()
	s.db.Model(&challenge).Update("verified_at", &now)

	// Get or Create User
	var user User
	err = s.db.WithContext(ctx).Where("phone_number = ?", phone).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = User{
			PhoneNumber: phone,
			LastLogin:   &now,
		}
		if err := s.db.Create(&user).Error; err != nil {
			return nil, err
		}
	} else {
		s.db.Model(&user).Update("last_login", &now)
	}

	return &user, nil
}
