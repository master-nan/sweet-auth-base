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
		{name: "credential not found", err: ErrIntegrationCredentialNotFound, statusCode: http.StatusNotFound, errorCode: ErrorCodeIntegrationCredentialNotFound},
		{name: "credential system mismatch", err: ErrIntegrationCredentialSystemMismatch, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationCredentialSystemMismatch},
		{name: "credential interface mismatch", err: ErrIntegrationCredentialInterfaceMismatch, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationCredentialInterfaceMismatch},
		{name: "credential inactive", err: ErrIntegrationCredentialInactive, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationCredentialInactive},
		{name: "credential expired", err: ErrIntegrationCredentialExpired, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationCredentialExpired},
		{name: "credential revoked", err: ErrIntegrationCredentialRevoked, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationCredentialRevoked},
		{name: "credential type unsupported", err: ErrIntegrationCredentialTypeUnsupported, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationCredentialTypeUnsupported},
		{name: "credential secret missing", err: ErrIntegrationCredentialSecretMissing, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationCredentialSecretMissing},
		{name: "credential decrypt failed", err: ErrIntegrationCredentialDecryptFailed, statusCode: http.StatusInternalServerError, errorCode: ErrorCodeIntegrationCredentialDecryptFailed},
		{name: "credential material invalid", err: ErrIntegrationCredentialMaterialInvalid, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationCredentialMaterialInvalid},
		{name: "credential injection invalid", err: ErrIntegrationCredentialInjectionInvalid, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationCredentialInjectionInvalid},
		{name: "input missing", err: ErrIntegrationExecutionInputMissing, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationExecutionInputMissing},
		{name: "input invalid", err: ErrIntegrationExecutionInputInvalid, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationExecutionInputInvalid},
		{name: "input semantic too large", err: ErrIntegrationExecutionInputSemanticTooLarge, statusCode: http.StatusRequestEntityTooLarge, errorCode: ErrorCodeIntegrationExecutionInputSemanticTooLarge},
		{name: "input storage too large", err: ErrIntegrationExecutionInputStorageTooLarge, statusCode: http.StatusRequestEntityTooLarge, errorCode: ErrorCodeIntegrationExecutionInputStorageTooLarge},
		{name: "input size mismatch", err: ErrIntegrationExecutionInputSizeMismatch, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationExecutionInputSizeMismatch},
		{name: "input contract mismatch", err: ErrIntegrationExecutionInputContractMismatch, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationExecutionInputContractMismatch},
		{name: "input hash mismatch", err: ErrIntegrationExecutionInputHashMismatch, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationExecutionInputHashMismatch},
		{name: "input version unsupported", err: ErrIntegrationExecutionInputVersionUnsupported, statusCode: http.StatusConflict, errorCode: ErrorCodeIntegrationExecutionInputVersionUnsupported},
		{name: "path missing", err: ErrIntegrationExecutionPathParameterMissing, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationExecutionPathParameterMissing},
		{name: "path unknown", err: ErrIntegrationExecutionPathParameterUnknown, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationExecutionPathParameterUnknown},
		{name: "query invalid", err: ErrIntegrationExecutionQueryParameterInvalid, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationExecutionQueryParameterInvalid},
		{name: "header not allowed", err: ErrIntegrationExecutionHeaderNotAllowed, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationExecutionHeaderNotAllowed},
		{name: "body invalid", err: ErrIntegrationExecutionBodyInvalid, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationExecutionBodyInvalid},
		{name: "sensitive input rejected", err: ErrIntegrationExecutionSensitiveInputRejected, statusCode: http.StatusBadRequest, errorCode: ErrorCodeIntegrationExecutionSensitiveInputRejected},
		{name: "input storage failed", err: ErrIntegrationExecutionInputStorageFailed, statusCode: http.StatusInternalServerError, errorCode: ErrorCodeIntegrationExecutionInputStorageFailed},
		{name: "input load failed", err: ErrIntegrationExecutionInputLoadFailed, statusCode: http.StatusInternalServerError, errorCode: ErrorCodeIntegrationExecutionInputLoadFailed},
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
