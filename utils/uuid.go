package utils

import (
	"github.com/google/uuid"
)

// GenerateID returns a new unique identifier string
func GenerateID() string {
	return uuid.New().String()
}
