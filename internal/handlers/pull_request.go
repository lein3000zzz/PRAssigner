package handlers

import (
	"assignerPR/internal/handlers/apidto"
	"assignerPR/internal/handlers/apierr"
	"assignerPR/internal/pullrequest"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PullRequestHandler struct {
	repo   pullrequest.PullRequestsRepo
	logger *zap.SugaredLogger
}

func NewPullRequestHandler(logger *zap.SugaredLogger, repo pullrequest.PullRequestsRepo) *PullRequestHandler {
	return &PullRequestHandler{
		repo:   repo,
		logger: logger,
	}
}

type createPRReq struct {
	PullRequestID   string `json:"pull_request_id" binding:"required"`
	PullRequestName string `json:"pull_request_name" binding:"required"`
	AuthorID        string `json:"author_id" binding:"required"`
}

type prResp struct {
	PR apidto.PullRequest `json:"pr"`
}

func (h *PullRequestHandler) CreatePR(c *gin.Context) {
	var req createPRReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.WriteApiErrJSON(c, http.StatusBadRequest, apierr.BadRequest)
		h.logger.Warnw("error parsing request", "error", err)
		return
	}

	pr, err := h.repo.CreatePR(req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		if apierr.Handle(c, err) {
			h.logger.Warnw("mapped error creating pull request", "error", err)
			return
		}

		h.logger.Errorw("CreatePR failed, couldnt map the error", "err", err)
		apierr.WriteApiErrJSON(c, http.StatusInternalServerError, apierr.InternalServerError)
		return
	}

	c.JSON(http.StatusCreated, prResp{
		apidto.FromPR(pr),
	})
}

type mergePRReq struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
}

type mergePRResp struct {
	PR apidto.PullRequest `json:"pr"`
}

func (h *PullRequestHandler) Merge(c *gin.Context) {
	var req mergePRReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.WriteApiErrJSON(c, http.StatusBadRequest, apierr.BadRequest)
		h.logger.Warnw("error parsing request", "error", err)
		return
	}

	pr, err := h.repo.Merge(req.PullRequestID)
	if err != nil {
		if apierr.Handle(c, err) {
			h.logger.Warnw("mapped error creating pull request", "error", err)
			return
		}
		h.logger.Errorw("Merge failed, couldnt map the error", "err", err)
		apierr.WriteApiErrJSON(c, http.StatusInternalServerError, apierr.InternalServerError)
		return
	}

	c.JSON(http.StatusOK, mergePRResp{
		apidto.FromPR(pr),
	})
}

type reassignPRReq struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
	OldReviewerID string `json:"old_reviewer_id" binding:"required"`
}

type reassignPRResp struct {
	PR         apidto.PullRequest `json:"pr"`
	ReplacedBy string             `json:"replaced_by"`
}

func (h *PullRequestHandler) ReassignPR(c *gin.Context) {
	var req reassignPRReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.WriteApiErrJSON(c, http.StatusBadRequest, apierr.BadRequest)
		h.logger.Warnw("error parsing request", "error", err)
		return
	}

	pr, replacedBy, err := h.repo.Reassign(req.PullRequestID, req.OldReviewerID)
	if err != nil {
		if apierr.Handle(c, err) {
			h.logger.Warnw("mapped error creating pull request", "error", err)
			return
		}
		h.logger.Errorw("Reassign failed, couldnt map the error", "err", err)
		apierr.WriteApiErrJSON(c, http.StatusInternalServerError, apierr.InternalServerError)
		return
	}

	c.JSON(http.StatusOK, reassignPRResp{
		PR:         apidto.FromPR(pr),
		ReplacedBy: replacedBy,
	})
}
