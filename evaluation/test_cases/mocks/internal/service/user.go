//go:build ignore
// +build ignore

package service

import (
	"database/sql"
)

type UserService struct {
	db *sql.DB
}

type UserWithProfile struct {
	ID      int
	Name    string
	Email   string
	Profile *Profile
}

type Profile struct {
	Bio     string
	Avatar  string
	Website string
}

func (s *UserService) GetUsersWithProfiles(userIDs []int) ([]*UserWithProfile, error) {
	users, err := s.GetUsersByIDs(userIDs)
	if err != nil {
		return nil, err
	}

	var result []*UserWithProfile
	for _, user := range users {
		profile, err := s.GetUserProfile(user.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, &UserWithProfile{
			ID:      user.ID,
			Name:    user.Name,
			Email:   user.Email,
			Profile: profile,
		})
	}
	return result, nil
}

func (s *UserService) GetUsersByIDs(userIDs []int) ([]*User, error) {
	// Implementation
	return nil, nil
}

func (s *UserService) GetUserProfile(userID int) (*Profile, error) {
	// Implementation
	return nil, nil
}

type User struct {
	ID    int
	Name  string
	Email string
}