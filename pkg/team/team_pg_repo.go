package team

import (
	"assignerPR/pkg/user"
	"errors"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TeamsRepoPg struct {
	logger *zap.SugaredLogger
	db     *gorm.DB
}

func NewTeamsRepoPg(logger *zap.SugaredLogger, db *gorm.DB) *TeamsRepoPg {
	return &TeamsRepoPg{
		logger: logger,
		db:     db,
	}
}

func (repo *TeamsRepoPg) CreateTeam(teamName string, members []*user.User) (*Team, error) {
	repo.logger.Debugw("CreateTeam()", "teamName", teamName, "membersCount", len(members))

	var team Team
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		team = Team{
			TeamName: teamName,
		}
		if err := tx.Create(&team).Error; err != nil {
			// На сложных операциях, (как пример, транзакция), gorm не всегда отлавливает и оборачивает ошибки,
			// возвращая просто ошибку бд, которую приходится проверять вручную
			if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "SQLSTATE 23505") {
				repo.logger.Warnw("couldnt create team - already exists", "teamName", teamName, "membersCount", len(members))
				return ErrTeamExists
			}
			repo.logger.Errorw("error creating team", "error", err)
			return err
		}

		if len(members) > 0 {
			usersCopy := make([]*user.User, len(members))
			for i, m := range members {
				copyU := *m
				copyU.TeamName = teamName
				usersCopy[i] = &copyU
			}

			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"username", "team_name", "is_active"}),
			}).Create(&usersCopy).Error; err != nil {
				repo.logger.Errorw("error creating team", "error", err)
				return err
			}
		}

		repo.logger.Debugw("team created", "teamName", teamName, "membersCount", len(members))
		return tx.Preload("Members").First(&team, "team_name = ?", teamName).Error
	})

	if err != nil {
		repo.logger.Errorw("failed to create team", "teamName", teamName, "err", err)
		return nil, err
	}

	repo.logger.Debugw("created team", "teamName", teamName, "membersCount", len(members))
	return &team, nil
}

func (repo *TeamsRepoPg) GetTeam(teamName string) (*Team, error) {
	repo.logger.Debugw("GetTeam()", "teamName", teamName)

	var team Team
	if err := repo.db.
		Preload("Members").
		First(&team, "team_name = ?", teamName).Error; err != nil {
		repo.logger.Errorw("failed to query team", "teamName", teamName, "err", err)
		return nil, err
	}

	repo.logger.Debugw("Team found", "teamName", teamName)
	return &team, nil
}
