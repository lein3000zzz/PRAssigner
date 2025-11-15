package apidto

import (
	"assignerPR/pkg/pullrequest"
	"time"
)

type PullRequest struct {
	PullRequestID     string     `json:"pull_request_id"`
	PullRequestName   string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            string     `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         *time.Time `json:"createdAt,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

type PRShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

func FromPR(pr *pullrequest.PullRequest) PullRequest {
	if pr == nil {
		return PullRequest{}
	}

	reviewerIDs := make([]string, 0, len(pr.AssignedReviewers))
	for _, r := range pr.AssignedReviewers {
		if r != nil && r.UserID != "" {
			reviewerIDs = append(reviewerIDs, r.UserID)
		}
	}

	var createdAtPtr *time.Time
	if !pr.CreatedAt.IsZero() {
		t := pr.CreatedAt
		createdAtPtr = &t
	}

	return PullRequest{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: reviewerIDs,
		CreatedAt:         createdAtPtr,
		MergedAt:          pr.MergedAt,
	}
}

func FromPRs(prs []*pullrequest.PullRequest) []PullRequest {
	out := make([]PullRequest, 0, len(prs))
	for _, pr := range prs {
		out = append(out, FromPR(pr))
	}
	return out
}

func FromPRToShort(pr *pullrequest.PullRequest) PRShort {
	if pr == nil {
		return PRShort{}
	}
	return PRShort{
		PullRequestID:   pr.PullRequestID,
		PullRequestName: pr.PullRequestName,
		AuthorID:        pr.AuthorID,
		Status:          pr.Status,
	}
}

func FromPRsToShort(prs []*pullrequest.PullRequest) []PRShort {
	out := make([]PRShort, 0, len(prs))
	for _, pr := range prs {
		out = append(out, FromPRToShort(pr))
	}
	return out
}
