package pullrequest

import (
	"assignerPR/pkg/user"
	"errors"
	"time"
)

const (
	StatusOpen   = "OPEN"
	StatusMerged = "MERGED"
)

// MaxReviewersPerPR Можно бы было вынести в .env для легкой модификации, но тогда вообще уже лучше
// раскатывать удаленный сервис для хранения секретов / конфигов по типу vault для автономности таких изменений
// и мгновенного вступления в силу.
// В текущей ситуации считаю достаточным оставить константой
const MaxReviewersPerPR = 2

var (
	ErrPRExists    = errors.New("PR_EXISTS")
	ErrPRMerged    = errors.New("PR_MERGED")
	ErrPRNotFound  = errors.New("PR_NOT_FOUND")
	ErrNotAssigned = errors.New("PR_NOT_ASSIGNED")
	ErrNoCandidate = errors.New("PR_NO_CANDIDATE")
)

type PullRequest struct {
	PullRequestID     string       `gorm:"primaryKey;type:varchar(64);column:pull_request_id"`
	PullRequestName   string       `gorm:"type:varchar(255);not null;column:pull_request_name"`
	AuthorID          string       `gorm:"type:varchar(64);index;not null;column:author_id"`
	Status            string       `gorm:"type:pull_request_status;not null;default:OPEN;index"`
	AssignedReviewers []*user.User `gorm:"many2many:pr_reviewers;joinForeignKey:PullRequestID;joinReferences:UserID"`
	CreatedAt         time.Time    `gorm:"column:created_at"`
	UpdatedAt         time.Time    `gorm:"column:updated_at"`
	MergedAt          *time.Time   `gorm:"column:merged_at"`
}

// UserStats - статистика для юзера, относится к дополнительному заданию - сделал статистику PR для членов команды
type UserStats struct {
	UserID      string
	OpenCount   int
	MergedCount int
}

type PullRequestsRepo interface {
	CreatePR(prID, prName, authorID string) (*PullRequest, error)
	Merge(prID string) (*PullRequest, error)
	Reassign(prID, oldUserID string) (pr *PullRequest, replacedBy string, err error)
	ListPRsByReviewer(userID string) ([]*PullRequest, error)
	GetTeamPRStats(teamName string) ([]*UserStats, error)
}
