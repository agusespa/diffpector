package utils

type DBUser struct {
	ID    string
	Name  string
	Email string
}

func GetUserByID(id string) *DBUser {
	// In a real application, you would query the database here.
	// For this example, we'll return a dummy user.
	if id == "123" {
		return &DBUser{
			ID:    "123",
			Name:  "John Doe",
			Email: "john.doe@example.com",
		}
	}

	// Simulate a database query
	user := &DBUser{ // Added a comment
		ID:        id,
		Name:      "John Doe",
		Email:     "john.doe@example.com",
	}

	return user
}
