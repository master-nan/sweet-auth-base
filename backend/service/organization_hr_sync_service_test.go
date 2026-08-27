package service

import (
	"backend/internal/database"
	"backend/internal/organization/hrsync"
	testutil "backend/internal/test"
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
		`{"success":true,"data":[{"id":"source-row-1","zjkid_ignore":"legal-1","pk_corp":"LEGAL-001","name":"法人甲","fatherpkzjkid_ignore":"","isenable":1,"changeTime":"2026-08-12T10:10:00"},{"id":"source-row-2","zjkid_ignore":"legal-2","pk_corp":"LEGAL-002","name":"法人乙","fatherpkzjkid_ignore":"legal-1","isenable":1,"changeTime":"2026-08-12T10:11:00"}]}`)
	if !result.Success() || result.BusinessSuccessCount() != 2 {
		t.Fatalf("legal result=%+v", result)
	}
	var legalEntity, childLegalEntity model.OrgLegalEntity
	if err := db.Where("source_id = ?", "legal-1").First(&legalEntity).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("source_id = ?", "legal-2").First(&childLegalEntity).Error; err != nil || childLegalEntity.ParentId == nil || *childLegalEntity.ParentId != legalEntity.Id || childLegalEntity.SyncStatus != "synced" {
		t.Fatalf("child legal entity=%+v err=%v", childLegalEntity, err)
	}
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

	seedOrganizationHRSyncContext(t, db, "EXEC-LEGAL-SUPERSEDED", "SYNC-LEGAL-SUPERSEDED", "task_legal_superseded", hrsync.ConsumerCodeLegalEntity, 1)
	result = consumeOrganizationHRBody(t, legal, "EXEC-LEGAL-SUPERSEDED", "SYNC-LEGAL-SUPERSEDED", "task_legal_superseded", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"legal-old","pk_corp":"LEGAL-CURRENT","name":"旧法人","isenable":0,"changeTime":"2026-08-12T10:12:00"},{"zjkid_ignore":"legal-current","pk_corp":"LEGAL-CURRENT","name":"当前法人","isenable":1,"changeTime":"2026-08-12T10:13:00"}]}`)
	if !result.Success() || result.BusinessSuccessCount() != 1 {
		t.Fatalf("superseded legal entity result=%+v", result)
	}
	var currentLegalEntity model.OrgLegalEntity
	if err := db.Where("code = ?", "LEGAL-CURRENT").First(&currentLegalEntity).Error; err != nil || currentLegalEntity.SourceId != "legal-current" || currentLegalEntity.Status != "enabled" {
		t.Fatalf("current legal entity=%+v err=%v", currentLegalEntity, err)
	}

	management := hrsync.NewManagementCompanyConsumer(service, contract)
	seedOrganizationHRSyncContext(t, db, "EXEC-MGMT", "SYNC-MGMT", "task_management", hrsync.ConsumerCodeManagementCompany, 1)
	result = consumeOrganizationHRBody(t, management, "EXEC-MGMT", "SYNC-MGMT", "task_management", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"company-1","pk_corp":"ORG-001","name":"管理公司","isenable":1,"changeTime":"2026-08-12T10:11:00"}]}`)
	if !result.Success() {
		t.Fatalf("management result=%+v", result)
	}
	var unit model.OrgUnit
	if err := db.Where("source_id = ?", "management_company:company-1").First(&unit).Error; err != nil || unit.SourceCode != "ORG-001" || unit.UnitType != "business_unit" {
		t.Fatalf("management unit=%+v err=%v", unit, err)
	}
	assertOrganizationStructureNode(t, db, "hr_management", unit.Id, nil, "enabled")

	legalDepartment := hrsync.NewLegalDepartmentConsumer(service, contract)
	seedOrganizationHRSyncContext(t, db, "EXEC-LEGAL-DEPT", "SYNC-LEGAL-DEPT", "task_legal_department", hrsync.ConsumerCodeLegalDepartment, 1)
	result = consumeOrganizationHRBody(t, legalDepartment, "EXEC-LEGAL-DEPT", "SYNC-LEGAL-DEPT", "task_legal_department", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"dept-1","code":"DEPT-SAME","name":"同名部门","pk_fathedeptzjkid_ignore":"legal-1","orgidzjkid_ignore":"legal-1","isenable":1,"changeTime":"2026-08-12T10:12:00"},{"zjkid_ignore":"dept-2","code":"DEPT-SAME","name":"另一法人同编码部门","pk_fathedeptzjkid_ignore":"legal-2","orgidzjkid_ignore":"legal-2","isenable":1,"changeTime":"2026-08-12T10:13:00"}]}`)
	if !result.Success() || result.BusinessSuccessCount() != 2 {
		t.Fatalf("legal department result=%+v", result)
	}
	var legalUnit model.OrgUnit
	if err := db.Where("source_id = ?", "legal_unit:dept-1").First(&legalUnit).Error; err != nil || legalUnit.PrimaryLegalEntityId == nil || *legalUnit.PrimaryLegalEntityId != legalEntity.Id {
		t.Fatalf("legal unit=%+v err=%v", legalUnit, err)
	}
	var duplicateCodeCount int64
	if err := db.Model(&model.OrgUnit{}).Where("source_code = ?", "DEPT-SAME").Count(&duplicateCodeCount).Error; err != nil || duplicateCodeCount != 2 {
		t.Fatalf("same source code departments: count=%d err=%v", duplicateCodeCount, err)
	}
	assertOrganizationStructureNode(t, db, "hr_legal", legalUnit.Id, nil, "enabled")
	seedOrganizationHRSyncContext(t, db, "EXEC-LEGAL-DEPT-OLD-REF", "SYNC-LEGAL-DEPT-OLD-REF", "task_legal_department_old_ref", hrsync.ConsumerCodeLegalDepartment, 1)
	result = consumeOrganizationHRBody(t, legalDepartment, "EXEC-LEGAL-DEPT-OLD-REF", "SYNC-LEGAL-DEPT-OLD-REF", "task_legal_department_old_ref", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"dept-old-company-ref","code":"DEPT-OLD-REF","name":"历史引用部门","pk_fathedeptzjkid_ignore":"legal-old","orgidzjkid_ignore":"legal-old","pk_corp":"LEGAL-CURRENT","isenable":1,"changeTime":"2026-08-12T10:14:00"}]}`)
	if !result.Success() {
		t.Fatalf("legal department old company reference result=%+v", result)
	}
	var oldReferenceUnit model.OrgUnit
	if err := db.Where("source_id = ?", "legal_unit:dept-old-company-ref").First(&oldReferenceUnit).Error; err != nil || oldReferenceUnit.PrimaryLegalEntityId == nil || *oldReferenceUnit.PrimaryLegalEntityId != currentLegalEntity.Id {
		t.Fatalf("legal department old company reference=%+v err=%v", oldReferenceUnit, err)
	}
	var managementNodeCount int64
	if err := db.Model(&model.OrgStructureNode{}).Joins("JOIN org_structures ON org_structures.id = org_structure_nodes.structure_id").Where("org_structures.code = ? AND org_structure_nodes.org_unit_id = ?", "hr_management", legalUnit.Id).Count(&managementNodeCount).Error; err != nil || managementNodeCount != 0 {
		t.Fatalf("legal unit mixed into management tree: count=%d err=%v", managementNodeCount, err)
	}
	managementDepartment := hrsync.NewManagementDepartmentConsumer(service, contract)
	seedOrganizationHRSyncContext(t, db, "EXEC-CROSS", "SYNC-CROSS", "task_management_department", hrsync.ConsumerCodeManagementDepartment, 1)
	result = consumeOrganizationHRBody(t, managementDepartment, "EXEC-CROSS", "SYNC-CROSS", "task_management_department", start, end,
		`{"success":true,"data":[{"zjkid_ignore":"management-cross","code":"MGMT-CROSS","name":"同名部门","pk_fathedeptzjkid_ignore":"dept-1","isenable":1,"changeTime":"2026-08-12T10:40:00"}]}`)
	if !result.Success() {
		t.Fatalf("cross-structure parent result=%+v", result)
	}
	var managementCross model.OrgUnit
	if err := db.Where("source_id = ?", "management_unit:management-cross").First(&managementCross).Error; err != nil {
		t.Fatal(err)
	}
	managementLegalBridge := assertOrganizationStructureNode(t, db, "hr_management", legalUnit.Id, nil, "enabled")
	assertOrganizationStructureNode(t, db, "hr_management", managementCross.Id, &managementLegalBridge.Id, "enabled")
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

func TestOrganizationHRPositionConsumerIdentityReferenceStateAndIdempotency(t *testing.T) {
	service, db := newOrganizationHRSyncTestService(t)
	contract, _ := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
	consumer := hrsync.NewPositionConsumer(service, contract)
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	units := []model.OrgUnit{
		organizationHRPositionUnit(nextSyncTestID(), "dept-a", "DEPT-A"),
		organizationHRPositionUnit(nextSyncTestID(), "dept-b", "DEPT-B"),
	}
	if err := db.Create(&units).Error; err != nil {
		t.Fatal(err)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-POSITION", "SYNC-POSITION", "task_position", hrsync.ConsumerCodePosition, 1)
	body := `{"success":true,"data":[` +
		`{"postidzjkid_ignore":"position-a","postCode":"POST-A","postname":"同名岗位","deptidzjkid_ignore":"dept-a","posLevel":"","isenable":1,"changeTime":"2026-08-12T10:10:00"},` +
		`{"postidzjkid_ignore":"position-b","postCode":"POST-B","postname":"同名岗位","deptidzjkid_ignore":"dept-b","posLevel":"L2","isenable":1,"changeTime":"2026-08-12T10:11:00"},` +
		`{"postidzjkid_ignore":"position-a","postCode":"POST-A","postname":"同名岗位","deptidzjkid_ignore":"dept-a","posLevel":"","isenable":1,"changeTime":"2026-08-12T10:10:00"}]}`
	result := consumeOrganizationHRBody(t, consumer, "EXEC-POSITION", "SYNC-POSITION", "task_position", start, end, body)
	if !result.Success() || result.BusinessSuccessCount() != 2 {
		t.Fatalf("position create=%+v", result)
	}
	var positions []model.OrgPosition
	if err := db.Order("source_id ASC").Find(&positions).Error; err != nil || len(positions) != 2 {
		t.Fatalf("positions=%+v err=%v", positions, err)
	}
	if positions[0].Name != positions[1].Name || positions[0].OrgUnitId == positions[1].OrgUnitId ||
		positions[0].PositionType != "professional" || positions[0].IsManagerPosition || positions[1].JobLevel != "L2" {
		t.Fatalf("same-name position mapping=%+v", positions)
	}
	assertOrganizationPositionRecordAction(t, db, "position-a", model.OrgSyncRecordActionCreate)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-POSITION", "SYNC-POSITION", "task_position", start, end, body)
	if !result.Success() {
		t.Fatalf("position replay=%+v", result)
	}
	assertOrganizationPositionRecordAction(t, db, "position-a", model.OrgSyncRecordActionNoop)
	assertOrganizationPositionCounts(t, db, 2, 2)

	result = consumeOrganizationHRBody(t, consumer, "EXEC-POSITION", "SYNC-POSITION", "task_position", start, end,
		`{"success":true,"data":[{"postidzjkid_ignore":"position-a","postCode":"POST-A","postname":"岗位更新","deptidzjkid_ignore":"dept-a","isenable":1,"changeTime":"2026-08-12T10:30:00"}]}`)
	if !result.Success() {
		t.Fatalf("position update=%+v", result)
	}
	assertOrganizationPositionRecordAction(t, db, "position-a", model.OrgSyncRecordActionUpdate)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-POSITION", "SYNC-POSITION", "task_position", start, end,
		`{"success":true,"data":[{"postidzjkid_ignore":"position-a","postCode":"POST-A","postname":"陈旧名称","deptidzjkid_ignore":"dept-a","isenable":1,"changeTime":"2026-08-12T10:20:00"}]}`)
	if !result.Success() {
		t.Fatalf("stale position=%+v", result)
	}
	var positionA model.OrgPosition
	if err := db.Where("source_id = ?", "position-a").First(&positionA).Error; err != nil || positionA.Name != "岗位更新" {
		t.Fatalf("stale overwrite position=%+v err=%v", positionA, err)
	}
	result = consumeOrganizationHRBody(t, consumer, "EXEC-POSITION", "SYNC-POSITION", "task_position", start, end,
		`{"success":true,"data":[{"postidzjkid_ignore":"position-a","postCode":"POST-A","postname":"岗位更新","deptidzjkid_ignore":"dept-a","isenable":0,"changeTime":"2026-08-12T10:40:00"}]}`)
	if !result.Success() {
		t.Fatalf("position disable=%+v", result)
	}
	assertOrganizationPositionRecordAction(t, db, "position-a", model.OrgSyncRecordActionDisable)
	if err := db.Where("source_id = ?", "position-a").First(&positionA).Error; err != nil || positionA.Status != "disabled" || positionA.SourceDeleted || positionA.IsManagerPosition {
		t.Fatalf("disabled position=%+v err=%v", positionA, err)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-POSITION-CODE", "SYNC-POSITION-CODE", "task_position", hrsync.ConsumerCodePosition, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-POSITION-CODE", "SYNC-POSITION-CODE", "task_position", start, end,
		`{"success":true,"data":[{"postidzjkid_ignore":"position-code-conflict","postCode":"POST-B","postname":"编码冲突","deptidzjkid_ignore":"dept-a","isenable":1,"changeTime":"2026-08-12T10:41:00"}]}`)
	if !result.Success() {
		t.Fatalf("same source code position=%+v", result)
	}
	var duplicatePositionCodeCount int64
	if err := db.Model(&model.OrgPosition{}).Where("source_code = ?", "POST-B").Count(&duplicatePositionCodeCount).Error; err != nil || duplicatePositionCodeCount != 2 {
		t.Fatalf("same source code positions: count=%d err=%v", duplicatePositionCodeCount, err)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-POSITION-MISSING", "SYNC-POSITION-MISSING", "task_position", hrsync.ConsumerCodePosition, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-POSITION-MISSING", "SYNC-POSITION-MISSING", "task_position", start, end,
		`{"success":true,"data":[{"postidzjkid_ignore":"position-missing","postCode":"POST-MISSING","postname":"缺组织岗位","deptidzjkid_ignore":"missing-unit","isenable":1,"changeTime":"2026-08-12T10:42:00"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonReferenceMissing) {
		t.Fatalf("position missing reference=%+v", result)
	}
	assertOrganizationRecord(t, db, "EXEC-POSITION-MISSING", "dependency_waiting", model.OrgSyncRecordActionDeferred, hrsync.ReasonReferenceMissing)
	var missingPosition int64
	if err := db.Model(&model.OrgPosition{}).Where("source_id = ?", "position-missing").Count(&missingPosition).Error; err != nil || missingPosition != 0 {
		t.Fatalf("missing-reference position count=%d err=%v", missingPosition, err)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-POSITION-INVALID", "SYNC-POSITION-INVALID", "task_position", hrsync.ConsumerCodePosition, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-POSITION-INVALID", "SYNC-POSITION-INVALID", "task_position", start, end,
		`{"success":true,"data":[{"postidzjkid_ignore":"","postCode":"POST-NO-ID","postname":"缺身份","deptidzjkid_ignore":"dept-a","isenable":1,"changeTime":"2026-08-12T10:43:00"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonSourceIDMissing) {
		t.Fatalf("position missing source id=%+v", result)
	}
	seedOrganizationHRSyncContext(t, db, "EXEC-POSITION-ENUM", "SYNC-POSITION-ENUM", "task_position", hrsync.ConsumerCodePosition, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-POSITION-ENUM", "SYNC-POSITION-ENUM", "task_position", start, end,
		`{"success":true,"data":[{"postidzjkid_ignore":"position-enum","postCode":"POST-ENUM","postname":"枚举异常","deptidzjkid_ignore":"dept-a","isenable":9,"changeTime":"2026-08-12T10:43:00"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonEnumUnknown) {
		t.Fatalf("position enum=%+v", result)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-POSITION-WINDOW", "SYNC-POSITION-WINDOW", "task_position", hrsync.ConsumerCodePosition, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-POSITION-WINDOW", "SYNC-POSITION-WINDOW", "task_position", start, end,
		`{"success":true,"data":[{"postidzjkid_ignore":"position-future","postCode":"POST-FUTURE","postname":"未来岗位","deptidzjkid_ignore":"dept-a","isenable":1,"changeTime":"2026-08-12T11:00:00"}]}`)
	if !result.Success() {
		t.Fatalf("future-only position=%+v", result)
	}
	var futurePosition int64
	if err := db.Model(&model.OrgPosition{}).Where("source_id = ?", "position-future").Count(&futurePosition).Error; err != nil || futurePosition != 0 {
		t.Fatalf("future position count=%d err=%v", futurePosition, err)
	}
}

func TestOrganizationHREmployeeConsumerIdentityStateStaleAndBoundaries(t *testing.T) {
	service, db := newOrganizationHRSyncTestService(t)
	contract, _ := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
	consumer := hrsync.NewEmployeeConsumer(service, contract)
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	seedOrganizationHRSyncContext(t, db, "EXEC-EMPLOYEE", "SYNC-EMPLOYEE", "task_employee", hrsync.ConsumerCodeEmployee, 1)
	body := `{"success":true,"data":[` +
		`{"psnidzjkid_ignore":"employee-a","jhcode":"EMP-001","name":"员工甲","mobile":null,"email":"","isenable":1,"changeTime":"2026-08-12T10:10:00","sendpost":"[]","deptidzjkid_ignore":"ignored-dept","postidzjkid_ignore":"ignored-position"},` +
		`{"psnidzjkid_ignore":"employee-a","jhcode":"EMP-001","name":"员工甲","mobile":null,"email":"","isenable":1,"changeTime":"2026-08-12T10:10:00","sendpost":"[]"},` +
		`{"psnidzjkid_ignore":"employee-future","jhcode":"EMP-FUTURE","name":"未来员工","isenable":1,"changeTime":"2026-08-12T11:00:00","sendpost":"[]"}]}`
	result := consumeOrganizationHRBody(t, consumer, "EXEC-EMPLOYEE", "SYNC-EMPLOYEE", "task_employee", start, end, body)
	if !result.Success() || result.BusinessSuccessCount() != 1 {
		t.Fatalf("employee create=%+v", result)
	}
	var employee model.OrgEmployee
	if err := db.Where("source_id = ?", "employee-a").First(&employee).Error; err != nil || employee.EmployeeNo != "EMP-001" || employee.Name != "员工甲" || employee.Mobile != "" || employee.Email != "" || employee.EmploymentStatus != "active" || employee.SourceDeleted || employee.PrimaryLegalEntityId != nil || employee.UserId != nil {
		t.Fatalf("employee=%+v err=%v", employee, err)
	}
	var future, assignments int64
	if err := db.Model(&model.OrgEmployee{}).Where("source_id = ?", "employee-future").Count(&future).Error; err != nil || future != 0 {
		t.Fatalf("future employee=%d err=%v", future, err)
	}
	if err := db.Model(&model.OrgAssignment{}).Count(&assignments).Error; err != nil || assignments != 0 {
		t.Fatalf("assignments=%d err=%v", assignments, err)
	}

	result = consumeOrganizationHRBody(t, consumer, "EXEC-EMPLOYEE", "SYNC-EMPLOYEE", "task_employee", start, end, body)
	if !result.Success() {
		t.Fatalf("employee repeat=%+v", result)
	}
	assertOrganizationEmployeeRecordAction(t, db, "employee-a", model.OrgSyncRecordActionNoop)
	boundUserID := nextSyncTestID()
	if err := db.Model(&model.OrgEmployee{}).Where("id = ?", employee.Id).Update("user_id", boundUserID).Error; err != nil {
		t.Fatal(err)
	}
	result = consumeOrganizationHRBody(t, consumer, "EXEC-EMPLOYEE", "SYNC-EMPLOYEE", "task_employee", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"employee-a","jhcode":"EMP-001","name":"员工甲更新","mobile":"","email":"employee@example.invalid","isenable":2,"changeTime":"2026-08-12T10:30:00","sendpost":"[]"}]}`)
	if !result.Success() {
		t.Fatalf("employee update=%+v", result)
	}
	if err := db.First(&employee, employee.Id).Error; err != nil || employee.Name != "员工甲更新" || employee.Email != "employee@example.invalid" || employee.EmploymentStatus != "suspended" || employee.EmploymentStatus == "resigned" || employee.UserId == nil || *employee.UserId != boundUserID {
		t.Fatalf("updated employee=%+v err=%v", employee, err)
	}
	assertOrganizationEmployeeRecordAction(t, db, "employee-a", model.OrgSyncRecordActionUpdate)

	result = consumeOrganizationHRBody(t, consumer, "EXEC-EMPLOYEE", "SYNC-EMPLOYEE", "task_employee", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"employee-a","jhcode":"EMP-001","name":"陈旧员工","mobile":"secret","email":"stale@example.invalid","isenable":1,"changeTime":"2026-08-12T10:20:00","sendpost":"[]"}]}`)
	if !result.Success() {
		t.Fatalf("stale employee=%+v", result)
	}
	if err := db.First(&employee, employee.Id).Error; err != nil || employee.Name != "员工甲更新" || employee.EmploymentStatus != "suspended" {
		t.Fatalf("stale overwrite=%+v err=%v", employee, err)
	}

	result = consumeOrganizationHRBody(t, consumer, "EXEC-EMPLOYEE", "SYNC-EMPLOYEE", "task_employee", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"employee-a","jhcode":"EMP-001","name":"同版本冲突","isenable":2,"changeTime":"2026-08-12T10:30:00","sendpost":"[]"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonSourceIDConflict) {
		t.Fatalf("same-version conflict=%+v", result)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-EMPLOYEE-NO", "SYNC-EMPLOYEE-NO", "task_employee", hrsync.ConsumerCodeEmployee, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-EMPLOYEE-NO", "SYNC-EMPLOYEE-NO", "task_employee", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"employee-b","jhcode":"EMP-001","name":"员工乙","isenable":1,"changeTime":"2026-08-12T10:40:00","sendpost":"[]"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonBusinessConflict) {
		t.Fatalf("employee number conflict=%+v", result)
	}
	var employees int64
	if err := db.Model(&model.OrgEmployee{}).Count(&employees).Error; err != nil || employees != 1 {
		t.Fatalf("employee count=%d err=%v", employees, err)
	}
	assertOrganizationEmployeeDataNotLeaked(t, db)
}

func TestOrganizationHREmployeeConsumerRejectsMissingIdentitySendpostAndOversize(t *testing.T) {
	service, db := newOrganizationHRSyncTestService(t)
	contract, _ := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
	consumer := hrsync.NewEmployeeConsumer(service, contract)
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	tests := []struct {
		execution string
		body      string
		reason    hrsync.ReasonCode
	}{
		{"EXEC-EMPLOYEE-MISSING-ID", `{"success":true,"data":[{"jhcode":"EMP-001","name":"员工","isenable":1,"changeTime":"2026-08-12T10:10:00"}]}`, hrsync.ReasonSourceIDMissing},
		{"EXEC-EMPLOYEE-MISSING-NO", `{"success":true,"data":[{"psnidzjkid_ignore":"employee-a","name":"员工","isenable":1,"changeTime":"2026-08-12T10:10:00"}]}`, hrsync.ReasonEnvelopeInvalid},
		{"EXEC-EMPLOYEE-SENDPOST", `{"success":true,"data":[{"psnidzjkid_ignore":"employee-a","jhcode":"EMP-001","name":"员工","isenable":1,"changeTime":"2026-08-12T10:10:00","sendpost":"not-json"}]}`, hrsync.ReasonEnvelopeInvalid},
	}
	for _, test := range tests {
		seedOrganizationHRSyncContext(t, db, test.execution, "SYNC-"+test.execution, "task_employee", hrsync.ConsumerCodeEmployee, 1)
		result := consumeOrganizationHRBody(t, consumer, test.execution, "SYNC-"+test.execution, "task_employee", start, end, test.body)
		if result.Success() || result.ReasonCode() != string(test.reason) {
			t.Fatalf("execution=%s result=%+v", test.execution, result)
		}
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-EMPLOYEE-OVERSIZE", "SYNC-EMPLOYEE-OVERSIZE", "task_employee", hrsync.ConsumerCodeEmployee, 1)
	body := []byte(strings.Repeat(" ", (16<<20)+1))
	digest := sha256.Sum256(body)
	request, err := integration.NewSyncConsumptionRequest(integration.SyncConsumptionRequestInput{
		ExecutionNo: "EXEC-EMPLOYEE-OVERSIZE", SyncBatchNo: "SYNC-EMPLOYEE-OVERSIZE", TaskCode: "task_employee", TaskVersion: 1,
		SliceNo: 1, WindowStart: &start, WindowEnd: &end, ContentType: "application/json", ResponseSize: int64(len(body)),
		ResponseHash: hex.EncodeToString(digest[:]), Body: body,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := consumer.Consume(context.Background(), request)
	if err != nil || result.Success() || result.ReasonCode() != string(hrsync.ReasonEnvelopeInvalid) {
		t.Fatalf("oversize result=%+v err=%v", result, err)
	}
	var employees int64
	if err := db.Model(&model.OrgEmployee{}).Count(&employees).Error; err != nil || employees != 0 {
		t.Fatalf("invalid employees=%d err=%v", employees, err)
	}
}

func TestOrganizationHRResignedEmployeeConsumerClosesAssignmentsAndProtectsEventOrder(t *testing.T) {
	service, db := newOrganizationHRSyncTestService(t)
	contract, _ := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
	consumer := hrsync.NewResignedEmployeeConsumer(service, contract)
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	userID := nextSyncTestID()
	employee := seedOrganizationHRResignationEmployee(t, db, "employee-resigned", "EMP-R-1", userID, start)
	assignment := seedOrganizationHRCurrentAssignment(t, db, employee.Id, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))

	seedOrganizationHRSyncContext(t, db, "EXEC-RESIGNED", "SYNC-RESIGNED", "task_resigned", hrsync.ConsumerCodeResignedEmployee, 1)
	body := `{"success":true,"data":[{"psnidzjkid_ignore":"employee-resigned","changeTime":"2026-08-12T10:30:00","lzdate":"2026-08-10","name":"must-not-persist"}]}`
	result := consumeOrganizationHRBody(t, consumer, "EXEC-RESIGNED", "SYNC-RESIGNED", "task_resigned", start, end, body)
	if !result.Success() || result.BusinessSuccessCount() != 2 {
		t.Fatalf("resignation result=%+v", result)
	}
	if err := db.First(&employee, employee.Id).Error; err != nil || employee.EmploymentStatus != "resigned" ||
		!organizationDatesEqual(employee.ValidTo, time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)) ||
		employee.UserId == nil || *employee.UserId != userID || employee.LocalNote != "keep-local" || employee.SourceDeleted {
		t.Fatalf("resigned employee=%+v err=%v", employee, err)
	}
	if err := db.First(&assignment, assignment.Id).Error; err != nil || assignment.Status != "disabled" ||
		!organizationDatesEqual(assignment.ValidTo, time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)) || assignment.SourceDeleted {
		t.Fatalf("closed assignment=%+v err=%v", assignment, err)
	}
	assertOrganizationResignationRecord(t, db, "EXEC-RESIGNED", hrsync.ObjectKindEmployee, model.OrgSyncRecordActionUpdate, "success")
	assertOrganizationResignationRecord(t, db, "EXEC-RESIGNED", hrsync.ObjectKindAssignment, model.OrgSyncRecordActionClose, "success")

	result = consumeOrganizationHRBody(t, consumer, "EXEC-RESIGNED", "SYNC-RESIGNED", "task_resigned", start, end, body)
	if !result.Success() || result.BusinessSuccessCount() != 2 {
		t.Fatalf("repeat resignation=%+v", result)
	}
	var recordCount int64
	if err := db.Model(&model.OrgSyncRecord{}).Joins("JOIN org_sync_batches ON org_sync_batches.id = org_sync_records.batch_id").
		Where("org_sync_batches.execution_id = ?", organizationExecutionID(t, db, "EXEC-RESIGNED")).Count(&recordCount).Error; err != nil || recordCount != 2 {
		t.Fatalf("repeat records=%d err=%v", recordCount, err)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-RESIGNED-STALE", "SYNC-RESIGNED-STALE", "task_resigned", hrsync.ConsumerCodeResignedEmployee, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-RESIGNED-STALE", "SYNC-RESIGNED-STALE", "task_resigned", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"employee-resigned","changeTime":"2026-08-12T10:20:00","lzdate":"2026-08-09"}]}`)
	if !result.Success() {
		t.Fatalf("stale resignation=%+v", result)
	}
	if err := db.First(&employee, employee.Id).Error; err != nil || !organizationDatesEqual(employee.ValidTo, time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("stale resignation overwrote employee=%+v err=%v", employee, err)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-ACTIVE-AFTER-LEAVE", "SYNC-ACTIVE-AFTER-LEAVE", "task_employee", hrsync.ConsumerCodeEmployee, 1)
	active := hrsync.NewEmployeeConsumer(service, contract)
	result = consumeOrganizationHRBody(t, active, "EXEC-ACTIVE-AFTER-LEAVE", "SYNC-ACTIVE-AFTER-LEAVE", "task_employee", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"employee-resigned","jhcode":"EMP-R-1","name":"不得再入职","isenable":1,"changeTime":"2026-08-12T10:40:00","sendpost":"[]"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonEmploymentStateConflict) {
		t.Fatalf("active after resignation=%+v", result)
	}
	if err := db.First(&employee, employee.Id).Error; err != nil || employee.EmploymentStatus != "resigned" || employee.Name != "existing" {
		t.Fatalf("employee reactivated=%+v err=%v", employee, err)
	}
}

func TestOrganizationHRResignationMissingEmployeeFutureAndAssignmentRollback(t *testing.T) {
	service, db := newOrganizationHRSyncTestService(t)
	contract, _ := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, time.UTC)
	consumer := hrsync.NewResignedEmployeeConsumer(service, contract)
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	seedOrganizationHRSyncContext(t, db, "EXEC-RESIGNED-MISSING", "SYNC-RESIGNED-MISSING", "task_resigned", hrsync.ConsumerCodeResignedEmployee, 1)
	result := consumeOrganizationHRBody(t, consumer, "EXEC-RESIGNED-MISSING", "SYNC-RESIGNED-MISSING", "task_resigned", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"missing","changeTime":"2026-08-12T10:10:00","lzdate":"2026-08-10"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonReferenceMissing) {
		t.Fatalf("missing employee=%+v", result)
	}
	assertOrganizationResignationRecord(t, db, "EXEC-RESIGNED-MISSING", hrsync.ObjectKindEmployee, model.OrgSyncRecordActionDeferred, "dependency_waiting")

	employee := seedOrganizationHRResignationEmployee(t, db, "employee-period", "EMP-R-2", 0, start)
	assignment := seedOrganizationHRCurrentAssignment(t, db, employee.Id, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC))
	seedOrganizationHRSyncContext(t, db, "EXEC-RESIGNED-PERIOD", "SYNC-RESIGNED-PERIOD", "task_resigned", hrsync.ConsumerCodeResignedEmployee, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-RESIGNED-PERIOD", "SYNC-RESIGNED-PERIOD", "task_resigned", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"employee-period","changeTime":"2026-08-12T10:30:00","lzdate":"2026-08-10"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonAssignmentPeriodInvalid) {
		t.Fatalf("invalid assignment period=%+v", result)
	}
	if err := db.First(&employee, employee.Id).Error; err != nil || employee.EmploymentStatus != "active" || employee.ValidTo != nil {
		t.Fatalf("employee partially resigned=%+v err=%v", employee, err)
	}
	if err := db.First(&assignment, assignment.Id).Error; err != nil || assignment.Status != "enabled" || assignment.ValidTo != nil {
		t.Fatalf("assignment partially closed=%+v err=%v", assignment, err)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-RESIGNED-FUTURE", "SYNC-RESIGNED-FUTURE", "task_resigned", hrsync.ConsumerCodeResignedEmployee, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-RESIGNED-FUTURE", "SYNC-RESIGNED-FUTURE", "task_resigned", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"employee-period","changeTime":"2026-08-12T11:00:00","lzdate":"2026-08-10"}]}`)
	if !result.Success() || result.BusinessSuccessCount() != 0 {
		t.Fatalf("future resignation=%+v", result)
	}
	if err := db.First(&employee, employee.Id).Error; err != nil || employee.EmploymentStatus != "active" {
		t.Fatalf("future resignation persisted=%+v err=%v", employee, err)
	}

	seedOrganizationHRSyncContext(t, db, "EXEC-RESIGNED-DATE", "SYNC-RESIGNED-DATE", "task_resigned", hrsync.ConsumerCodeResignedEmployee, 1)
	result = consumeOrganizationHRBody(t, consumer, "EXEC-RESIGNED-DATE", "SYNC-RESIGNED-DATE", "task_resigned", start, end,
		`{"success":true,"data":[{"psnidzjkid_ignore":"employee-period","changeTime":"2026-08-12T10:30:00","lzdate":"2026-02-30"}]}`)
	if result.Success() || result.ReasonCode() != string(hrsync.ReasonEnvelopeInvalid) {
		t.Fatalf("invalid resignation date=%+v", result)
	}
}

func newOrganizationHRSyncTestService(t *testing.T) (*OrganizationHRSyncService, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{}, &model.IntegrationSyncBatch{}, &model.IntegrationExecution{}, &model.OrgLegalEntity{}, &model.OrgUnit{}, &model.OrgStructure{}, &model.OrgStructureNode{}, &model.OrgPosition{}, &model.OrgEmployee{}, &model.OrgAssignment{}, &model.OrgSyncBatch{}, &model.OrgSyncRecord{})
	sf, err := utils.NewSnowflake(11)
	if err != nil {
		t.Fatal(err)
	}
	repository := impl.NewOrganizationHRSyncRepositoryImpl(&database.PrimaryDB{DB: db})
	return NewOrganizationHRSyncService(repository, sf), db
}

func seedOrganizationHRResignationEmployee(t *testing.T, db *gorm.DB, rawSourceID, employeeNo string, userID int, changedAt time.Time) model.OrgEmployee {
	t.Helper()
	var user *int
	if userID != 0 {
		user = &userID
	}
	employee := model.OrgEmployee{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SourceSystemCode: hrsync.OrganizationHRSourceSystemCode,
		SourceId: rawSourceID, EmployeeNo: employeeNo, Name: "existing", EmploymentStatus: "active",
		SourceVersion: changedAt.Format(time.RFC3339Nano), SourceUpdatedAt: &changedAt, LastSyncAt: &changedAt,
		SourceDeleted: false, SyncStatus: "synced", UserId: user, LocalNote: "keep-local",
	}
	if err := db.Create(&employee).Error; err != nil {
		t.Fatal(err)
	}
	return employee
}

func seedOrganizationHRCurrentAssignment(t *testing.T, db *gorm.DB, employeeID int, validFrom time.Time) model.OrgAssignment {
	t.Helper()
	legal := model.OrgLegalEntity{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SourceSystemCode: hrsync.OrganizationHRSourceSystemCode,
		SourceId: fmt.Sprintf("legal-%d", employeeID), Code: fmt.Sprintf("LEGAL-%d", employeeID), Name: "legal",
		EntityType: "legal_company", Status: "enabled", SyncStatus: "synced",
	}
	if err := db.Create(&legal).Error; err != nil {
		t.Fatal(err)
	}
	unit := model.OrgUnit{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SourceSystemCode: hrsync.OrganizationHRSourceSystemCode,
		SourceId: fmt.Sprintf("management_unit:unit-%d", employeeID), Code: fmt.Sprintf("UNIT-%d", employeeID), Name: "unit",
		UnitType: "department", Status: "enabled", SyncStatus: "synced",
	}
	if err := db.Create(&unit).Error; err != nil {
		t.Fatal(err)
	}
	assignment := model.OrgAssignment{
		Basic: model.Basic{Id: nextSyncTestID(), State: true}, SourceSystemCode: hrsync.OrganizationHRSourceSystemCode,
		SourceId: fmt.Sprintf("assignment-%d", employeeID), EmployeeId: employeeID, LegalEntityId: legal.Id, OrgUnitId: unit.Id,
		AssignmentType: "secondary", ValidFrom: &validFrom, Status: "enabled", SourceDeleted: false, SyncStatus: "synced",
	}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	return assignment
}

func organizationExecutionID(t *testing.T, db *gorm.DB, executionNo string) int {
	t.Helper()
	var execution model.IntegrationExecution
	if err := db.Where("execution_no = ?", executionNo).First(&execution).Error; err != nil {
		t.Fatal(err)
	}
	return execution.Id
}

func assertOrganizationResignationRecord(t *testing.T, db *gorm.DB, executionNo string, kind hrsync.ObjectKind, action, status string) {
	t.Helper()
	var record model.OrgSyncRecord
	err := db.Joins("JOIN org_sync_batches ON org_sync_batches.id = org_sync_records.batch_id").
		Where("org_sync_batches.execution_id = ? AND org_sync_records.object_type = ?", organizationExecutionID(t, db, executionNo), kind).
		First(&record).Error
	if err != nil || record.Action != action || record.Status != status || record.ErrorMessage != "" || record.SourceCode != "" {
		t.Fatalf("resignation record=%+v action=%s status=%s err=%v", record, action, status, err)
	}
}

func assertOrganizationEmployeeRecordAction(t *testing.T, db *gorm.DB, rawSourceID, action string) {
	t.Helper()
	key, err := hrsync.NewSourceKey(hrsync.OrganizationHRSourceSystemCode, hrsync.ObjectKindEmployee, rawSourceID)
	if err != nil {
		t.Fatal(err)
	}
	var record model.OrgSyncRecord
	if err := db.Where("object_type = ? AND source_id = ?", hrsync.ObjectKindEmployee, key.Digest()).First(&record).Error; err != nil || record.Action != action {
		t.Fatalf("employee record=%+v action=%s err=%v", record, action, err)
	}
}

func assertOrganizationEmployeeDataNotLeaked(t *testing.T, db *gorm.DB) {
	t.Helper()
	var records []model.OrgSyncRecord
	if err := db.Find(&records).Error; err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		serialized := fmt.Sprintf("%+v", record)
		for _, forbidden := range []string{"员工甲", "employee@example.invalid", "secret"} {
			if strings.Contains(serialized, forbidden) {
				t.Fatalf("employee source fact leaked into record: %+v", record)
			}
		}
	}
}

func organizationHRPositionUnit(id int, rawSourceID, code string) model.OrgUnit {
	changedAt := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	return model.OrgUnit{
		Basic: model.Basic{Id: id, State: true}, SourceSystemCode: hrsync.OrganizationHRSourceSystemCode,
		SourceId: "management_unit:" + rawSourceID, Code: code, Name: code, UnitType: "department",
		Status: "enabled", SourceUpdatedAt: &changedAt, SourceVersion: changedAt.Format(time.RFC3339Nano),
		LastSyncAt: &changedAt, SourceStatus: "enabled", SourceDeleted: false, SyncStatus: "synced",
	}
}

func assertOrganizationPositionCounts(t *testing.T, db *gorm.DB, positions, records int64) {
	t.Helper()
	for _, item := range []struct {
		model any
		want  int64
	}{{&model.OrgPosition{}, positions}, {&model.OrgSyncRecord{}, records}} {
		var got int64
		if err := db.Model(item.model).Count(&got).Error; err != nil || got != item.want {
			t.Fatalf("model=%T count=%d want=%d err=%v", item.model, got, item.want, err)
		}
	}
}

func assertOrganizationPositionRecordAction(t *testing.T, db *gorm.DB, rawSourceID, action string) {
	t.Helper()
	key, err := hrsync.NewSourceKey(hrsync.OrganizationHRSourceSystemCode, hrsync.ObjectKindPosition, rawSourceID)
	if err != nil {
		t.Fatal(err)
	}
	var record model.OrgSyncRecord
	if err := db.Where("object_type = ? AND source_id = ?", hrsync.ObjectKindPosition, key.Digest()).First(&record).Error; err != nil || record.Action != action {
		t.Fatalf("position record=%+v action=%s err=%v", record, action, err)
	}
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
