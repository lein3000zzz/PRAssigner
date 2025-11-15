package apierr

var (
	BadRequest = APIError{
		Code:    "INVALID_REQUEST",
		Message: "invalid request body",
	}
	Unauthorized = APIError{
		Code:    "UNAUTHORIZED_REQUEST",
		Message: "unauthorized request",
	}
	NotFound = APIError{
		Code:    "NOT_FOUND",
		Message: "resource not found",
	}
	InternalServerError = APIError{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: "internal server error",
	}
	TeamExists = APIError{
		Code:    "TEAM_EXISTS",
		Message: "team_name already exists",
	}
	PRExists = APIError{
		Code:    "PR_EXISTS",
		Message: "PR id already exists",
	}
	PRMerged = APIError{
		Code:    "PR_MERGED",
		Message: "cannot reassign on merged PR",
	}
	NotAssigned = APIError{
		Code:    "NOT_ASSIGNED",
		Message: "reviewer is not assigned to this PR",
	}
	NoCandidate = APIError{
		Code:    "NO_CANDIDATE",
		Message: "no active replacement candidate in team",
	}
)
