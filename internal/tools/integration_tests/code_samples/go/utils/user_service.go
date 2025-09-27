
package utils

// UserService provides user-related services.
type UserService struct {
	userRepo *UserRepository
}

// NewUserService creates a new UserService.
func NewUserService(userRepo *UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// GetUser gets a user by their ID.
func (s *UserService) GetUser(id string) (*User, error) {
	return s.userRepo.GetUserByID(id)
}
