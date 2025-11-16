package apierr

import (
	"assignerPR/internal/pullrequest"
	"assignerPR/pkg/team"
	"assignerPR/pkg/user"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrResponse struct {
	Error APIError `json:"error"`
}

// Map нужен, чтобы при изменении ошибок (для логов, например, то есть для повышения читаемости,
// либо чего-то еще) респонсы остались те же
func Map(err error) (int, APIError, bool) {
	switch {
	case errors.Is(err, team.ErrTeamExists):
		return http.StatusBadRequest, TeamExists, true

	case errors.Is(err, pullrequest.ErrPRNotFound),
		errors.Is(err, user.ErrUserNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound, NotFound, true

	case errors.Is(err, pullrequest.ErrPRExists):
		return http.StatusConflict, PRExists, true
	case errors.Is(err, pullrequest.ErrPRMerged):
		return http.StatusConflict, PRMerged, true
	case errors.Is(err, pullrequest.ErrNotAssigned):
		return http.StatusConflict, NotAssigned, true
	case errors.Is(err, pullrequest.ErrNoCandidate):
		return http.StatusConflict, NoCandidate, true
	default:
		// Это автоматически не хэндлится, потому что не уточнено ничего, и мы, возможно, хотим захэндлить иначе
		return http.StatusInternalServerError, InternalServerError, false
	}
}

func Handle(c *gin.Context, err error) bool {
	if status, apiErr, ok := Map(err); ok {
		// c.JSON(status, ErrResponse{
		//	 Error: apiErr,
		// })
		WriteApiErrJSON(c, status, apiErr)
		return true
	}

	return false
}

func WriteApiErrJSON(c *gin.Context, status int, apiErr APIError) {
	c.JSON(status, ErrResponse{
		Error: apiErr,
	})
}
