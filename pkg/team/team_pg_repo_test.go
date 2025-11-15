package team_test

import (
	"assignerPR/pkg/team"
	"assignerPR/pkg/user"
	"errors"
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
		sqlmock.QueryMatcherOption(TeamsCreateMatcher()),
	)
	require.NoError(t, err)

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	}), &gorm.Config{})

	require.NoError(t, err)

	return gdb, mock, func() { mockDB.Close() }
}

func TeamsCreateMatcher() sqlmock.QueryMatcher {
	return sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		normalize := func(s string) string {
			return strings.Join(strings.Fields(s), " ")
		}

		act := normalize(actual)
		exp := normalize(expected)

		if strings.HasPrefix(act, exp) {
			return nil
		}

		return sqlmock.ErrCancelled
	})
}

func TestTeamsRepoPg_CreateTeam(t *testing.T) {
	type createTeamArgs struct {
		teamName string
		members  []*user.User
	}

	tests := []struct {
		name     string
		args     createTeamArgs
		mockFunc func(sqlmock.Sqlmock)
		wantErr  error
		wantTeam *team.Team
	}{
		{
			name: "success",
			args: createTeamArgs{
				teamName: "backend",
				members: []*user.User{
					{
						UserID:   "user-123",
						Username: "abobus",
						TeamName: "backend",
						IsActive: true,
					},
				},
			},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectExec(`INSERT INTO "teams"`).
					WithArgs("backend", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				m.ExpectExec(`INSERT INTO "users"`).
					WithArgs("user-123", "abobus", "backend", true, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				rows := sqlmock.NewRows([]string{
					"team_name", "created_at", "updated_at",
				}).AddRow(
					"backend", time.Now(), time.Now(),
				)
				m.ExpectQuery(`SELECT * FROM "teams"`).
					WithArgs("backend", "backend", 1).
					WillReturnRows(rows)
				userRows := sqlmock.NewRows([]string{
					"user_id", "username", "team_name", "is_active", "created_at", "updated_at",
				}).AddRow(
					"user-123", "abobus", "backend", true, time.Now(), time.Now(),
				)
				m.ExpectQuery(`SELECT * FROM "users"`).
					WithArgs("backend").
					WillReturnRows(userRows)
				m.ExpectCommit()
			},
			wantErr: nil,
			wantTeam: &team.Team{
				TeamName: "backend",
				Members: []*user.User{
					{
						UserID:   "user-123",
						Username: "abobus",
						TeamName: "backend",
						IsActive: true,
					},
				},
			},
		},
		{
			name: "team already exists",
			args: createTeamArgs{
				teamName: "backend",
				members:  []*user.User{},
			},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectExec(`INSERT INTO "teams"`).
					WithArgs("backend", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errors.New("SQLSTATE 23505"))
				m.ExpectRollback()
			},
			wantErr:  team.ErrTeamExists,
			wantTeam: nil,
		},
		{
			name: "sql error",
			args: createTeamArgs{
				teamName: "backend",
				members:  []*user.User{},
			},
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectExec(`INSERT INTO "teams"`).
					WithArgs("backend", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(gorm.ErrInvalidDB)
				m.ExpectRollback()
			},
			wantErr:  gorm.ErrInvalidDB,
			wantTeam: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := newMockDB(t)
			defer cleanup()

			logger := zap.NewNop().Sugar()
			repo := team.NewTeamsRepoPg(logger, db)

			tt.mockFunc(mock)

			got, err := repo.CreateTeam(tt.args.teamName, tt.args.members)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.wantTeam != nil {
				require.NotNil(t, got)
				require.Equal(t, tt.wantTeam.TeamName, got.TeamName)
				require.Len(t, got.Members, len(tt.wantTeam.Members))
				for i, member := range got.Members {
					require.Equal(t, tt.wantTeam.Members[i].UserID, member.UserID)
					require.Equal(t, tt.wantTeam.Members[i].Username, member.Username)
					require.Equal(t, tt.wantTeam.Members[i].TeamName, member.TeamName)
					require.Equal(t, tt.wantTeam.Members[i].IsActive, member.IsActive)
				}
			} else {
				require.Nil(t, got)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTeamsRepoPg_GetTeam(t *testing.T) {
	tests := []struct {
		name     string
		teamName string
		mockFunc func(sqlmock.Sqlmock)
		wantErr  error
		wantTeam *team.Team
	}{
		{
			name:     "success",
			teamName: "backend",
			mockFunc: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"team_name", "created_at", "updated_at",
				}).AddRow(
					"backend", time.Now(), time.Now(),
				)
				m.ExpectQuery(`SELECT * FROM "teams"`).
					WithArgs("backend", 1).
					WillReturnRows(rows)
				userRows := sqlmock.NewRows([]string{
					"user_id", "username", "team_name", "is_active", "created_at", "updated_at",
				}).AddRow(
					"user-123", "abobus", "backend", true, time.Now(), time.Now(),
				)
				m.ExpectQuery(`SELECT * FROM "users"`).
					WithArgs("backend").
					WillReturnRows(userRows)
			},
			wantErr: nil,
			wantTeam: &team.Team{
				TeamName: "backend",
				Members: []*user.User{
					{
						UserID:   "user-123",
						Username: "abobus",
						TeamName: "backend",
						IsActive: true,
					},
				},
			},
		},
		{
			name:     "team not found",
			teamName: "unknown",
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT * FROM "teams"`).
					WithArgs("unknown", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr:  gorm.ErrRecordNotFound,
			wantTeam: nil,
		},
		{
			name:     "sql error",
			teamName: "backend",
			mockFunc: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(`SELECT * FROM "teams"`).
					WithArgs("backend", 1).
					WillReturnError(gorm.ErrInvalidDB)
			},
			wantErr:  gorm.ErrInvalidDB,
			wantTeam: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := newMockDB(t)
			defer cleanup()

			logger := zap.NewNop().Sugar()
			repo := team.NewTeamsRepoPg(logger, db)

			tt.mockFunc(mock)

			got, err := repo.GetTeam(tt.teamName)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.wantTeam != nil {
				require.NotNil(t, got)
				require.Equal(t, tt.wantTeam.TeamName, got.TeamName)
				require.Len(t, got.Members, len(tt.wantTeam.Members))
				for i, member := range got.Members {
					require.Equal(t, tt.wantTeam.Members[i].UserID, member.UserID)
					require.Equal(t, tt.wantTeam.Members[i].Username, member.Username)
					require.Equal(t, tt.wantTeam.Members[i].TeamName, member.TeamName)
					require.Equal(t, tt.wantTeam.Members[i].IsActive, member.IsActive)
				}
			} else {
				require.Nil(t, got)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
