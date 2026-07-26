package response

import (
	"backend/model"
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/datatypes"
)

func TestOrgEmployeeResponseUsesSafeWhitelistAndMaskedContactFields(t *testing.T) {
	userID := 77
	employee := model.OrgEmployee{
		Basic:            model.Basic{Id: 10, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-person-10",
		SourceCode:       "SRC-EMP-10",
		EmployeeNo:       "EMP-10",
		Name:             "Alice",
		Mobile:           "13800138000",
		Email:            "alice@example.com",
		EmploymentStatus: "active",
		SourceVersion:    "version-secret",
		SyncStatus:       "synced",
		UserId:           &userID,
		LocalNote:        "platform note",
		LocalTags:        datatypes.JSON([]byte(`["tag-a"]`)),
	}

	listObject := marshalJSONObject(t, NewOrgEmployeeListRes(employee))
	assertJSONKeysAbsent(t, listObject,
		"source_system_code", "source_id", "source_code", "source_version",
		"sync_status", "mobile", "email", "local_note", "local_tags",
	)
	if got := int(listObject["user_id"].(float64)); got != userID {
		t.Fatalf("expected bound account user_id %d, got %d", userID, got)
	}
	if listObject["binding_status"] != "bound" {
		t.Fatalf("unexpected binding status: %v", listObject["binding_status"])
	}

	detailObject := marshalJSONObject(t, NewOrgEmployeeDetailRes(employee))
	assertJSONKeysAbsent(t, detailObject,
		"source_system_code", "source_id", "source_code", "source_version",
		"sync_status", "mobile", "email",
	)
	if detailObject["mobile_masked"] != "138****8000" {
		t.Fatalf("unexpected masked mobile: %v", detailObject["mobile_masked"])
	}
	if detailObject["email_masked"] != "a***@example.com" {
		t.Fatalf("unexpected masked email: %v", detailObject["email_masked"])
	}
	serialized, err := json.Marshal(detailObject)
	if err != nil {
		t.Fatalf("marshal detail object: %v", err)
	}
	if strings.Contains(string(serialized), employee.Mobile) ||
		strings.Contains(string(serialized), employee.Email) ||
		strings.Contains(string(serialized), employee.SourceId) ||
		strings.Contains(string(serialized), employee.SourceVersion) {
		t.Fatalf("sensitive or internal value leaked: %s", serialized)
	}

	detail := NewOrgEmployeeDetailRes(employee)
	detail.SetBoundAccount(NewOrgBoundUserSummaryRes(userID, "alice"))
	accountObject := marshalJSONObject(t, detail)["bound_account"].(map[string]interface{})
	if accountObject["user_name"] != "alice" {
		t.Fatalf("unexpected bound account summary: %+v", accountObject)
	}
	assertJSONKeysAbsent(t, accountObject,
		"password", "roles", "access_tokens", "phone_number", "email",
	)
	option := NewOrgEmployeeOptionRes(employee, false)
	if option.Value != employee.Id || option.Label != "EMP-10 - Alice" {
		t.Fatalf("unexpected employee option: %+v", option)
	}
}

func TestOrgAssignmentResponseUsesSafeWhitelistAndReferenceSummaries(t *testing.T) {
	positionID := 30
	assignment := model.OrgAssignment{
		Basic:            model.Basic{Id: 40, State: true},
		SourceSystemCode: "authority",
		SourceId:         "assignment-source-40",
		EmployeeId:       10,
		LegalEntityId:    20,
		OrgUnitId:        25,
		PositionId:       &positionID,
		AssignmentType:   "part_time",
		IsPrimary:        false,
		IsManager:        true,
		Status:           "enabled",
		SourceVersion:    "source-version-secret",
		SourceDeleted:    true,
		SyncStatus:       "failed",
	}
	result := NewOrgAssignmentListRes(assignment, "history")
	legal := NewOrgReferenceSummaryRes(20, "LE-20", "法人二十")
	unit := NewOrgReferenceSummaryRes(25, "OU-25", "组织二十五")
	position := NewOrgReferenceSummaryRes(30, "POS-30", "岗位三十")
	result.SetReferences(&legal, &unit, &position)

	object := marshalJSONObject(t, result)
	assertJSONKeysAbsent(
		t,
		object,
		"source_system_code",
		"source_id",
		"source_version",
		"source_deleted",
		"sync_status",
	)
	if got := int(object["employee_id"].(float64)); got != assignment.EmployeeId {
		t.Fatalf("employee_id=%d, want %d", got, assignment.EmployeeId)
	}
	if object["time_scope"] != "history" {
		t.Fatalf("time_scope=%v", object["time_scope"])
	}
	if object["org_unit"].(map[string]interface{})["name"] != unit.Name ||
		object["position"].(map[string]interface{})["name"] != position.Name {
		t.Fatalf("assignment reference summaries=%+v", object)
	}
}

func TestOrgStructureNodeResponseOmitsInternalMaterializedPath(t *testing.T) {
	node := model.OrgStructureNode{
		Basic:            model.Basic{Id: 12, State: true},
		StructureId:      1,
		OrgUnitId:        20,
		SourceSystemCode: "authority",
		SourceId:         "node-source-id",
		SourceParentId:   "parent-source-id",
		Path:             "/1/12/",
		Level:            2,
		Sort:             3,
		Status:           "enabled",
		SyncStatus:       "synced",
	}

	object := marshalJSONObject(t, NewOrgStructureNodeDetailRes(node))
	assertJSONKeysAbsent(t, object,
		"path", "source_system_code", "source_id", "source_parent_id", "sync_status",
	)
	if got := int(object["org_unit_id"].(float64)); got != node.OrgUnitId {
		t.Fatalf("expected org_unit_id %d, got %d", node.OrgUnitId, got)
	}
	if got := int(object["level"].(float64)); got != node.Level {
		t.Fatalf("expected safe display level %d, got %d", node.Level, got)
	}
}

func TestOrgSyncResponsesSeparateDefaultAndPrivilegedErrorDetails(t *testing.T) {
	batch := model.OrgSyncBatch{
		Basic:        model.Basic{Id: 1, State: true},
		BatchNo:      "BATCH-1",
		SyncType:     "full",
		ObjectScope:  "all",
		Status:       "failed",
		ErrorSummary: "upstream payload validation failed",
	}
	record := model.OrgSyncRecord{
		Basic:          model.Basic{Id: 2, State: true},
		BatchId:        1,
		ObjectType:     "employee",
		SourceId:       "sensitive-source-id",
		SourceCode:     "EMP-1",
		Action:         "insert",
		Status:         "failed",
		ErrorCode:      "org_employee_missing",
		ErrorMessage:   "employee dependency was not found",
		DependencyType: "employee",
		DependencyKey:  "sensitive-dependency-key",
		RetryCount:     1,
	}

	batchDefault := marshalJSONObject(t, NewOrgSyncBatchDetailRes(batch))
	assertJSONKeysAbsent(t, batchDefault, "error_summary")
	if batchDefault["has_error"] != true {
		t.Fatalf("expected batch has_error marker, got %v", batchDefault["has_error"])
	}
	batchError := marshalJSONObject(t, NewOrgSyncBatchErrorRes(batch))
	if batchError["error_summary"] != batch.ErrorSummary {
		t.Fatalf("privileged batch error response missing detail: %+v", batchError)
	}

	recordDefault := marshalJSONObject(t, NewOrgSyncRecordDetailRes(record))
	assertJSONKeysAbsent(t, recordDefault, "source_id", "error_message", "dependency_key")
	if recordDefault["has_error"] != true {
		t.Fatalf("expected record has_error marker, got %v", recordDefault["has_error"])
	}
	recordError := marshalJSONObject(t, NewOrgSyncRecordErrorRes(record))
	if recordError["error_message"] != record.ErrorMessage ||
		recordError["dependency_key"] != record.DependencyKey {
		t.Fatalf("privileged record error response missing detail: %+v", recordError)
	}
	assertJSONKeysAbsent(t, recordError, "source_id")
}

func TestOrganizationDefaultResponsesOmitSourceAndSyncInternals(t *testing.T) {
	legal := model.OrgLegalEntity{
		Basic:            model.Basic{Id: 1, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-1",
		SourceVersion:    "version-1",
		LastError:        "internal-error",
		SyncStatus:       "failed",
		Code:             "LE-1",
		Name:             "Legal",
		EntityType:       "legal_company",
		Status:           "enabled",
	}
	unit := model.OrgUnit{
		Basic:            model.Basic{Id: 2, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-2",
		SourceVersion:    "version-2",
		LastError:        "internal-error",
		SyncStatus:       "failed",
		Code:             "U-1",
		Name:             "Unit",
		UnitType:         "department",
		Status:           "enabled",
	}

	for name, value := range map[string]any{
		"legal_entity": NewOrgLegalEntityDetailRes(legal),
		"org_unit":     NewOrgUnitDetailRes(unit),
	} {
		t.Run(name, func(t *testing.T) {
			object := marshalJSONObject(t, value)
			assertJSONKeysAbsent(t, object,
				"source_system_code", "source_id", "source_version",
				"sync_status", "last_error",
			)
		})
	}
}

func TestOrgLegalEntityTreeAndOptionUseInternalIDWithoutSourceIdentity(t *testing.T) {
	entity := model.OrgLegalEntity{
		Basic:            model.Basic{Id: 42, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-42",
		SourceCode:       "external-code-42",
		SourceVersion:    "version-42",
		Code:             "LE-42",
		Name:             "法人四十二",
		EntityType:       "legal_company",
		Status:           "enabled",
	}

	tree := marshalJSONObject(t, NewOrgLegalEntityTreeNodeRes(entity, false))
	option := marshalJSONObject(t, NewOrgLegalEntityOptionRes(entity, false))
	for name, object := range map[string]map[string]any{
		"tree":   tree,
		"option": option,
	} {
		t.Run(name, func(t *testing.T) {
			assertJSONKeysAbsent(
				t,
				object,
				"source_system_code",
				"source_id",
				"source_code",
				"source_version",
			)
			if got := int(object["value"].(float64)); got != entity.Id {
				t.Fatalf("value = %d, want legal_entity_id %d", got, entity.Id)
			}
			if object["label"] != "LE-42 - 法人四十二" {
				t.Fatalf("unexpected selector label: %v", object["label"])
			}
		})
	}
}

func TestOrgStructureTreeKeepsNodeAndBusinessIdentitySeparate(t *testing.T) {
	node := model.OrgStructureNode{
		Basic:            model.Basic{Id: 51, State: true},
		StructureId:      11,
		OrgUnitId:        21,
		SourceSystemCode: "authority",
		SourceId:         "source-node-51",
		Path:             "/51/",
		Level:            2,
		Sort:             3,
		Status:           "enabled",
		SyncStatus:       "synced",
	}
	unit := model.OrgUnit{
		Basic:            model.Basic{Id: 21, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-unit-21",
		Code:             "OU-21",
		Name:             "运营中心",
		UnitType:         "center",
		Status:           "enabled",
		SyncStatus:       "synced",
	}

	object := marshalJSONObject(t, NewOrgStructureOrgTreeNodeRes(node, unit, false))
	if got := int(object["structure_node_id"].(float64)); got != node.Id {
		t.Fatalf("structure_node_id = %d, want %d", got, node.Id)
	}
	if got := int(object["org_unit_id"].(float64)); got != unit.Id {
		t.Fatalf("org_unit_id = %d, want %d", got, unit.Id)
	}
	assertJSONKeysAbsent(
		t,
		object,
		"value",
		"path",
		"source_id",
		"source_version",
		"sync_status",
	)
}

func marshalJSONObject(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return object
}

func assertJSONKeysAbsent(t *testing.T, object map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, exists := object[key]; exists {
			t.Fatalf("response unexpectedly contains %q: %+v", key, object)
		}
	}
}
