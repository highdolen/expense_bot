package domain

import "time"

type User struct {
	ID        int64
	Username  string
	FirstName string
	Timezone  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Location возвращает часовой пояс пользователя.
// Если он не задан или некорректен, используется часовой пояс по умолчанию.
func (u User) Location(defaultLocation *time.Location) *time.Location {
	if u.Timezone == "" {
		return defaultLocation
	}

	location, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return defaultLocation
	}

	return location
}
