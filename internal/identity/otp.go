package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
)

// GenerateOTP creates a 6-digit numeric OTP string
func GenerateOTP() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n), nil
}

// HashOTP hashes the code with a salt using SHA-256
func HashOTP(code, salt string) string {
	hash := sha256.New()
	hash.Write([]byte(code + salt))
	return hex.EncodeToString(hash.Sum(nil))
}

// GenerateSalt creates a random 16-character salt
func GenerateSalt() (string, error) {
	salt := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}
	return hex.EncodeToString(salt), nil
}

// VerifyOTP compares a raw code against a hash and salt
func VerifyOTP(code, salt, hash string) bool {
	return HashOTP(code, salt) == hash
}
