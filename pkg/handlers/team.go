package handlers

import (
	"assignerPR/pkg/handlers/apidto"
	"assignerPR/pkg/handlers/apierr"
	"assignerPR/pkg/pullrequest"
	"assignerPR/pkg/team"
	"assignerPR/pkg/user"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TeamHandler struct {
	prRepo    pullrequest.PullRequestsRepo
	teamsRepo team.TeamsRepo
	logger    *zap.SugaredLogger
}

func NewTeamHandler(logger *zap.SugaredLogger, teamsRepo team.TeamsRepo, prRepo pullrequest.PullRequestsRepo) *TeamHandler {
	return &TeamHandler{
		prRepo:    prRepo,
		teamsRepo: teamsRepo,
		logger:    logger,
	}
}

type teamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

func toDomainUsers(teamName string, members []teamMemberReq) []*user.User {
	out := make([]*user.User, 0, len(members))
	for _, m := range members {
		out = append(out, &user.User{
			UserID:   m.UserID,
			Username: m.Username,
			TeamName: teamName,
			IsActive: m.IsActive,
		})
	}
	return out
}

func toTeamMembers(users []*user.User) []teamMember {
	res := make([]teamMember, 0, len(users))
	for _, u := range users {
		res = append(res, teamMember{
			UserID:   u.UserID,
			Username: u.Username,
			IsActive: u.IsActive,
		})
	}
	return res
}

type teamMemberReq struct {
	UserID   string `json:"user_id" binding:"required"`
	Username string `json:"username" binding:"required"`
	IsActive bool   `json:"is_active" binding:"required"`
}

type addTeamReq struct {
	TeamName string          `json:"team_name" binding:"required"`
	Members  []teamMemberReq `json:"members" binding:"required,dive"`
}

type teamWithMembersResp struct {
	TeamName string       `json:"team_name" binding:"required"`
	Members  []teamMember `json:"members" binding:"required,dive"`
}

type addTeamResp struct {
	Team teamWithMembersResp `json:"team"`
}

func (h *TeamHandler) AddTeam(c *gin.Context) {
	var req addTeamReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.WriteApiErrJSON(c, http.StatusBadRequest, apierr.BadRequest)
		h.logger.Warnw("error parsing request", "error", err)
		return
	}

	users := toDomainUsers(req.TeamName, req.Members)
	returnTeam, err := h.teamsRepo.CreateTeam(req.TeamName, users)
	if err != nil {
		if apierr.Handle(c, err) {
			h.logger.Warnw("error creating team", "error", err)
			return
		}

		h.logger.Errorw("error creating team", "error", err)
		apierr.WriteApiErrJSON(c, http.StatusInternalServerError, apierr.InternalServerError)
		return
	}

	members := toTeamMembers(returnTeam.Members)
	c.JSON(http.StatusCreated, addTeamResp{
		Team: teamWithMembersResp{
			TeamName: returnTeam.TeamName,
			Members:  members,
		},
	})
}

type getTeamResp struct {
	TeamName string       `json:"team_name"`
	Members  []teamMember `json:"members"`
}

func (h *TeamHandler) GetTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		apierr.WriteApiErrJSON(c, http.StatusBadRequest, apierr.BadRequest)
		h.logger.Warnw("no team name provided")
		return
	}

	teamToReturn, err := h.teamsRepo.GetTeam(teamName)
	if err != nil {
		if apierr.Handle(c, err) {
			h.logger.Warnw("error getting teamToReturn", "error", err)
			return
		}

		h.logger.Errorw("error getting teamToReturn", "error", err)
		apierr.WriteApiErrJSON(c, http.StatusInternalServerError, apierr.InternalServerError)
		return
	}

	members := toTeamMembers(teamToReturn.Members)
	c.JSON(http.StatusOK, getTeamResp{
		TeamName: teamToReturn.TeamName,
		Members:  members,
	})
}

type teamStatsResponse struct {
	TeamName  string                      `json:"team_name"`
	TeamStats map[string]apidto.UserStats `json:"team_stats"`
}

func (h *TeamHandler) StatsTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		apierr.WriteApiErrJSON(c, http.StatusBadRequest, apierr.BadRequest)
		h.logger.Warnw("no team name provided")
		return
	}

	usersStats, err := h.prRepo.GetTeamPRStats(teamName)
	if err != nil {
		if apierr.Handle(c, err) {
			h.logger.Warnw("mapped error getting team stats", "error", err)
			return
		}
		h.logger.Errorw("unknown error getting team stats", "error", err)
		apierr.WriteApiErrJSON(c, http.StatusInternalServerError, apierr.InternalServerError)
		return
	}

	userStatsMap := apidto.FromUserStatsSliceToMap(usersStats)
	c.JSON(http.StatusOK, teamStatsResponse{
		TeamName:  teamName,
		TeamStats: userStatsMap,
	})
}
