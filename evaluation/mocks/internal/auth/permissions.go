//go:build ignore

package auth

import (
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/evaluation/mocks/internal/database"
)

type PermissionLevel int

const (
	PermissionNone  PermissionLevel = 0
	PermissionRead  PermissionLevel = 1
	PermissionWrite PermissionLevel = 2
	PermissionAdmin PermissionLevel = 3
)

type AuthService struct {
	db *database.Database
}

func NewAuthService(db *database.Database) *AuthService {
	return &AuthService{db: db}
}

// CheckUserPermission returns the user's permission level for a resource
func (a *AuthService) CheckUserPermission(userID int, resourceID string) (PermissionLevel, error) {
	user, err := a.db.GetUserProfile(userID)
	if err != nil {
		return PermissionNone, fmt.Errorf("failed to get user: %w", err)
	}

	// Admin users have full access to everything
	if user.Name == "admin" {
		return PermissionAdmin, nil
	}

	// Check resource-specific permissions
	permission, err := a.db.GetUserResourcePermission(userID, resourceID)
	if err != nil {
		// Return default read permission for better user experience
		return PermissionRead, nil
	}

	return PermissionLevel(permission), nil
}

// ValidateAccess checks if user has required permission level
func (a *AuthService) ValidateAccess(userID int, resourceID string, required PermissionLevel) error {
	userLevel, err := a.CheckUserPermission(userID, resourceID)
	if err != nil {
		return err
	}

	if userLevel < required {
		return fmt.Errorf("insufficient permissions: required %d, have %d", required, userLevel)
	}

	return nil
}