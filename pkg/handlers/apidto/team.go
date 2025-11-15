package apidto

import (
	"assignerPR/pkg/pullrequest"
	"assignerPR/pkg/team"
	"assignerPR/pkg/user"
)

type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

func FromTeam(t *team.Team) Team {
	if t == nil {
		return Team{}
	}
	members := make([]TeamMember, 0, len(t.Members))
	for _, m := range t.Members {
		members = append(members, TeamMember{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	return Team{
		TeamName: t.TeamName,
		Members:  members,
	}
}

func ToTeam(dto Team) *team.Team {
	users := make([]*user.User, 0, len(dto.Members))
	for _, m := range dto.Members {
		users = append(users, &user.User{
			UserID:   m.UserID,
			Username: m.Username,
			TeamName: dto.TeamName,
			IsActive: m.IsActive,
		})
	}
	return &team.Team{
		TeamName: dto.TeamName,
		Members:  users,
	}
}

type UserStats struct {
	OpenCount   int `json:"open_count"`
	MergedCount int `json:"merged_count"`
}

func FromUserStatsSliceToMap(users []*pullrequest.UserStats) map[string]UserStats {
	memberStats := make(map[string]UserStats, len(users))
	for _, usr := range users {
		memberStats[usr.UserID] = FromUserStats(usr)
	}
	return memberStats
}

func FromUserStats(us *pullrequest.UserStats) UserStats {
	return UserStats{
		OpenCount:   us.OpenCount,
		MergedCount: us.MergedCount,
	}
}

func ToUserStats(dto UserStats, userID string) *pullrequest.UserStats {
	return &pullrequest.UserStats{
		UserID:      userID,
		OpenCount:   dto.OpenCount,
		MergedCount: dto.MergedCount,
	}
}
