// Package password membungkus bcrypt agar pemakaiannya konsisten.
package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// MaxLength adalah batas byte yang bisa diproses bcrypt.
const MaxLength = 72

// Hash menghasilkan bcrypt hash dari password plaintext.
func Hash(plain string) (string, error) {
	if len(plain) > MaxLength {
		return "", fmt.Errorf("password exceeds %d characters", MaxLength)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashed), nil
}

// Verify membandingkan password plaintext dengan hash yang tersimpan.
func Verify(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
