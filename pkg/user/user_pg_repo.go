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

func (repo *UsersRepoPg) GetByID(userID string) (*User, error) {
	repo.logger.Debugw("getByID()", "userID", userID)

	var user User
	if err := repo.db.First(&user, "user_id = ?", userID).Error; err != nil {
		repo.logger.Errorw("error getting user by id", "userID", userID)
		return nil, err
	}

	repo.logger.Debugw("got user by id", "userID", userID)
	return &user, nil
}

func (repo *UsersRepoPg) UpsertUsers(teamName string, users []*User) error {
	repo.logger.Debugw("upsertUsers()", "teamName", teamName)

	if len(users) == 0 {
		repo.logger.Debugw("no users to update", "teamName", teamName)
		return nil
	}

	usersCopy := make([]*User, len(users))
	for i, user := range users {
		newU := *user
		newU.TeamName = teamName
		usersCopy[i] = &newU
	}

	repo.logger.Debugw("upserting users", "teamName", teamName, "users", usersCopy)
	return repo.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"username", "team_name", "is_active"}),
	}).Create(&usersCopy).Error
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

func (repo *UsersRepoPg) ListActiveByTeamExcept(teamName string, excludeIDs []string, limit int) ([]*User, error) {
	repo.logger.Debugw("listActiveByTeamExcept()", "teamName", teamName)

	var users []*User
	query := repo.db.Where("team_name = ? AND is_active = true", teamName)

	if len(excludeIDs) > 0 {
		query = query.Where("user_id NOT IN ?", excludeIDs)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Order("user_id").Find(&users).Error; err != nil {
		repo.logger.Errorw("error listing active users", "error", err)
		return nil, err
	}

	repo.logger.Debugw("got active users", "users", users)
	return users, nil
}

func (repo *UsersRepoPg) ListByTeam(teamName string) ([]*User, error) {
	repo.logger.Debugw("listByTeam()", "teamName", teamName)

	var users []*User
	if err := repo.db.
		Where("team_name = ?", teamName).
		Order("user_id").
		Find(&users).Error; err != nil {
		repo.logger.Errorw("error listing users", "error", err)
		return nil, err
	}

	repo.logger.Debugw("got users by team", "teamName", teamName, "users", users)
	return users, nil
}

func (repo *UsersRepoPg) ListReviewPRIDs(userID string) ([]string, error) {
	repo.logger.Debugw("listReviewPRIDs()", "userID", userID)

	var ids []string
	err := repo.db.
		Model(&PRReviewer{}).
		Where("user_id = ?", userID).
		Pluck("pull_request_id", &ids).Error
	if err != nil {
		repo.logger.Errorw("error listing PR reviewers", "error", err)
		return nil, err
	}

	repo.logger.Debugw("got PR reviewers", "userID", userID, "pull_requestIds", ids)
	return ids, nil
}
