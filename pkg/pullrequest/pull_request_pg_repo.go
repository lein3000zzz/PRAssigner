package pullrequest

import (
	"assignerPR/pkg/user"
	"errors"
	"sort"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PRReviewer struct {
	PullRequestID string `gorm:"column:pull_request_id"`
	UserID        string `gorm:"column:user_id"`
}

type PullRequestsRepoPg struct {
	logger *zap.SugaredLogger
	db     *gorm.DB
}

func NewPullRequestsRepoPg(logger *zap.SugaredLogger, db *gorm.DB) *PullRequestsRepoPg {
	return &PullRequestsRepoPg{
		logger: logger,
		db:     db,
	}
}

func (repo *PullRequestsRepoPg) CreatePR(prID, prName, authorID string) (*PullRequest, error) {
	repo.logger.Debugw("CreatePR()", "prID", prID, "authorID", authorID)

	var pr *PullRequest
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var author user.User
		if err := tx.First(&author, "user_id = ?", authorID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				repo.logger.Warnw("Author does not exist", "prID", prID, "authorID", authorID)
				return ErrPRNotFound
			}
			repo.logger.Errorw("Error finding author", "prID", prID, "authorID", authorID)
			return err
		}

		pr = &PullRequest{
			PullRequestID:   prID,
			PullRequestName: prName,
			AuthorID:        authorID,
			Status:          StatusOpen,
		}

		if err := tx.Create(pr).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				repo.logger.Warnw("PR already exists", "prID", prID, "authorID", authorID)
				return ErrPRExists
			}
			repo.logger.Errorw("Error creating PR", "prID", prID, "authorID", authorID)
			return err
		}

		reviewers, err := repo.pickInitialReviewersInTx(tx, author.TeamName, authorID)
		if err != nil {
			repo.logger.Errorw("Error picking initial reviewers", "prID", prID, "authorID", authorID)
			return err
		}

		if len(reviewers) > 0 {
			if err := tx.Model(pr).Association("AssignedReviewers").Append(reviewers); err != nil {
				repo.logger.Errorw("Error appending reviewers", "prID", prID, "authorID", authorID)
				return err
			}
		}

		return tx.Preload("AssignedReviewers", func(tx2 *gorm.DB) *gorm.DB {
			return tx2.Order("users.user_id ASC")
		}).First(pr, "pull_request_id = ?", prID).Error
	})

	if err != nil {
		repo.logger.Errorw("Error creating PR", "prID", prID, "authorID", authorID)
		return nil, err
	}

	repo.logger.Debugw("PR created", "prID", prID, "authorID", authorID)
	return pr, err
}

func (repo *PullRequestsRepoPg) pickInitialReviewersInTx(tx *gorm.DB, teamName, authorID string) ([]*user.User, error) {
	repo.logger.Debugw("pickInitialReviewersInTx()", "teamName", teamName, "authorID", authorID)

	var reviewers []*user.User
	// Maybe тут можно было как-то покрасивее написать запрос с использованием самих моделей, но я до красивого варианта не дошел.
	err := tx.
		Joins("LEFT JOIN pr_reviewers prr ON prr.user_id = users.user_id").
		Joins("LEFT JOIN pull_requests pr ON pr.pull_request_id = prr.pull_request_id AND pr.status = ?", StatusOpen).
		Where("users.team_name = ? AND users.is_active = TRUE AND users.user_id <> ?", teamName, authorID).
		Group("users.user_id").
		Order("COUNT(prr.user_id) ASC").
		Order("RANDOM()").
		Limit(MaxReviewersPerPR).
		Find(&reviewers).Error

	repo.logger.Debugw("pickInitialReviewersInTx()", "err", err)
	return reviewers, err
}

func (repo *PullRequestsRepoPg) Merge(prID string) (*PullRequest, error) {
	repo.logger.Debugw("Merge()", "prID", prID)

	var pr PullRequest
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := repo.lockAndLoadPR(tx, prID, &pr); err != nil {
			repo.logger.Errorw("Error loading PR", "prID", prID, "err", err)
			return err
		}

		if pr.Status == StatusMerged {
			repo.logger.Warnw("PR already merged", "prID", prID)
			return nil
		}

		now := time.Now().UTC()
		pr.Status = StatusMerged
		pr.MergedAt = &now
		pr.UpdatedAt = now

		if err := tx.Model(&pr).
			Select("status", "merged_at", "updated_at").
			Updates(&pr).Error; err != nil {
			repo.logger.Errorw("Error updating PR merged", "prID", prID)
			return err
		}

		repo.logger.Debugw("PR merged", "prID", prID)
		return nil
	})

	if err != nil {
		repo.logger.Errorw("Error updating PR merged", "prID", prID)
		return nil, err
	}

	repo.logger.Debugw("Merged", "prID", prID)
	return &pr, nil
}

func (repo *PullRequestsRepoPg) Reassign(prID, oldUserID string) (*PullRequest, string, error) {
	repo.logger.Debugw("Reassign()", "prID", prID, "oldUserID", oldUserID)

	if prID == "" || oldUserID == "" {
		repo.logger.Warnw("No PR ID or oldUserID found")
		return nil, "", ErrNotAssigned
	}

	var updatedPR *PullRequest
	var replacedBy string

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var pr PullRequest
		if err := repo.lockAndLoadPR(tx, prID, &pr); err != nil {
			repo.logger.Errorw("Error loading PR", "prID", prID, "err", err)
			return err
		}

		if pr.Status == StatusMerged {
			repo.logger.Warnw("PR already merged", "prID", prID)
			return ErrPRMerged
		}

		oldReviewer, ok := findReviewer(pr.AssignedReviewers, oldUserID)
		if !ok {
			repo.logger.Warnw("no reviewer to reassign", "prID", prID, "oldUserID", oldUserID)
			return ErrNotAssigned
		}

		excludeSet := make(map[string]struct{}, len(pr.AssignedReviewers)+2)
		excludeSet[oldUserID] = struct{}{}
		if pr.AuthorID != "" {
			excludeSet[pr.AuthorID] = struct{}{}
		}
		for _, r := range pr.AssignedReviewers {
			excludeSet[r.UserID] = struct{}{}
		}
		exclude := make([]string, 0, len(excludeSet))
		for id := range excludeSet {
			exclude = append(exclude, id)
		}

		var candidate user.User
		query := tx.Model(&user.User{}).
			Joins("LEFT JOIN pr_reviewers prr ON prr.user_id = users.user_id").
			Joins("LEFT JOIN pull_requests pr ON pr.pull_request_id = prr.pull_request_id AND pr.status = ?", StatusOpen).
			Where("users.team_name = ? AND users.is_active = TRUE", oldReviewer.TeamName).
			Group("users.user_id")

		if len(exclude) > 0 {
			query = query.Where("users.user_id NOT IN ?", exclude)
		}

		if err := query.
			Order("COUNT(prr.user_id) ASC").
			Order("RANDOM()").
			Limit(1).
			First(&candidate).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				repo.logger.Errorw("no candidates for reassign", "prID", prID, "oldUserID", oldUserID)
				return ErrNoCandidate
			}
			repo.logger.Errorw("error reassigning PR", "prID", prID, "err", err)
			return err
		}

		newReviewers := make([]*user.User, 0, len(pr.AssignedReviewers))
		for _, r := range pr.AssignedReviewers {
			if r.UserID != oldReviewer.UserID {
				newReviewers = append(newReviewers, r)
			}
		}
		newReviewers = append(newReviewers, &candidate)

		if err := tx.Model(&pr).Association("AssignedReviewers").Replace(newReviewers); err != nil {
			repo.logger.Errorw("error reassigning PR", "prID", prID, "err", err)
			return err
		}

		if err := repo.reloadPR(tx, prID, &pr); err != nil {
			repo.logger.Errorw("error reassigning PR", "prID", prID, "err", err)
			return err
		}

		updatedPR = &pr
		replacedBy = candidate.UserID
		repo.logger.Debugw("Reassigned PR", "prID", prID)

		return nil
	})

	if err != nil {
		repo.logger.Errorw("Error reassigning PR", "prID", prID)
		return nil, "", err
	}

	repo.logger.Debugw("Reassigned PR", "prID", prID, "oldUserID", oldUserID, "replacedBy", replacedBy)
	return updatedPR, replacedBy, nil
}

func (repo *PullRequestsRepoPg) lockAndLoadPR(tx *gorm.DB, prID string, pr *PullRequest) error {
	repo.logger.Debugw("lockAndLoadPR()", "prID", prID)

	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("AssignedReviewers", func(tx2 *gorm.DB) *gorm.DB {
			return tx2.Order("users.user_id ASC")
		}).
		First(pr, "pull_request_id = ?", prID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			repo.logger.Warnw("PR does not exist", "prID", prID)
			return ErrPRNotFound
		}

		repo.logger.Errorw("Error finding PR", "prID", prID)
		return err
	}

	repo.logger.Debugw("PR found", "prID", prID)
	return nil
}

func (repo *PullRequestsRepoPg) reloadPR(tx *gorm.DB, prID string, pr *PullRequest) error {
	repo.logger.Debugw("reloadPR()", "prID", prID)

	return tx.
		Preload("AssignedReviewers", func(tx2 *gorm.DB) *gorm.DB {
			return tx2.Order("users.user_id ASC")
		}).
		First(pr, "pull_request_id = ?", prID).Error
}

func (repo *PullRequestsRepoPg) ListPRsByReviewer(userID string) ([]*PullRequest, error) {
	repo.logger.Debugw("ListPRsByReviewer()", "userID", userID)

	if userID == "" {
		repo.logger.Warnw("userID is empty", "userID", userID)
		return []*PullRequest{}, nil
	}

	var prShort []*PullRequest

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var ids []string
		if err := tx.Model(&PRReviewer{}).
			Where("user_id = ?", userID).
			Pluck("pull_request_id", &ids).Error; err != nil {
			repo.logger.Errorw("error plucking pr ids", "userID", userID, "err", err)
			return err
		}
		if len(ids) == 0 {
			prShort = []*PullRequest{}
			repo.logger.Warnw("no PR reviewer", "userID", userID)
			return nil
		}

		order := make(map[string]int, len(ids))
		for i, id := range ids {
			order[id] = i
		}
		fallback := len(ids)

		var rows []*PullRequest
		if err := tx.Model(&PullRequest{}).
			Where("pull_request_id IN ?", ids).
			Find(&rows).Error; err != nil {
			repo.logger.Errorw("error loading prs", "userID", userID, "err", err)
			return err
		}

		sort.SliceStable(rows, func(i, j int) bool {
			oi := orderIndex(order, rows[i].PullRequestID, fallback)
			oj := orderIndex(order, rows[j].PullRequestID, fallback)
			return oi < oj
		})

		prShort = rows
		return nil
	})

	if err != nil {
		repo.logger.Errorw("error loading prs", "userID", userID, "err", err)
		return nil, err
	}

	repo.logger.Debugw("listed PRs by reviewer")
	return prShort, nil
}
