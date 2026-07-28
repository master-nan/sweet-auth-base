package service

import (
	"backend/dto/request"
	"backend/enum"
	testutil "backend/internal/test"
	"backend/model"
	"encoding/json"
	"strings"
	"testing"
)

func TestOrgServiceSyncBatchReadBoundary(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	batches := []model.OrgSyncBatch{
		{
			Basic:        model.Basic{Id: 1, State: true},
			BatchNo:      "ORG-SYNC-001",
			SyncType:     "incremental",
			ObjectScope:  "employee",
			TotalCount:   2,
			SuccessCount: 1,
			FailedCount:  1,
			Status:       "failed",
			ErrorSummary: "one employee failed",
		},
		{
			Basic:        model.Basic{Id: 2, State: true},
			BatchNo:      "ORG-SYNC-002",
			SyncType:     "full",
			ObjectScope:  "all",
			TotalCount:   1,
			SuccessCount: 1,
			Status:       "success",
		},
	}
	testutil.MustCreate(t, db, &batches)

	result, err := orgService.QuerySyncBatches(nil, request.OrgSyncBatchQueryReq{
		Basic:  request.Basic{Page: 1, Num: 10},
		Status: "failed",
	}, orgSyncBatchQueryTable())
	if err != nil {
		t.Fatalf("query sync batches: %v", err)
	}
	if result.Total != 1 || len(result.Data) != 1 {
		t.Fatalf("unexpected sync batch query result: %+v", result)
	}
	if result.Data[0].Id != 1 || !result.Data[0].HasError {
		t.Fatalf("failed batch summary is invalid: %+v", result.Data[0])
	}

	detail, err := orgService.GetSyncBatchDetail(nil, 1)
	if err != nil {
		t.Fatalf("get sync batch detail: %v", err)
	}
	encodedDetail, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal sync batch detail: %v", err)
	}
	if strings.Contains(string(encodedDetail), "one employee failed") {
		t.Fatalf("ordinary detail leaked error summary: %s", encodedDetail)
	}

	errorDetail, err := orgService.GetSyncBatchError(nil, 1)
	if err != nil {
		t.Fatalf("get sync batch error: %v", err)
	}
	if errorDetail.ErrorSummary != "one employee failed" {
		t.Fatalf("unexpected sync batch error detail: %+v", errorDetail)
	}
}

func TestOrgServiceSyncRecordReadBoundaryAndLocalObjectFilter(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	batch := model.OrgSyncBatch{
		Basic:       model.Basic{Id: 10, State: true},
		BatchNo:     "ORG-SYNC-010",
		SyncType:    "incremental",
		ObjectScope: "employee",
		Status:      "failed",
	}
	testutil.MustCreate(t, db, &batch)
	localEmployeeID := 201
	otherEmployeeID := 202
	records := []model.OrgSyncRecord{
		{
			Basic:          model.Basic{Id: 11, State: true},
			BatchId:        batch.Id,
			ObjectType:     "employee",
			SourceId:       "source-employee-201",
			SourceCode:     "EMP-201",
			LocalId:        &localEmployeeID,
			Action:         "update",
			Status:         "failed",
			ErrorCode:      "org_employee_missing",
			ErrorMessage:   "employee dependency is unavailable",
			DependencyType: "employee",
			DependencyKey:  "source-employee-201",
		},
		{
			Basic:      model.Basic{Id: 12, State: true},
			BatchId:    batch.Id,
			ObjectType: "employee",
			SourceId:   "source-employee-202",
			SourceCode: "EMP-202",
			LocalId:    &otherEmployeeID,
			Action:     "update",
			Status:     "success",
		},
	}
	testutil.MustCreate(t, db, &records)

	result, err := orgService.QuerySyncRecords(nil, request.OrgSyncRecordQueryReq{
		Basic:      request.Basic{Page: 1, Num: 10},
		ObjectType: "employee",
		LocalId:    &localEmployeeID,
		Status:     "failed",
	}, orgSyncRecordQueryTable())
	if err != nil {
		t.Fatalf("query sync records: %v", err)
	}
	if result.Total != 1 || len(result.Data) != 1 {
		t.Fatalf("unexpected sync record query result: %+v", result)
	}
	if result.Data[0].Id != 11 || !result.Data[0].HasError {
		t.Fatalf("failed sync record summary is invalid: %+v", result.Data[0])
	}

	detail, err := orgService.GetSyncRecordDetail(nil, 11)
	if err != nil {
		t.Fatalf("get sync record detail: %v", err)
	}
	encodedDetail, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal sync record detail: %v", err)
	}
	if strings.Contains(string(encodedDetail), "source-employee-201") ||
		strings.Contains(string(encodedDetail), "employee dependency is unavailable") {
		t.Fatalf("ordinary detail leaked source identity or error message: %s", encodedDetail)
	}

	errorDetail, err := orgService.GetSyncRecordError(nil, 11)
	if err != nil {
		t.Fatalf("get sync record error: %v", err)
	}
	if errorDetail.ErrorCode != "org_employee_missing" ||
		errorDetail.ErrorMessage != "employee dependency is unavailable" ||
		errorDetail.DependencyKey != "source-employee-201" {
		t.Fatalf("unexpected sync record error detail: %+v", errorDetail)
	}
}

func orgSyncBatchQueryTable() model.SysTable {
	return organizationQueryTestTable("org_sync_batch", map[string]enum.SysTableFieldType{
		"id":            enum.BigIntFieldType,
		"execution_id":  enum.BigIntFieldType,
		"sync_type":     enum.VarcharFieldType,
		"object_scope":  enum.VarcharFieldType,
		"status":        enum.VarcharFieldType,
		"batch_no":      enum.VarcharFieldType,
		"started_at":    enum.DatetimeFieldType,
		"completed_at":  enum.DatetimeFieldType,
		"failed_count":  enum.IntFieldType,
		"success_count": enum.IntFieldType,
	})
}

func orgSyncRecordQueryTable() model.SysTable {
	return organizationQueryTestTable("org_sync_record", map[string]enum.SysTableFieldType{
		"id":                    enum.BigIntFieldType,
		"batch_id":              enum.BigIntFieldType,
		"execution_id":          enum.BigIntFieldType,
		"object_type":           enum.VarcharFieldType,
		"local_id":              enum.BigIntFieldType,
		"action":                enum.VarcharFieldType,
		"status":                enum.VarcharFieldType,
		"dependency_type":       enum.VarcharFieldType,
		"local_handling_status": enum.VarcharFieldType,
		"source_code":           enum.VarcharFieldType,
		"error_code":            enum.VarcharFieldType,
	})
}

func organizationQueryTestTable(
	tableCode string,
	fields map[string]enum.SysTableFieldType,
) model.SysTable {
	table := model.SysTable{
		Basic:     model.Basic{Id: 1, State: true},
		TableCode: tableCode,
	}
	for code, fieldType := range fields {
		table.TableFields = append(table.TableFields, model.SysTableField{
			FieldCode:        code,
			FieldType:        fieldType,
			IsPrimaryKey:     code == "id",
			IsListShow:       true,
			IsQuickSearch:    code == "batch_no" || code == "source_code" || code == "error_code",
			IsAdvancedSearch: true,
			IsSort:           true,
		})
	}
	return table
}
