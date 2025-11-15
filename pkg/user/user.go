package user

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound = errors.New("USER_NOT_FOUND")
)

type User struct {
	UserID    string `gorm:"primaryKey;type:varchar(64);column:user_id" json:"user_id"`
	Username  string `gorm:"type:varchar(255);not null;column:username" json:"username"`
	TeamName  string `gorm:"type:varchar(64);index;not null;column:team_name" json:"team_name"`
	IsActive  bool   `gorm:"not null;default:true;column:is_active" json:"is_active"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UsersRepo interface {
	SetIsActive(userID string, isActive bool) (*User, error)
}
