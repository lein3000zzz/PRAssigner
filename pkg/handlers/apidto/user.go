package apidto

import "assignerPR/pkg/user"

type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

func FromUser(u *user.User) User {
	if u == nil {
		return User{}
	}
	return User{
		UserID:   u.UserID,
		Username: u.Username,
		TeamName: u.TeamName,
		IsActive: u.IsActive,
	}
}

func FromUsers(users []*user.User) []User {
	out := make([]User, 0, len(users))
	for _, u := range users {
		out = append(out, FromUser(u))
	}
	return out
}

func ToUser(dto User) *user.User {
	return &user.User{
		UserID:   dto.UserID,
		Username: dto.Username,
		TeamName: dto.TeamName,
		IsActive: dto.IsActive,
	}
}

func ToUsers(dtos []User) []*user.User {
	out := make([]*user.User, 0, len(dtos))
	for _, dto := range dtos {
		out = append(out, ToUser(dto))
	}
	return out
}
