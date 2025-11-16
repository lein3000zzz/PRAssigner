package pullrequest

import "assignerPR/pkg/user"

// можно было бы для корректного логирования сделать это методами репозитория, но я решил сделать их здесь

func findReviewer(reviewers []*user.User, userID string) (*user.User, bool) {
	for _, reviewer := range reviewers {
		if reviewer.UserID == userID {
			return reviewer, true
		}
	}
	return nil, false
}

func orderIndex(order map[string]int, id string, fallback int) int {
	if idx, ok := order[id]; ok {
		return idx
	}
	return fallback
}
