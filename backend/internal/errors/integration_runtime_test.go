package errors

import (
	"net/http"
	"testing"
)

func TestIntegrationRuntimeErrorsExposeStableCodes(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		errorCode  int
	}{
		{name: "not found", err: ErrIntegrationExecutionNotFound, statusCode: http.StatusNotFound, errorCode: ErrorCodeIntegrationExecutionNotFound},
		{name: "idempotency", err: ErrIntegrationExecutionIdempotencyConflict, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationExecutionIdempotencyConflict},
		{name: "status", err: ErrIntegrationExecutionStatusInvalid, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationExecutionStatusInvalid},
		{name: "revision", err: ErrIntegrationExecutionRevisionConflict, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationExecutionRevisionConflict},
		{name: "configuration", err: ErrIntegrationExecutionConfigurationInvalid, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationExecutionConfigurationInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientError, classified := ToClientError(test.err)
			if !classified || clientError.StatusCode != test.statusCode || clientError.ErrorCode != test.errorCode {
				t.Fatalf("client error = %+v classified=%v", clientError, classified)
			}
		})
	}
}
