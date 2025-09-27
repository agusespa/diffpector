package utils

import (
	"fmt"
)

type User struct {
	ID   string
	Name string
}

func (u *User) Greet() {
	fmt.Printf("Hello, my name is %s\n", u.Name)
}

