package service

import (
	"backend/internal/database"
	"backend/internal/organization/hrsync"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"backend/internal/integration"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestOrganizationHRConsumersMapLegalAndIndependentStructures(t *testing.T) {
	service, db := newOrganizationHRSyncTestService(t)
	contract, err := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	legal := hrsync.NewLegalEntityConsumer(service, contract)
	seedOrganizationHRSyncContext(t, db, "EXEC-LEGAL", "SYNC-LEGAL", "task_legal", hrsync.ConsumerCodeLegalEntity, 1)
	result := consumeOrganizationHRBody(t, legal, "EXEC-LEGAL", "SYNC-LEGAL", "task_legal", start, end,
		`{"success":true,"data":[{"id":"source-row-1","zjkid_ignore":"legal-1","pk_corp":"LEGAL-001","name":"法人甲","isenable":1,"changeTime":"2026-08-12T10:10:00"}]}`)
	if !result.Success() || result.BusinessSuccessCount() != 1 {
		t.Fatalf("legal result=%+v", result)
	}
	var legalEntity model.OrgLegalEntity
	result = consumeOrganizationHRBody(t, legal, "EXEC-LEGAL", "SYNC-LEGAL", "task_legal", start, end,
		`{"success":true,"data":[{"id":"source-row-1","zjkid_ignore":"legal-1","pk_corp":"LEGAL-001","name":"法人甲更新","isenable":0,"changeTime":"2026-08-12T10:30:00"}]}`)
	if !result.Success() {
		t.Fatalf("legal inactive update=%+v", result)
	}
	if err := db.Where("code = ?", "LEGAL-001").First(&legalEntity).Error; err != nil || legalEntity.Status != "disabled" || legalEntity.SourceDeleted {
		t.Fatalf("inactive legal entity=%+v err=%v", legalEntity, err)
	}
	seedOrganizationHRSyncContext(t, db, "EXEC-LEGAL-CONFLICT", "SYNC-LEGAL-CONFLICT", "task_legal", hrsync.ConsumerCodeLegalEntity, 1)
	result = consumeOrganizationHRBody(t, legal, "EXEC-LEGAL-CONFLICT", "SYNC-LEGAL-CONFLICT", "task_legal", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"legal-2","pk_corp":"LEGAL-001","name":"冲突法人","isenable":1,"changeTime":"2026-08-12T10:31:00"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonSourceIDConflict) {
		t.Fatalf("legal identity conflict=%+v", result)
	}
	if err := db.Where("code = ?", "LEGAL-001").First(&legalEntity).Error; err != nil || legalEntity.SourceId != "legal-1" || legalEntity.EntityType != "legal_company" {
		t.Fatalf("legal entity=%+v err=%v", legalEntity, err)
	}
	var legalUnitCount int64
	if err := db.Model(&model.OrgUnit{}).Where("code = ?", "LEGAL-001").Count(&legalUnitCount).Error; err != nil || legalUnitCount != 0 {
		t.Fatalf("legal company leaked into org_unit: count=%d err=%v", legalUnitCount, err)
	}

	management := hrsync.NewManagementCompanyConsumer(service, contract)
	seedOrganizationHRSyncContext(t, db, "EXEC-MGMT", "SYNC-MGMT", "task_management", hrsync.ConsumerCodeManagementCompany, 1)
	result = consumeOrganizationHRBody(t, management, "EXEC-MGMT", "SYNC-MGMT", "task_management", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"company-1","pk_corp":"ORG-001","name":"管理公司","isenable":1,"changeTime":"2026-08-12T10:11:00"}]}`)
	if !result.Success() {
		t.Fatalf("management result=%+v", result)
	}
	var unit model.OrgUnit
	if err := db.Where("code = ?", "ORG-001").First(&unit).Error; err != nil || unit.UnitType != "business_unit" {
		t.Fatalf("management unit=%+v err=%v", unit, err)
	}
	assertOrganizationStructureNode(t, db, "hr_management", unit.Id, nil, "enabled")

	legalDepartment := hrsync.NewLegalDepartmentConsumer(service, contract)
	seedOrganizationHRSyncContext(t, db, "EXEC-LEGAL-DEPT", "SYNC-LEGAL-DEPT", "task_legal_department", hrsync.ConsumerCodeLegalDepartment, 1)
	result = consumeOrganizationHRBody(t, legalDepartment, "EXEC-LEGAL-DEPT", "SYNC-LEGAL-DEPT", "task_legal_department", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"dept-1","code":"DEPT-SAME","name":"同名部门","pk_fathedeptzjkid_ignore":"","orgidzjkid_ignore":"legal-1","isenable":1,"changeTime":"2026-08-12T10:12:00"}]}`)
	if !result.Success() {
		t.Fatalf("legal department result=%+v", result)
	}
	var legalUnit model.OrgUnit
	if err := db.Where("source_id = ?", "legal_unit:dept-1").First(&legalUnit).Error; err != nil || legalUnit.PrimaryLegalEntityId == nil || *legalUnit.PrimaryLegalEntityId != legalEntity.Id {
		t.Fatalf("legal unit=%+v err=%v", legalUnit, err)
	}
	assertOrganizationStructureNode(t, db, "hr_legal", legalUnit.Id, nil, "enabled")
	var managementNodeCount int64
	if err := db.Model(&model.OrgStructureNode{}).Joins("JOIN org_structures ON org_structures.id = org_structure_nodes.structure_id").Where("org_structures.code = ? AND org_structure_nodes.org_unit_id = ?", "hr_management", legalUnit.Id).Count(&managementNodeCount).Error; err != nil || managementNodeCount != 0 {
		t.Fatalf("legal unit mixed into management tree: count=%d err=%v", managementNodeCount, err)
	}
	managementDepartment := hrsync.NewManagementDepartmentConsumer(service, contract)
	seedOrganizationHRSyncContext(t, db, "EXEC-CROSS", "SYNC-CROSS", "task_management_department", hrsync.ConsumerCodeManagementDepartment, 1)
	result = consumeOrganizationHRBody(t, managementDepartment, "EXEC-CROSS", "SYNC-CROSS", "task_management_department", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"management-cross","code":"MGMT-CROSS","name":"同名部门","pk_fathedeptzjkid_ignore":"dept-1","isenable":1,"changeTime":"2026-08-12T10:40:00"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonParentInvalid) {
		t.Fatalf("cross-structure parent result=%+v", result)
	}
	var sameNameCount int64
	if err := db.Model(&model.OrgUnit{}).Where("name = ?", "同名部门").Count(&sameNameCount).Error; err != nil || sameNameCount != 2 {
		t.Fatalf("same-name units were merged: count=%d err=%v", sameNameCount, err)
	}
}

func TestOrganizationHRDepartmentRelationsDeferredCycleAndIdempotency(t *testing.T) {
	service, db := newOrganizationHRSyncTestService(t)
	contract, _ := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
	consumer := hrsync.NewManagementDepartmentConsumer(service, contract)
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	seedOrganizationHRSyncContext(t, db, "EXEC-CHILD-FIRST", "SYNC-CHILD-FIRST", "task_department", hrsync.ConsumerCodeManagementDepartment, 1)
	body := `{"success":true,"data":[` +
		`{"zjkid_ignore":"child","code":"DEPT-CHILD","name":"子部门","pk_fathedeptzjkid_ignore":"parent","isenable":1,"changeTime":"2026-08-12T10:20:00"},` +
		`{"zjkid_ignore":"parent","code":"DEPT-PARENT","name":"父部门","pk_fathedeptzjkid_ignore":"","isenable":1,"changeTime":"2026-08-12T10:19:00"}]}`
	result := consumeOrganizationHRBody(t, consumer, "EXEC-CHILD-FIRST", "SYNC-CHILD-FIRST", "task_department", start, end, body)
	if !result.Success() {
		t.Fatalf("child-first result=%+v", result)
	}
	var parent, child model.OrgUnit
	if err := db.Where("source_id = ?", "management_unit:parent").First(&parent).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("source_id = ?", "management_unit:child").First(&child).Error; err != nil {
		t.Fatal(err)
	}
	parentNode := assertOrganizationStructureNode(t, db, "hr_management", parent.Id, nil, "enabled")
	assertOrganizationStructureNode(t, db, "hr_management", child.Id, &parentNode.Id, "enabled")

	result = consumeOrganizationHRBody(t, consumer, "EXEC-CHILD-FIRST", "SYNC-CHILD-FIRST", "task_department", start, end, body)
	if !result.Success() {
		t.Fatalf("repeat result=%+v", result)
	}
	assertOrganizationDomainCounts(t, db, 2, 2, 2)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-CHILD-FIRST", "SYNC-CHILD-FIRST", "task_department", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"child","code":"DEPT-CHILD","name":"子部门","pk_fathedeptzjkid_ignore":"","isenable":1,"changeTime":"2026-08-12T10:20:00"},{"zjkid_ignore":"parent","code":"DEPT-PARENT","name":"父部门","pk_fathedeptzjkid_ignore":"","isenable":1,"changeTime":"2026-08-12T10:19:00"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonSourceIDConflict) {
		t.Fatalf("same-version parent conflict=%+v", result)
	}
	assertOrganizationStructureNode(t, db, "hr_management", child.Id, &parentNode.Id, "enabled")

	seedOrganizationHRSyncContext(t, db, "EXEC-MISSING", "SYNC-MISSING", "task_department", hrsync.ConsumerCodeManagementDepartment, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-MISSING", "SYNC-MISSING", "task_department", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"orphan","code":"DEPT-ORPHAN","name":"孤立部门","pk_fathedeptzjkid_ignore":"missing","isenable":1,"changeTime":"2026-08-12T10:21:00"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonParentUnresolved) {
		t.Fatalf("missing parent result=%+v", result)
	}
	assertOrganizationRecord(t, db, "EXEC-MISSING", "dependency_waiting", model.OrgSyncRecordActionDeferred, hrsync.ReasonParentUnresolved)

	seedOrganizationHRSyncContext(t, db, "EXEC-CYCLE", "SYNC-CYCLE", "task_department", hrsync.ConsumerCodeManagementDepartment, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-CYCLE", "SYNC-CYCLE", "task_department", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"cycle-a","code":"CYCLE-A","name":"A","pk_fathedeptzjkid_ignore":"cycle-b","isenable":1,"changeTime":"2026-08-12T10:22:00"},{"zjkid_ignore":"cycle-b","code":"CYCLE-B","name":"B","pk_fathedeptzjkid_ignore":"cycle-a","isenable":1,"changeTime":"2026-08-12T10:22:00"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonParentCycle) {
		t.Fatalf("cycle result=%+v", result)
	}
	assertOrganizationRecord(t, db, "EXEC-CYCLE", "failed", model.OrgSyncRecordActionError, hrsync.ReasonParentCycle)

	seedOrganizationHRSyncContext(t, db, "EXEC-INACTIVE-PARENT", "SYNC-INACTIVE-PARENT", "task_department", hrsync.ConsumerCodeManagementDepartment, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-INACTIVE-PARENT", "SYNC-INACTIVE-PARENT", "task_department", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"inactive-parent","code":"INACTIVE-PARENT","name":"停用父级","pk_fathedeptzjkid_ignore":"","isenable":0,"changeTime":"2026-08-12T10:24:00"},{"zjkid_ignore":"active-child","code":"ACTIVE-CHILD","name":"有效子级","pk_fathedeptzjkid_ignore":"inactive-parent","isenable":1,"changeTime":"2026-08-12T10:25:00"}]}`)
	if !result.Success() {
		t.Fatalf("inactive historical parent result=%+v", result)
	}
	var inactiveParent, activeChild model.OrgUnit
	if err := db.Where("source_id = ?", "management_unit:inactive-parent").First(&inactiveParent).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("source_id = ?", "management_unit:active-child").First(&activeChild).Error; err != nil {
		t.Fatal(err)
	}
	inactiveNode := assertOrganizationStructureNode(t, db, "hr_management", inactiveParent.Id, nil, "disabled")
	assertOrganizationStructureNode(t, db, "hr_management", activeChild.Id, &inactiveNode.Id, "enabled")

	seedOrganizationHRSyncContext(t, db, "EXEC-SELF", "SYNC-SELF", "task_department", hrsync.ConsumerCodeManagementDepartment, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-SELF", "SYNC-SELF", "task_department", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"self","code":"SELF","name":"自引用","pk_fathedeptzjkid_ignore":"self","isenable":1,"changeTime":"2026-08-12T10:26:00"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonParentSelfReference) {
		t.Fatalf("self-parent result=%+v", result)
	}
}

func TestOrganizationHRConsumerFiltersFutureAndFailsWholeSlice(t *testing.T) {
	service, db := newOrganizationHRSyncTestService(t)
	contract, _ := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
	consumer := hrsync.NewLegalEntityConsumer(service, contract)
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	seedOrganizationHRSyncContext(t, db, "EXEC-WINDOW", "SYNC-WINDOW", "task_legal", hrsync.ConsumerCodeLegalEntity, 1)
	result := consumeOrganizationHRBody(t, consumer, "EXEC-WINDOW", "SYNC-WINDOW", "task_legal", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"replay","pk_corp":"REPLAY","name":"重放","isenable":1,"changeTime":"2026-08-12T09:55:00"},{"zjkid_ignore":"future","pk_corp":"FUTURE","name":"未来","isenable":1,"changeTime":"2026-08-12T11:00:00"},{"zjkid_ignore":"","pk_corp":"BAD","name":"坏记录","isenable":1,"changeTime":"2026-08-12T10:30:00"}]}`)
	if result.Success() || result.BusinessSuccessCount() != 1 || result.BusinessFailedCount() != 1 {
		t.Fatalf("window result=%+v", result)
	}
	var futureCount int64
	if err := db.Model(&model.OrgLegalEntity{}).Where("code = ?", "FUTURE").Count(&futureCount).Error; err != nil || futureCount != 0 {
		t.Fatalf("future persisted: count=%d err=%v", futureCount, err)
	}
	var leaked int64
	if err := db.Model(&model.OrgSyncRecord{}).Where("error_message <> '' OR source_code <> ''").Count(&leaked).Error; err != nil || leaked != 0 {
		t.Fatalf("source body leaked into records: count=%d err=%v", leaked, err)
	}
}

func newOrganizationHRSyncTestService(t *testing.T) (*OrganizationHRSyncService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:org-hr-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "-"))), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.IntegrationSyncBatch{}, &model.IntegrationExecution{}, &model.OrgLegalEntity{}, &model.OrgUnit{}, &model.OrgStructure{}, &model.OrgStructureNode{}, &model.OrgSyncBatch{}, &model.OrgSyncRecord{}); err != nil {
		t.Fatal(err)
	}
	sf, err := utils.NewSnowflake(11)
	if err != nil {
		t.Fatal(err)
	}
	repository := impl.NewOrganizationHRSyncRepositoryImpl(&database.PrimaryDB{DB: db})
	return NewOrganizationHRSyncService(repository, sf), db
}

func seedOrganizationHRSyncContext(t *testing.T, db *gorm.DB, executionNo, batchNo, taskCode, consumerCode string, sliceNo int) {
	t.Helper()
	base := nextSyncTestID()
	version := 1
	slice := sliceNo
	batch := model.IntegrationSyncBatch{Basic: model.Basic{Id: base, State: true}, BatchNo: batchNo, SyncTaskID: base, TaskCode: taskCode, TaskName: taskCode, TaskVersion: version, TaskRevision: 1, SystemCode: "hr", InterfaceCode: taskCode, InterfaceVersion: 1, ConsumerCode: consumerCode, ConsumerVersion: 1, TriggerType: model.IntegrationSyncTriggerManual, TriggerKey: batchNo, Status: model.IntegrationSyncBatchStatusRunning, CheckpointMode: model.IntegrationSyncCheckpointTimestamp}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	execution := model.IntegrationExecution{Basic: model.Basic{Id: nextSyncTestID(), State: true}, ExecutionNo: executionNo, ExternalSystemID: 1, ExternalSystemCode: "hr", ExternalSystemName: "HR", InterfaceDefinitionID: 1, InterfaceCode: taskCode, InterfaceName: taskCode, InterfaceVersion: 1, TriggerSource: model.IntegrationTriggerSourceManual, Status: model.IntegrationExecutionStatusRunning, IdempotencyScope: executionNo, IdempotencyKey: executionNo, InputHash: strings.Repeat("0", 64), SyncBatchID: &batch.Id, SyncSliceNo: &slice, SyncConsumerCode: consumerCode, SyncConsumerVersion: &version, Revision: 1}
	if err := db.Create(&execution).Error; err != nil {
		t.Fatal(err)
	}
}

func consumeOrganizationHRBody(t *testing.T, consumer integration.SyncResultConsumer, executionNo, batchNo, taskCode string, start, end time.Time, body string) integration.SyncConsumptionResult {
	t.Helper()
	digest := sha256.Sum256([]byte(body))
	request, err := integration.NewSyncConsumptionRequest(integration.SyncConsumptionRequestInput{ExecutionNo: executionNo, SyncBatchNo: batchNo, TaskCode: taskCode, TaskVersion: 1, SliceNo: 1, WindowStart: &start, WindowEnd: &end, ContentType: "application/json", ResponseSize: int64(len(body)), ResponseHash: hex.EncodeToString(digest[:]), Body: []byte(body)})
	if err != nil {
		t.Fatal(err)
	}
	result, err := consumer.Consume(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertOrganizationStructureNode(t *testing.T, db *gorm.DB, structureCode string, unitID int, parentID *int, status string) model.OrgStructureNode {
	t.Helper()
	var node model.OrgStructureNode
	err := db.Joins("JOIN org_structures ON org_structures.id = org_structure_nodes.structure_id").Where("org_structures.code = ? AND org_structure_nodes.org_unit_id = ?", structureCode, unitID).First(&node).Error
	if err != nil || node.Status != status || (node.ParentNodeId == nil) != (parentID == nil) || (parentID != nil && *node.ParentNodeId != *parentID) {
		t.Fatalf("structure node=%+v parent=%v status=%s err=%v", node, parentID, status, err)
	}
	return node
}

func assertOrganizationDomainCounts(t *testing.T, db *gorm.DB, units, nodes, records int64) {
	t.Helper()
	for _, item := range []struct {
		model any
		want  int64
	}{{&model.OrgUnit{}, units}, {&model.OrgStructureNode{}, nodes}, {&model.OrgSyncRecord{}, records}} {
		var got int64
		if err := db.Model(item.model).Count(&got).Error; err != nil || got != item.want {
			t.Fatalf("model=%T count=%d want=%d err=%v", item.model, got, item.want, err)
		}
	}
}

func assertOrganizationRecord(t *testing.T, db *gorm.DB, executionNo, status, action string, reason hrsync.ReasonCode) {
	t.Helper()
	var execution model.IntegrationExecution
	if err := db.Where("execution_no = ?", executionNo).First(&execution).Error; err != nil {
		t.Fatal(err)
	}
	var record model.OrgSyncRecord
	if err := db.Where("execution_id = ? AND error_code = ?", execution.Id, reason).First(&record).Error; err != nil || record.Status != status || record.Action != action {
		t.Fatalf("record=%+v err=%v", record, err)
	}
}
