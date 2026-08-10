package integration

import (
	myerrors "backend/internal/errors"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSyncResultConsumerRegistryResolveAndDuplicateProtection(t *testing.T) {
	consumer := SyncResultConsumerFunc(func(context.Context, SyncConsumptionRequest) (SyncConsumptionResult, error) {
		return NewSyncConsumptionResult(true, "", 2, 0, "ORG-1")
	})
	registration := syncConsumerTestRegistration(consumer)
	registry, err := NewStaticSyncConsumerRegistry(registration)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := registry.Resolve("test_sync", 1)
	if err != nil || resolved.Metadata().Version != 1 {
		t.Fatalf("resolve=%+v err=%v", resolved.Metadata(), err)
	}
	if _, err := registry.Resolve("test_sync", 2); !errors.Is(err, myerrors.ErrSyncConsumerNotRegistered) {
		t.Fatalf("version drift error=%v", err)
	}
	if _, err := NewStaticSyncConsumerRegistry(registration, registration); !errors.Is(err, myerrors.ErrSyncConsumerDuplicate) {
		t.Fatalf("duplicate error=%v", err)
	}
}

func TestSyncConsumptionRequestIsImmutableAndRedacted(t *testing.T) {
	body := []byte(`{"employees":[{"id":"10001"}]}`)
	digest := sha256.Sum256(body)
	request, err := NewSyncConsumptionRequest(SyncConsumptionRequestInput{
		ExecutionNo: "INT-1", SyncBatchNo: "SYNC-1", TaskCode: "org_employee_sync", TaskVersion: 1,
		SliceNo: 1, ContentType: "application/json; charset=utf-8", ResponseSize: int64(len(body)),
		ResponseHash: hex.EncodeToString(digest[:]), Body: body,
	})
	if err != nil {
		t.Fatal(err)
	}
	body[0] = 'x'
	copyBody := request.Body()
	copyBody[1] = 'x'
	if string(request.Body()) != `{"employees":[{"id":"10001"}]}` {
		t.Fatal("request body was not isolated")
	}
	payload, err := json.Marshal(request)
	if err != nil || string(payload) != `{}` {
		t.Fatalf("request serialized private body: %s err=%v", payload, err)
	}
	for _, forbidden := range []string{"Authorization", "Cookie", "Credential", "Token", "10001"} {
		if strings.Contains(request.String(), forbidden) || strings.Contains(request.GoString(), forbidden) {
			t.Fatalf("safe formatter leaked %q", forbidden)
		}
	}
}

func TestResolvedSyncConsumerSuccessFailureTimeoutAndPanic(t *testing.T) {
	tests := []struct {
		name     string
		consumer SyncResultConsumer
		wantErr  error
		wantOK   bool
	}{
		{name: "success", consumer: SyncResultConsumerFunc(func(context.Context, SyncConsumptionRequest) (SyncConsumptionResult, error) {
			return NewSyncConsumptionResult(true, "", 1, 0, "ORG-1")
		}), wantOK: true},
		{name: "business failure", consumer: SyncResultConsumerFunc(func(context.Context, SyncConsumptionRequest) (SyncConsumptionResult, error) {
			return NewSyncConsumptionResult(false, "org_payload_invalid", 0, 1, "ORG-2")
		})},
		{name: "error", consumer: SyncResultConsumerFunc(func(context.Context, SyncConsumptionRequest) (SyncConsumptionResult, error) {
			return SyncConsumptionResult{}, errors.New("sensitive internal failure")
		}), wantErr: myerrors.ErrSyncBusinessProcessingFailed},
		{name: "timeout", consumer: SyncResultConsumerFunc(func(ctx context.Context, _ SyncConsumptionRequest) (SyncConsumptionResult, error) {
			<-ctx.Done()
			return SyncConsumptionResult{}, ctx.Err()
		}), wantErr: myerrors.ErrSyncConsumerTimeout},
		{name: "panic", consumer: SyncResultConsumerFunc(func(context.Context, SyncConsumptionRequest) (SyncConsumptionResult, error) {
			panic("secret panic")
		}), wantErr: myerrors.ErrSyncConsumerPanic},
	}
	request := syncConsumerTestRequest(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registration := syncConsumerTestRegistration(test.consumer)
			if test.name == "timeout" {
				registration.Metadata.MaxDuration = time.Millisecond
			}
			registry, err := NewStaticSyncConsumerRegistry(registration)
			if err != nil {
				t.Fatal(err)
			}
			resolved, err := registry.Resolve("test_sync", 1)
			if err != nil {
				t.Fatal(err)
			}
			result, err := resolved.Consume(context.Background(), request)
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("error=%v want=%v", err, test.wantErr)
			}
			if test.wantErr == nil && err != nil {
				t.Fatal(err)
			}
			if test.wantOK && !result.Success() {
				t.Fatalf("result=%+v", result)
			}
			if test.name == "business failure" && (result.Success() || result.ReasonCode() != "org_payload_invalid") {
				t.Fatalf("business result=%+v", result)
			}
		})
	}
}

func syncConsumerTestRegistration(consumer SyncResultConsumer) SyncConsumerRegistration {
	return SyncConsumerRegistration{Metadata: SyncConsumerMetadata{
		Code: "test_sync", Version: 1, Name: "Test Sync", Status: SyncConsumerStatusEnabled,
		ContentTypes: []string{"application/json"}, MaxResponseBytes: 1 << 20, MaxDuration: time.Second,
		CheckpointModes: []string{"none", "timestamp"},
	}, Consumer: consumer}
}

func syncConsumerTestRequest(t *testing.T) SyncConsumptionRequest {
	t.Helper()
	body := []byte(`{"ok":true}`)
	digest := sha256.Sum256(body)
	request, err := NewSyncConsumptionRequest(SyncConsumptionRequestInput{
		ExecutionNo: "INT-1", SyncBatchNo: "SYNC-1", TaskCode: "test_sync", TaskVersion: 1, SliceNo: 1,
		ContentType: "application/json", ResponseSize: int64(len(body)), ResponseHash: hex.EncodeToString(digest[:]), Body: body,
	})
	if err != nil {
		t.Fatal(err)
	}
	return request
}
