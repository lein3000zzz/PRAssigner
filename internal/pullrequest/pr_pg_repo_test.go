package pullrequest_test

import (
	pullrequest2 "assignerPR/internal/pullrequest"
	"assignerPR/pkg/user"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(PRMatcher()))
	require.NoError(t, err)

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB, DriverName: "postgres"}), &gorm.Config{})
	require.NoError(t, err)

	return gdb, mock, func() { _ = mockDB.Close() }
}

func PRMatcher() sqlmock.QueryMatcher {
	return sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		normalize := func(s string) string { return strings.Join(strings.Fields(s), " ") }
		if strings.HasPrefix(normalize(actual), normalize(expected)) {
			return nil
		}

		log.Println(actual)
		log.Println(expected)

		return sqlmock.ErrCancelled
	})
}

func TestPullRequestsRepoPg_CreatePR(t *testing.T) {
	fixedTime := time.Now()

	type createPRArgs struct {
		prID     string
		prName   string
		authorID string
	}

	tests := []struct {
		name     string
		args     createPRArgs
		mockFunc func(sqlmock.Sqlmock)
		wantErr  error
		wantPR   *pullrequest2.PullRequest
	}{
		{
			name: "success",
			args: createPRArgs{prID: "pr-123", prName: "Fix bug", authorID: "user-123"},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()

				authorRows := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-123", "author", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "users"`).WillReturnRows(authorRows)

				m.ExpectExec(`INSERT INTO "pull_requests"`).
					WithArgs("pr-123", "Fix bug", "user-123", pullrequest2.StatusOpen, sqlmock.AnyArg(), sqlmock.AnyArg(), nil).
					WillReturnResult(sqlmock.NewResult(1, 1))

				candidateRows := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-456", "reviewer1", "backend", true, fixedTime, fixedTime).
					AddRow("user-789", "reviewer2", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT "users"."user_id"`).WillReturnRows(candidateRows)

				m.ExpectExec(`UPDATE "pull_requests" SET "updated_at"`).
					WithArgs(sqlmock.AnyArg(), "pr-123").
					WillReturnResult(sqlmock.NewResult(1, 1))

				m.ExpectExec(`INSERT INTO "users"`).
					WithArgs(
						"user-456", "reviewer1", "backend", true, sqlmock.AnyArg(), sqlmock.AnyArg(),
						"user-789", "reviewer2", "backend", true, sqlmock.AnyArg(), sqlmock.AnyArg(),
					).WillReturnResult(sqlmock.NewResult(2, 2))

				m.ExpectExec(`INSERT INTO "pr_reviewers"`).
					WithArgs("pr-123", "user-456", "pr-123", "user-789").
					WillReturnResult(sqlmock.NewResult(2, 2))

				prRows := sqlmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "updated_at", "merged_at"}).
					AddRow("pr-123", "Fix bug", "user-123", pullrequest2.StatusOpen, fixedTime, fixedTime, nil)
				m.ExpectQuery(`SELECT * FROM "pull_requests"`).WillReturnRows(prRows)

				linkRows := sqlmock.NewRows([]string{"pull_request_id", "user_id"}).
					AddRow("pr-123", "user-456").
					AddRow("pr-123", "user-789")
				m.ExpectQuery(`SELECT * FROM "pr_reviewers"`).WillReturnRows(linkRows)

				assignedRows := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-456", "reviewer1", "backend", true, fixedTime, fixedTime).
					AddRow("user-789", "reviewer2", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "users"`).WillReturnRows(assignedRows)

				m.ExpectCommit()
			},
			wantPR: &pullrequest2.PullRequest{
				PullRequestID:   "pr-123",
				PullRequestName: "Fix bug",
				AuthorID:        "user-123",
				Status:          pullrequest2.StatusOpen,
				AssignedReviewers: []*user.User{
					{UserID: "user-456", Username: "reviewer1", TeamName: "backend", IsActive: true},
					{UserID: "user-789", Username: "reviewer2", TeamName: "backend", IsActive: true},
				},
			},
		},
		{
			name: "author not found",
			args: createPRArgs{prID: "pr-404", prName: "Fix bug", authorID: "unknown"},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(`SELECT * FROM "users"`).WillReturnError(gorm.ErrRecordNotFound)
				m.ExpectRollback()
			},
			wantErr: pullrequest2.ErrPRNotFound,
		},
		{
			name: "pr already exists",
			args: createPRArgs{prID: "pr-123", prName: "Fix bug", authorID: "user-123"},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()

				authorRows := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-123", "author", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "users"`).WillReturnRows(authorRows)

				m.ExpectExec(`INSERT INTO "pull_requests"`).
					WithArgs("pr-123", "Fix bug", "user-123", pullrequest2.StatusOpen, sqlmock.AnyArg(), sqlmock.AnyArg(), nil).
					WillReturnError(errors.New("SQLSTATE 23505"))

				m.ExpectRollback()
			},
			wantErr: pullrequest2.ErrPRExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := newMockDB(t)
			defer cleanup()

			repo := pullrequest2.NewPullRequestsRepoPg(zap.NewNop().Sugar(), db)
			if tt.mockFunc != nil {
				tt.mockFunc(mock)
			}

			got, err := repo.CreatePR(tt.args.prID, tt.args.prName, tt.args.authorID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assertPR(t, got, tt.wantPR)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPullRequestsRepoPg_Merge(t *testing.T) {
	fixedTime := time.Now()

	tests := []struct {
		name     string
		prID     string
		mockFunc func(sqlmock.Sqlmock)
		wantErr  error
		wantPR   *pullrequest2.PullRequest
	}{
		{
			name: "success",
			prID: "pr-123",
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()

				prRows := sqlmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "updated_at", "merged_at"}).
					AddRow("pr-123", "Fix bug", "user-123", pullrequest2.StatusOpen, fixedTime, fixedTime, nil)
				m.ExpectQuery(`SELECT * FROM "pull_requests"`).WillReturnRows(prRows)

				linkRows := sqlmock.NewRows([]string{"pull_request_id", "user_id"}).
					AddRow("pr-123", "user-456").
					AddRow("pr-123", "user-789")
				m.ExpectQuery(`SELECT * FROM "pr_reviewers"`).WillReturnRows(linkRows)

				assignedRows := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-456", "reviewer1", "backend", true, fixedTime, fixedTime).
					AddRow("user-789", "reviewer2", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "users"`).WillReturnRows(assignedRows)

				m.ExpectExec(`UPDATE "pull_requests"`).
					WithArgs(pullrequest2.StatusMerged, sqlmock.AnyArg(), sqlmock.AnyArg(), "pr-123").
					WillReturnResult(sqlmock.NewResult(1, 1))

				m.ExpectCommit()
			},
			wantPR: &pullrequest2.PullRequest{
				PullRequestID: "pr-123",
				Status:        pullrequest2.StatusMerged,
			},
		},
		{
			name: "pr not found",
			prID: "unknown",
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(`SELECT * FROM "pull_requests"`).WillReturnError(gorm.ErrRecordNotFound)
				m.ExpectRollback()
			},
			wantErr: pullrequest2.ErrPRNotFound,
		},
		{
			name: "already merged",
			prID: "pr-123",
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()

				prRows := sqlmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "updated_at", "merged_at"}).
					AddRow("pr-123", "Fix bug", "user-123", pullrequest2.StatusMerged, fixedTime, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "pull_requests"`).WillReturnRows(prRows)

				linkRows := sqlmock.NewRows([]string{"pull_request_id", "user_id"}).
					AddRow("pr-123", "user-456")
				m.ExpectQuery(`SELECT * FROM "pr_reviewers"`).WillReturnRows(linkRows)

				assignedRows := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-456", "reviewer1", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "users"`).WillReturnRows(assignedRows)

				m.ExpectCommit()
			},
			wantPR: &pullrequest2.PullRequest{
				PullRequestID: "pr-123",
				Status:        pullrequest2.StatusMerged,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := newMockDB(t)
			defer cleanup()

			repo := pullrequest2.NewPullRequestsRepoPg(zap.NewNop().Sugar(), db)
			if tt.mockFunc != nil {
				tt.mockFunc(mock)
			}

			got, err := repo.Merge(tt.prID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				if tt.wantPR != nil {
					require.Equal(t, tt.wantPR.PullRequestID, got.PullRequestID)
					require.Equal(t, tt.wantPR.Status, got.Status)
				}
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPullRequestsRepoPg_Reassign(t *testing.T) {
	fixedTime := time.Now()

	type reassignArgs struct {
		prID      string
		oldUserID string
	}

	tests := []struct {
		name         string
		args         reassignArgs
		mockFunc     func(sqlmock.Sqlmock)
		wantErr      error
		wantPR       *pullrequest2.PullRequest
		wantReplaced string
	}{
		{
			name: "success",
			args: reassignArgs{prID: "pr-123", oldUserID: "user-999"},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()

				prRows := sqlmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "updated_at", "merged_at"}).
					AddRow("pr-123", "Fix bug", "user-123", pullrequest2.StatusOpen, fixedTime, fixedTime, nil)
				m.ExpectQuery(`SELECT * FROM "pull_requests"`).WillReturnRows(prRows)

				linkRows := sqlmock.NewRows([]string{"pull_request_id", "user_id"}).
					AddRow("pr-123", "user-111").
					AddRow("pr-123", "user-999")
				m.ExpectQuery(`SELECT * FROM "pr_reviewers"`).WillReturnRows(linkRows)

				assignedRows := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-111", "reviewerA", "backend", true, fixedTime, fixedTime).
					AddRow("user-999", "reviewerB", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "users"`).WillReturnRows(assignedRows)

				candidateRows := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-222", "reviewerC", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT "users"."user_id"`).WillReturnRows(candidateRows)

				m.ExpectExec(`UPDATE "pull_requests" SET "updated_at"`).
					WithArgs(sqlmock.AnyArg(), "pr-123").
					WillReturnResult(sqlmock.NewResult(1, 1))

				m.ExpectExec(`INSERT INTO "users"`).
					WithArgs(
						"user-111", "reviewerA", "backend", true, sqlmock.AnyArg(), sqlmock.AnyArg(),
						"user-222", "reviewerC", "backend", true, sqlmock.AnyArg(), sqlmock.AnyArg(),
					).
					WillReturnResult(sqlmock.NewResult(2, 2))

				m.ExpectExec(`INSERT INTO "pr_reviewers"`).
					WithArgs("pr-123", "user-111", "pr-123", "user-222").
					WillReturnResult(sqlmock.NewResult(2, 2))

				m.ExpectExec(`DELETE FROM "pr_reviewers"`).WillReturnResult(sqlmock.NewResult(2, 2))

				reloadPRRows := sqlmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "updated_at", "merged_at"}).
					AddRow("pr-123", "Fix bug", "user-123", pullrequest2.StatusOpen, fixedTime, fixedTime, nil)
				m.ExpectQuery(`SELECT * FROM "pull_requests"`).WillReturnRows(reloadPRRows)

				reloadLinks := sqlmock.NewRows([]string{"pull_request_id", "user_id"}).
					AddRow("pr-123", "user-111").
					AddRow("pr-123", "user-222")
				m.ExpectQuery(`SELECT * FROM "pr_reviewers"`).WillReturnRows(reloadLinks)

				reloadUsers := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-111", "reviewerA", "backend", true, fixedTime, fixedTime).
					AddRow("user-222", "reviewerC", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "users"`).WillReturnRows(reloadUsers)

				m.ExpectCommit()
			},
			wantPR: &pullrequest2.PullRequest{
				PullRequestID:   "pr-123",
				PullRequestName: "Fix bug",
				AuthorID:        "user-123",
				Status:          pullrequest2.StatusOpen,
				AssignedReviewers: []*user.User{
					{UserID: "user-111", Username: "reviewerA", TeamName: "backend", IsActive: true},
					{UserID: "user-222", Username: "reviewerC", TeamName: "backend", IsActive: true},
				},
			},
			wantReplaced: "user-222",
		},
		{
			name: "pr not found",
			args: reassignArgs{prID: "missing", oldUserID: "user-999"},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(`SELECT * FROM "pull_requests"`).WillReturnError(gorm.ErrRecordNotFound)
				m.ExpectRollback()
			},
			wantErr: pullrequest2.ErrPRNotFound,
		},
		{
			name: "no candidate",
			args: reassignArgs{prID: "pr-123", oldUserID: "user-999"},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()

				prRows := sqlmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "updated_at", "merged_at"}).
					AddRow("pr-123", "Fix bug", "user-123", pullrequest2.StatusOpen, fixedTime, fixedTime, nil)
				m.ExpectQuery(`SELECT * FROM "pull_requests"`).WillReturnRows(prRows)

				linkRows := sqlmock.NewRows([]string{"pull_request_id", "user_id"}).
					AddRow("pr-123", "user-999")
				m.ExpectQuery(`SELECT * FROM "pr_reviewers"`).WillReturnRows(linkRows)

				assignedRows := sqlmock.NewRows([]string{"user_id", "username", "team_name", "is_active", "created_at", "updated_at"}).
					AddRow("user-999", "reviewerB", "backend", true, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "users"`).WillReturnRows(assignedRows)

				m.ExpectQuery(`SELECT "users"."user_id"`).WillReturnError(gorm.ErrRecordNotFound)

				m.ExpectRollback()
			},
			wantErr: pullrequest2.ErrNoCandidate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := newMockDB(t)
			defer cleanup()

			repo := pullrequest2.NewPullRequestsRepoPg(zap.NewNop().Sugar(), db)
			if tt.mockFunc != nil {
				tt.mockFunc(mock)
			}

			got, replaced, err := repo.Reassign(tt.args.prID, tt.args.oldUserID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantReplaced, replaced)
				assertPR(t, got, tt.wantPR)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPullRequestsRepoPg_ListPRsByReviewer(t *testing.T) {
	fixedTime := time.Now()

	tests := []struct {
		name     string
		userID   string
		mockFunc func(sqlmock.Sqlmock)
		wantErr  error
		wantPRs  []*pullrequest2.PullRequest
	}{
		{
			name:   "success",
			userID: "user-456",
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()

				pluckRows := sqlmock.NewRows([]string{"pull_request_id"}).
					AddRow("pr-1").
					AddRow("pr-2")
				m.ExpectQuery(`SELECT "pull_request_id" FROM "pr_reviewers"`).WillReturnRows(pluckRows)

				prRows := sqlmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "updated_at", "merged_at"}).
					AddRow("pr-1", "Fix bug", "user-111", pullrequest2.StatusOpen, fixedTime, fixedTime, nil).
					AddRow("pr-2", "Add feature", "user-222", pullrequest2.StatusMerged, fixedTime, fixedTime, fixedTime)
				m.ExpectQuery(`SELECT * FROM "pull_requests"`).WillReturnRows(prRows)

				m.ExpectCommit()
			},
			wantPRs: []*pullrequest2.PullRequest{
				{PullRequestID: "pr-1", PullRequestName: "Fix bug"},
				{PullRequestID: "pr-2", PullRequestName: "Add feature"},
			},
		},
		{
			name:   "no prs",
			userID: "user-456",
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				pluckRows := sqlmock.NewRows([]string{"pull_request_id"})
				m.ExpectQuery(`SELECT "pull_request_id" FROM "pr_reviewers" WHERE user_id`).WillReturnRows(pluckRows)
				m.ExpectCommit()
			},
			wantPRs: []*pullrequest2.PullRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := newMockDB(t)
			defer cleanup()

			repo := pullrequest2.NewPullRequestsRepoPg(zap.NewNop().Sugar(), db)
			if tt.mockFunc != nil {
				tt.mockFunc(mock)
			}

			got, err := repo.ListPRsByReviewer(tt.userID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, len(tt.wantPRs), len(got))
			for i := range tt.wantPRs {
				require.Equal(t, tt.wantPRs[i].PullRequestID, got[i].PullRequestID)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPullRequestsRepoPg_GetTeamPRStats(t *testing.T) {
	tests := []struct {
		name      string
		teamName  string
		mockFunc  func(sqlmock.Sqlmock)
		wantErr   error
		wantStats []*pullrequest2.UserStats
	}{
		{
			name:     "success",
			teamName: "backend",
			mockFunc: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"user_id", "open_count", "merged_count"}).
					AddRow("user-456", int64(2), int64(1)).
					AddRow("user-789", int64(1), int64(3))
				m.ExpectQuery(`SELECT users.user_id`).WillReturnRows(rows)
			},
			wantStats: []*pullrequest2.UserStats{
				{UserID: "user-456", OpenCount: 2, MergedCount: 1},
				{UserID: "user-789", OpenCount: 1, MergedCount: 3},
			},
		},
		{
			name:     "sql error",
			teamName: "backend",
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT users.user_id`).WillReturnError(errors.New("invalid db"))
			},
			wantErr: errors.New("invalid db"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := newMockDB(t)
			defer cleanup()

			repo := pullrequest2.NewPullRequestsRepoPg(zap.NewNop().Sugar(), db)
			if tt.mockFunc != nil {
				tt.mockFunc(mock)
			}

			got, err := repo.GetTeamPRStats(tt.teamName)

			if tt.wantErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(tt.wantStats), len(got))
				for i := range tt.wantStats {
					require.Equal(t, tt.wantStats[i].UserID, got[i].UserID)
					require.Equal(t, tt.wantStats[i].OpenCount, got[i].OpenCount)
					require.Equal(t, tt.wantStats[i].MergedCount, got[i].MergedCount)
				}
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func assertPR(t *testing.T, got, want *pullrequest2.PullRequest) {
	require.NotNil(t, got)
	require.NotNil(t, want)
	require.Equal(t, want.PullRequestID, got.PullRequestID)
	require.Equal(t, want.PullRequestName, got.PullRequestName)
	require.Equal(t, want.AuthorID, got.AuthorID)
	require.Equal(t, want.Status, got.Status)
	require.Len(t, got.AssignedReviewers, len(want.AssignedReviewers))
	for i := range want.AssignedReviewers {
		require.Equal(t, want.AssignedReviewers[i].UserID, got.AssignedReviewers[i].UserID)
	}
}
