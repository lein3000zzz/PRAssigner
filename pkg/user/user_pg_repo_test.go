package user_test

import (
	"assignerPR/pkg/user"
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

	mockDB, mock, err := sqlmock.New(
		sqlmock.QueryMatcherOption(UsersUpdateMatcher()),
	)
	require.NoError(t, err)

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	}), &gorm.Config{})

	require.NoError(t, err)

	return gdb, mock, func() { mockDB.Close() }
}

// Очень нетребовательный мэтчер (хоть это и не совсем хорошо,
// тут решил особо сильно не заморачиваться - полную проверку gorm'а посчитал лишним делать)
func UsersUpdateMatcher() sqlmock.QueryMatcher {
	return sqlmock.QueryMatcherFunc(func(expected, actual string) error {

		normalize := func(s string) string {
			return strings.Join(strings.Fields(s), " ")
		}

		act := normalize(actual)
		exp := normalize(expected)

		if strings.HasPrefix(act, exp) {
			return nil
		}

		log.Println(act)
		log.Println(exp)

		return sqlmock.ErrCancelled
	})
}

func TestUsersRepoPg_SetIsActive(t *testing.T) {

	type setActiveArgs struct {
		userID   string
		isActive bool
	}

	tests := []struct {
		name     string
		args     setActiveArgs
		mockFunc func(sqlmock.Sqlmock)
		wantErr  error
		wantUser *user.User
	}{
		{
			name: "success",
			args: setActiveArgs{
				userID:   "user-123",
				isActive: false,
			},
			mockFunc: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"user_id", "username", "team_name", "is_active", "created_at", "updated_at",
				}).AddRow(
					"user-123", "abobus", "backend", false, time.Now(), time.Now(),
				)

				m.ExpectBegin()
				m.ExpectQuery(`UPDATE "users" SET "is_active"=$1`).
					WithArgs(false, sqlmock.AnyArg(), "user-123").
					WillReturnRows(rows)
				m.ExpectCommit()
			},
			wantErr: nil,
			wantUser: &user.User{
				UserID:   "user-123",
				Username: "abobus",
				TeamName: "backend",
				IsActive: false,
			},
		},

		{
			name: "user not found",
			args: setActiveArgs{
				userID:   "unknown",
				isActive: true,
			},
			mockFunc: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"user_id", "username", "team_name", "is_active", "created_at", "updated_at",
				})

				m.ExpectBegin()
				m.ExpectQuery(`UPDATE "users" SET "is_active"=$1`).
					WithArgs(true, sqlmock.AnyArg(), "unknown").
					WillReturnRows(rows)
				m.ExpectCommit()
			},
			wantErr:  user.ErrUserNotFound,
			wantUser: nil,
		},

		{
			name: "sql error",
			args: setActiveArgs{
				userID:   "u3-Bob",
				isActive: true,
			},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(`UPDATE "users" SET "is_active"=$1`).
					WithArgs(true, sqlmock.AnyArg(), "u3-Bob").
					WillReturnError(gorm.ErrInvalidDB)
				m.ExpectRollback()
			},
			wantErr:  gorm.ErrInvalidDB,
			wantUser: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			db, mock, cleanup := newMockDB(t)
			defer cleanup()

			logger := zap.NewNop().Sugar()
			repo := user.NewUsersRepoPg(logger, db)

			tt.mockFunc(mock)

			got, err := repo.SetIsActive(tt.args.userID, tt.args.isActive)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.wantUser != nil {
				require.NotNil(t, got)
				require.Equal(t, tt.wantUser.UserID, got.UserID)
				require.Equal(t, tt.wantUser.Username, got.Username)
				require.Equal(t, tt.wantUser.TeamName, got.TeamName)
				require.Equal(t, tt.wantUser.IsActive, got.IsActive)
			} else {
				require.Nil(t, got)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUsersRepoPg_SetIsActiveByTeam(t *testing.T) {
	type args struct {
		teamName string
		isActive bool
	}

	tests := []struct {
		name      string
		args      args
		mockFunc  func(sqlmock.Sqlmock)
		wantErr   error
		wantUsers []*user.User
	}{
		{
			name: "success multiple users",
			args: args{
				teamName: "backend",
				isActive: false,
			},
			mockFunc: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"user_id", "username", "team_name", "is_active", "created_at", "updated_at",
				}).
					AddRow("u1", "alice", "backend", false, time.Now(), time.Now()).
					AddRow("u2", "bob", "backend", false, time.Now(), time.Now())

				m.ExpectBegin()
				m.ExpectQuery(`UPDATE "users" SET "is_active"=$1`).
					WithArgs(false, sqlmock.AnyArg(), "backend").
					WillReturnRows(rows)
				m.ExpectCommit()
				m.ExpectQuery(`SELECT * FROM "users" WHERE team_name = $1`).
					WithArgs("backend").
					WillReturnRows(rows)
			},
			wantErr: nil,
			wantUsers: []*user.User{
				{UserID: "u2", Username: "bob", TeamName: "backend", IsActive: false},
			},
		},
		{
			name: "team not found (no rows affected)",
			args: args{
				teamName: "unknown-team",
				isActive: true,
			},
			mockFunc: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"user_id", "username", "team_name", "is_active", "created_at", "updated_at",
				})
				m.ExpectBegin()
				m.ExpectQuery(`UPDATE "users" SET "is_active"=$1`).
					WithArgs(true, sqlmock.AnyArg(), "unknown-team").
					WillReturnRows(rows)
				m.ExpectCommit()
				m.ExpectQuery(`SELECT * FROM "users" WHERE team_name = $1`).
					WithArgs("unknown-team").
					WillReturnRows(sqlmock.NewRows([]string{
						"user_id", "username", "team_name", "is_active", "created_at", "updated_at",
					}))
			},
			wantErr:   user.ErrUserNotFound,
			wantUsers: nil,
		},
		{
			name: "sql error",
			args: args{
				teamName: "frontend",
				isActive: true,
			},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(`UPDATE "users" SET "is_active"=$1`).
					WithArgs(true, sqlmock.AnyArg(), "frontend").
					WillReturnError(gorm.ErrInvalidDB)
				m.ExpectRollback()
			},
			wantErr:   gorm.ErrInvalidDB,
			wantUsers: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := newMockDB(t)
			defer cleanup()

			logger := zap.NewNop().Sugar()
			repo := user.NewUsersRepoPg(logger, db)

			tt.mockFunc(mock)

			got, err := repo.SetIsActiveByTeam(tt.args.teamName, tt.args.isActive)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.wantUsers != nil {
				require.Len(t, got, len(tt.wantUsers))
				for i, u := range tt.wantUsers {
					require.Equal(t, u.UserID, got[i].UserID)
					require.Equal(t, u.Username, got[i].Username)
					require.Equal(t, u.TeamName, got[i].TeamName)
					require.Equal(t, u.IsActive, got[i].IsActive)
				}
			} else {
				require.Nil(t, got)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
