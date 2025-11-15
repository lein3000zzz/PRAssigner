package team

import (
	"assignerPR/pkg/user"
	"errors"
	"time"
)

var (
	ErrTeamExists = errors.New("TEAM_EXISTS")
)

type Team struct {
	TeamName  string `gorm:"primaryKey;type:varchar(64);column:team_name"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Members []*user.User `gorm:"foreignKey:TeamName;references:TeamName;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

type TeamsRepo interface {
	CreateTeam(teamName string, members []*user.User) (*Team, error)
	GetTeam(teamName string) (*Team, error)
}
