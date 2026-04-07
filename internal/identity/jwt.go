package identity

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/shukisha/backend/internal/platform/config"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email,omitempty"`
	Name   string    `json:"name,omitempty"`
	Role   string    `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken creates a basic token for mobile users (OTP flow)
func GenerateToken(userID uuid.UUID) (string, error) {
	return GenerateAdminToken(userID, "", "", "")
}

// GenerateAdminToken creates an enriched token for admin users (email+password flow)
func GenerateAdminToken(userID uuid.UUID, email, name, role string) (string, error) {
	secret := []byte(config.GetEnv("JWT_SECRET", "shukisha_secret_change_me"))

	claims := Claims{
		UserID: userID,
		Email:  email,
		Name:   name,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "shukisha",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func VerifyToken(tokenString string) (*Claims, error) {
	secret := []byte(config.GetEnv("JWT_SECRET", "shukisha_secret_change_me"))

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
