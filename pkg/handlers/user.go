package handlers

import (
	"assignerPR/pkg/handlers/apidto"
	"assignerPR/pkg/handlers/apierr"
	"assignerPR/pkg/pullrequest"
	"assignerPR/pkg/user"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UserHandler struct {
	prRepo   pullrequest.PullRequestsRepo
	userRepo user.UsersRepo
	logger   *zap.SugaredLogger
}

func NewUserHandler(logger *zap.SugaredLogger, userRepo user.UsersRepo, prRepo pullrequest.PullRequestsRepo) *UserHandler {
	return &UserHandler{
		prRepo:   prRepo,
		userRepo: userRepo,
		logger:   logger,
	}
}

type setActiveReq struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type userResp struct {
	User apidto.User `json:"user"`
}

func (h *UserHandler) SetIsActive(c *gin.Context) {
	var req setActiveReq

	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.WriteApiErrJSON(c, http.StatusBadRequest, apierr.BadRequest)
		h.logger.Warnw("error parsing request", "error", err)
		return
	}

	usr, err := h.userRepo.SetIsActive(req.UserID, req.IsActive)
	if err != nil {
		if apierr.Handle(c, err) {
			h.logger.Warnw("error setting user", "error", err)
			return
		}

		apierr.WriteApiErrJSON(c, http.StatusInternalServerError, apierr.InternalServerError)
		h.logger.Warnw("error setting user", "error", err)
		return
	}

	c.JSON(http.StatusOK, userResp{
		User: apidto.FromUser(usr),
	})
}

type getPRResp struct {
	UserID       string           `json:"user_id"`
	PullRequests []apidto.PRShort `json:"pull_requests"`
}

func (h *UserHandler) GetUserReviews(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		apierr.WriteApiErrJSON(c, http.StatusBadRequest, apierr.BadRequest)
		h.logger.Warnw("no user_id provided")
		return
	}

	prs, err := h.prRepo.ListPRsByReviewer(userID)
	if err != nil {
		if apierr.Handle(c, err) {
			h.logger.Warnw("mapped error listing user reviews", "error", err)
			return
		}
		apierr.WriteApiErrJSON(c, http.StatusInternalServerError, apierr.InternalServerError)
		h.logger.Errorw("error listing user reviews", "userID", userID, "err", err)
		return
	}

	c.JSON(http.StatusOK, getPRResp{
		UserID:       userID,
		PullRequests: apidto.FromPRsToShort(prs),
	})
}
