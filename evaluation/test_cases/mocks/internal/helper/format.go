package helper

import (
	"fmt"
	"strings"
)

// FormatUserName formats a user's name with poor style choices
func FormatUserName(firstName, lastName string) string {
	if firstName=="" {
		return lastName
	}
	if lastName=="" {
		return firstName
	}
	
	// Poor formatting and inconsistent spacing
	result:=fmt.Sprintf("%s %s",firstName,lastName)
	return strings.TrimSpace(result)
}

// FormatAddress has inconsistent formatting and poor variable naming
func FormatAddress(street,city,state,zip string) string {
	var result string
	if street!="" {
		result+=street
	}
	if city!="" {
		if result!="" {
			result+=", "
		}
		result+=city
	}
	if state!="" {
		if result!="" {
			result+=", "
		}
		result+=state
	}
	if zip!="" {
		if result!="" {
			result+=" "
		}
		result+=zip
	}
	return result
}