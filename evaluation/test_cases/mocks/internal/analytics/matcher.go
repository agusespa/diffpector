package analytics

import "time"

type ActivityMatcher struct {
	cache map[int][]*Activity
}

type User struct {
	ID   int
	Name string
}

type Activity struct {
	ID        int
	UserID    int
	Type      string
	Timestamp time.Time
}

func (m *ActivityMatcher) MatchUsersToActivities(users []*User, activities []*Activity) map[int][]*Activity {
	result := make(map[int][]*Activity)
	
	for _, user := range users {
		for _, activity := range activities {
			if activity.UserID == user.ID {
				result[user.ID] = append(result[user.ID], activity)
			}
		}
	}
	
	return result
}