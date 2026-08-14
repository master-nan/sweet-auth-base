package errors

import (
	"testing"
)

func TestIntegrationRuntimeErrorsExposeStableCodes(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		kind      Kind
		errorCode int
	}{
		{name: "not found", err: ErrIntegrationExecutionNotFound, kind: KindNotFound, errorCode: ErrorCodeIntegrationExecutionNotFound},
		{name: "idempotency", err: ErrIntegrationExecutionIdempotencyConflict, kind: KindConflict, errorCode: ErrorCodeIntegrationExecutionIdempotencyConflict},
		{name: "status", err: ErrIntegrationExecutionStatusInvalid, kind: KindConflict, errorCode: ErrorCodeIntegrationExecutionStatusInvalid},
		{name: "revision", err: ErrIntegrationExecutionRevisionConflict, kind: KindConflict, errorCode: ErrorCodeIntegrationExecutionRevisionConflict},
		{name: "configuration", err: ErrIntegrationExecutionConfigurationInvalid, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationExecutionConfigurationInvalid},
		{name: "credential not found", err: ErrIntegrationCredentialNotFound, kind: KindNotFound, errorCode: ErrorCodeIntegrationCredentialNotFound},
		{name: "credential system mismatch", err: ErrIntegrationCredentialSystemMismatch, kind: KindConflict, errorCode: ErrorCodeIntegrationCredentialSystemMismatch},
		{name: "credential interface mismatch", err: ErrIntegrationCredentialInterfaceMismatch, kind: KindConflict, errorCode: ErrorCodeIntegrationCredentialInterfaceMismatch},
		{name: "credential inactive", err: ErrIntegrationCredentialInactive, kind: KindConflict, errorCode: ErrorCodeIntegrationCredentialInactive},
		{name: "credential expired", err: ErrIntegrationCredentialExpired, kind: KindConflict, errorCode: ErrorCodeIntegrationCredentialExpired},
		{name: "credential revoked", err: ErrIntegrationCredentialRevoked, kind: KindConflict, errorCode: ErrorCodeIntegrationCredentialRevoked},
		{name: "credential type unsupported", err: ErrIntegrationCredentialTypeUnsupported, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationCredentialTypeUnsupported},
		{name: "credential secret missing", err: ErrIntegrationCredentialSecretMissing, kind: KindConflict, errorCode: ErrorCodeIntegrationCredentialSecretMissing},
		{name: "credential decrypt failed", err: ErrIntegrationCredentialDecryptFailed, kind: KindInternal, errorCode: ErrorCodeIntegrationCredentialDecryptFailed},
		{name: "credential material invalid", err: ErrIntegrationCredentialMaterialInvalid, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationCredentialMaterialInvalid},
		{name: "credential injection invalid", err: ErrIntegrationCredentialInjectionInvalid, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationCredentialInjectionInvalid},
		{name: "input missing", err: ErrIntegrationExecutionInputMissing, kind: KindConflict, errorCode: ErrorCodeIntegrationExecutionInputMissing},
		{name: "input invalid", err: ErrIntegrationExecutionInputInvalid, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationExecutionInputInvalid},
		{name: "input semantic too large", err: ErrIntegrationExecutionInputSemanticTooLarge, kind: KindPayloadTooLarge, errorCode: ErrorCodeIntegrationExecutionInputSemanticTooLarge},
		{name: "input storage too large", err: ErrIntegrationExecutionInputStorageTooLarge, kind: KindPayloadTooLarge, errorCode: ErrorCodeIntegrationExecutionInputStorageTooLarge},
		{name: "input size mismatch", err: ErrIntegrationExecutionInputSizeMismatch, kind: KindConflict, errorCode: ErrorCodeIntegrationExecutionInputSizeMismatch},
		{name: "input contract mismatch", err: ErrIntegrationExecutionInputContractMismatch, kind: KindConflict, errorCode: ErrorCodeIntegrationExecutionInputContractMismatch},
		{name: "input hash mismatch", err: ErrIntegrationExecutionInputHashMismatch, kind: KindConflict, errorCode: ErrorCodeIntegrationExecutionInputHashMismatch},
		{name: "input version unsupported", err: ErrIntegrationExecutionInputVersionUnsupported, kind: KindConflict, errorCode: ErrorCodeIntegrationExecutionInputVersionUnsupported},
		{name: "path missing", err: ErrIntegrationExecutionPathParameterMissing, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationExecutionPathParameterMissing},
		{name: "path unknown", err: ErrIntegrationExecutionPathParameterUnknown, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationExecutionPathParameterUnknown},
		{name: "query invalid", err: ErrIntegrationExecutionQueryParameterInvalid, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationExecutionQueryParameterInvalid},
		{name: "header not allowed", err: ErrIntegrationExecutionHeaderNotAllowed, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationExecutionHeaderNotAllowed},
		{name: "body invalid", err: ErrIntegrationExecutionBodyInvalid, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationExecutionBodyInvalid},
		{name: "sensitive input rejected", err: ErrIntegrationExecutionSensitiveInputRejected, kind: KindInvalidArgument, errorCode: ErrorCodeIntegrationExecutionSensitiveInputRejected},
		{name: "input storage failed", err: ErrIntegrationExecutionInputStorageFailed, kind: KindInternal, errorCode: ErrorCodeIntegrationExecutionInputStorageFailed},
		{name: "input load failed", err: ErrIntegrationExecutionInputLoadFailed, kind: KindInternal, errorCode: ErrorCodeIntegrationExecutionInputLoadFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			applicationError, classified := Classify(test.err)
			if !classified || applicationError.Kind != test.kind || applicationError.Code != test.errorCode {
				t.Fatalf("application error = %+v classified=%v", applicationError, classified)
			}
		})
	}
}
