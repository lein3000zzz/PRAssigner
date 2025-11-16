package user

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PRReviewer struct {
	PullRequestID string `gorm:"column:pull_request_id"`
	UserID        string `gorm:"column:user_id"`
}

type UsersRepoPg struct {
	logger *zap.SugaredLogger
	db     *gorm.DB
}

func NewUsersRepoPg(logger *zap.SugaredLogger, db *gorm.DB) *UsersRepoPg {
	return &UsersRepoPg{
		logger: logger,
		db:     db,
	}
}

func (repo *UsersRepoPg) SetIsActive(userID string, isActive bool) (*User, error) {
	repo.logger.Debugw("setIsActive()", "userID", userID, "isActive", isActive)

	var user User
	tx := repo.db.
		Model(&user).
		Where("user_id = ?", userID).
		Clauses(clause.Returning{}).
		Update("is_active", isActive)

	if tx.Error != nil {
		repo.logger.Errorw("error setting is_active", "userID", userID, "err", tx.Error)
		return nil, tx.Error
	}

	if tx.RowsAffected == 0 {
		repo.logger.Errorw("error setting is_active - no user found with this id", "userID", userID)
		return nil, ErrUserNotFound
	}

	repo.logger.Debugw("got user by id", "userID", userID)
	return &user, nil
}

func (repo *UsersRepoPg) SetIsActiveByTeam(teamName string, isActive bool) ([]*User, error) {
	repo.logger.Debugw("SetIsActiveByTeam()", "teamName", teamName)

	var updatedUsers []*User
	tx := repo.db.
		Model(&User{}).
		Where("team_name = ?", teamName).
		Clauses(clause.Returning{}).
		Update("is_active", isActive).
		Scan(&updatedUsers)
	if tx.Error != nil {
		repo.logger.Errorw("failed to deactivate team members", "teamName", teamName, "err", tx.Error)
		return nil, tx.Error
	}

	if tx.RowsAffected == 0 {
		repo.logger.Warnw("no active members found to deactivate", "teamName", teamName)
		return nil, ErrUserNotFound
	}

	repo.logger.Debugw("team members deactivated", "teamName", teamName, "affected", tx.RowsAffected)
	return updatedUsers, nil
}
