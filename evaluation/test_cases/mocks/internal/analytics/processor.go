//go:build ignore

package analytics

import (
	"fmt"
	"time"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Activity struct {
	ID     int       `json:"id"`
	UserID int       `json:"user_id"`
	Action string    `json:"action"`
	Time   time.Time `json:"time"`
}

type UserReport struct {
	User       *User       `json:"user"`
	Activities []*Activity `json:"activities"`
	Generated  time.Time   `json:"generated"`
}

type UserService interface {
	GetUsersBatch(userIDs []int) ([]*User, error)
	GetUser(userID int) (*User, error)
}

type ActivityService interface {
	GetActivitiesBatch(userIDs []int) (map[int][]*Activity, error)
	GetUserActivities(userID int) ([]*Activity, error)
}

type Processor struct {
	userService     UserService
	activityService ActivityService
}

func NewProcessor(userService UserService, activityService ActivityService) *Processor {
	return &Processor{
		userService:     userService,
		activityService: activityService,
	}
}

// GenerateUserReports processes user reports
func (p *Processor) GenerateUserReports(userIDs []int) ([]*UserReport, error) {
	if len(userIDs) == 0 {
		return nil, fmt.Errorf("no users provided")
	}

	users, err := p.userService.GetUsersBatch(userIDs)
	if err != nil {
		return nil, err
	}

	activities, err := p.activityService.GetActivitiesBatch(userIDs)
	if err != nil {
		return nil, err
	}

	reports := make([]*UserReport, len(users))
	for i, user := range users {
		reports[i] = &UserReport{
			User:       user,
			Activities: activities[user.ID],
			Generated:  time.Now(),
		}
	}

	return reports, nil
}
