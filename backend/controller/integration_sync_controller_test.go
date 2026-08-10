package controller

import (
	"backend/dto/response"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type syncTaskApplicationStub struct {
	syncTaskApplication
	detail response.SyncTaskDetailRes
}

func (s syncTaskApplicationStub) GetSyncTask(context.Context, int) (response.SyncTaskDetailRes, error) {
	return s.detail, nil
}

type syncBatchApplicationStub struct {
	syncBatchApplication
	detail response.SyncBatchDetailRes
}

func (s syncBatchApplicationStub) GetSyncBatch(context.Context, int) (response.SyncBatchDetailRes, error) {
	return s.detail, nil
}

func TestIntegrationSyncControllerReturnsWhitelistedDetails(t *testing.T) {
	controller := &IntegrationSyncController{
		tasks:   syncTaskApplicationStub{detail: response.SyncTaskDetailRes{SyncTaskListRes: response.SyncTaskListRes{ID: 1, TaskCode: "employee_sync", TaskName: "Employee Sync", Version: 1}}},
		batches: syncBatchApplicationStub{detail: response.SyncBatchDetailRes{SyncBatchListRes: response.SyncBatchListRes{ID: 2, BatchNo: "SYNC-2", TaskCode: "employee_sync"}}},
	}
	for _, testCase := range []struct {
		name string
		call func(*gin.Context)
	}{
		{name: "task", call: controller.TaskDetail},
		{name: "batch", call: controller.BatchDetail},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("GET", "/detail", nil)
			ctx.Params = gin.Params{{Key: "id", Value: "1"}}
			testCase.call(ctx)
			value, ok := ctx.Get("response")
			if !ok {
				t.Fatal("controller did not set unified response")
			}
			payload, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"static_input", "input_plan\"", "input_snapshot", "credential", "authorization", "gmt_delete", "create_user"} {
				if strings.Contains(strings.ToLower(string(payload)), forbidden) {
					t.Fatalf("detail leaked %q: %s", forbidden, payload)
				}
			}
		})
	}
}
