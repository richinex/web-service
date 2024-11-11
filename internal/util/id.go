// internal/util/id.go

package util

import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "github.com/google/uuid"
    "strings"
)

// GenerateID generates a URL-safe, base64 encoded UUID
func GenerateID() string {
    id := uuid.New()
    return base64.RawURLEncoding.EncodeToString(id[:])
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
    b := make([]byte, length)
    if _, err := rand.Read(b); err != nil {
        return "", fmt.Errorf("failed to generate secure token: %w", err)
    }
    return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "="), nil
}