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

// UsersRepo - вообще можно было бы оставить только SetIsActive, но все остальное было сделано под предлогом зависимости от самой сущности
// user в других репах, поэтому я решил и сюда это добавить для полноты реализации и упрощения генерации данных для тестов.
// Почему не оставил методы здесь и не импортировал юзеррепу с этими методами туда?
// Ответ: там требуется несколько действий, которые для консистентности должны быть совершены в рамках одной транзакции, что потребовало
// бы передавать *gorm.DB. Добавлять *gorm.DB в интерфейс мне показалось преступлением, поэтому я сделал фактически дубликат кода
// (хотя я его сделал только для себя, поскольку он нигде не используется (если успею по дедлайнам, то TODO)).
// RUG, получается: Repeat Until Good
type UsersRepo interface {
	SetIsActive(userID string, isActive bool) (*User, error)
	GetByID(userID string) (*User, error)
	UpsertUsers(teamName string, users []*User) error
	ListActiveByTeamExcept(teamName string, excludeIDs []string, limit int) ([]*User, error)
	ListByTeam(teamName string) ([]*User, error)
	ListReviewPRIDs(userID string) ([]string, error)
}
