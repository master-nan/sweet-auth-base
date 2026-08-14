package middleware

import (
	"backend/dto/response"
	apperrors "backend/internal/errors"
	"net/http"
)

// toClientError is the single HTTP adapter for stable application errors.
func toClientError(err error) (*response.AdminError, bool) {
	applicationErr, classified := apperrors.Classify(err)
	if applicationErr == nil {
		return nil, false
	}
	return &response.AdminError{
		StatusCode:   httpStatusForErrorKind(applicationErr.Kind),
		ErrorCode:    applicationErr.Code,
		ErrorMessage: applicationErr.SafeMessage,
		Success:      false,
	}, classified
}

func httpStatusForErrorKind(kind apperrors.Kind) int {
	switch kind {
	case apperrors.KindInvalidArgument:
		return http.StatusBadRequest
	case apperrors.KindUnauthenticated:
		return http.StatusUnauthorized
	case apperrors.KindForbidden:
		return http.StatusForbidden
	case apperrors.KindNotFound:
		return http.StatusNotFound
	case apperrors.KindConflict:
		return http.StatusConflict
	case apperrors.KindUnprocessable:
		return http.StatusUnprocessableEntity
	case apperrors.KindPayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case apperrors.KindRateLimited:
		return http.StatusTooManyRequests
	case apperrors.KindDependencyFailed:
		return http.StatusBadGateway
	case apperrors.KindUnavailable:
		return http.StatusServiceUnavailable
	case apperrors.KindTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func shouldLogApplicationError(err error, classified bool) bool {
	if !classified {
		return true
	}
	category := apperrors.CategoryOf(err)
	if category == apperrors.CategoryDatabase || category == apperrors.CategorySystem {
		return true
	}
	switch apperrors.KindOf(err) {
	case apperrors.KindInternal, apperrors.KindDependencyFailed, apperrors.KindUnavailable, apperrors.KindTimeout:
		return true
	default:
		return false
	}
}
