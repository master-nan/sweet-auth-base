package hrsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"backend/internal/integration"
	"backend/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSourceKeyIsIsolatedAndRedacted(t *testing.T) {
	legal, err := NewSourceKey("hr_source", ObjectKindLegalEntity, "sensitive-id-1")
	if err != nil {
		t.Fatal(err)
	}
	unit, _ := NewSourceKey("hr_source", ObjectKindLegalUnit, "sensitive-id-1")
	if legal.Digest() == unit.Digest() {
		t.Fatal("object kind was not included in source identity")
	}
	if strings.Contains(fmt.Sprintf("%s %#v", legal, legal), "sensitive-id-1") {
		t.Fatal("raw source ID leaked through formatting")
	}
	for _, raw := range []string{"", strings.Repeat("x", MaxRawSourceIDLength+1)} {
		if _, err := NewSourceKey("hr_source", ObjectKindEmployee, raw); !errors.Is(err, ErrSourceKeyInvalid) {
			t.Fatalf("raw=%q err=%v", raw, err)
		}
	}
}

func TestSourceDTOAndNormalizersKeepSourceFieldsAtAdapterBoundary(t *testing.T) {
	var source HRCompanySourceDTO
	if err := json.Unmarshal([]byte(`{"zjkid_ignore":"company-1","pk_corp":"C001","name":"Example","shortname":"EX","fatherpkzjkid_ignore":"","isenable":1,"changeTime":"2026-08-12T10:30:00","managericno":"must-be-ignored"}`), &source); err != nil {
		t.Fatal(err)
	}
	normalizer := Normalizer{SourceSystemCode: "hr_source", SourceLocation: time.FixedZone("source", 8*60*60)}
	input, err := normalizer.NormalizeLegalEntitySource(source)
	if err != nil {
		t.Fatal(err)
	}
	if input.Key.RawSourceID() != "company-1" || input.Code != "C001" || input.Status != CanonicalStatusEnabled || !input.SourceChangedAt.Equal(time.Date(2026, 8, 12, 2, 30, 0, 0, time.UTC)) {
		t.Fatalf("normalized=%+v", input)
	}
	if _, err := normalizer.NormalizeAssignmentSource(HRAssignmentSourceDTO{SourceID: "assignment-1"}); !errors.Is(err, ErrSourceContractUnconfirmed) {
		t.Fatalf("assignment P0 must remain gated: %v", err)
	}
	position, err := normalizer.NormalizePositionSource(HRPositionSourceDTO{
		SourceID: "position-1", SourceCode: "POST-001", Name: "同名岗位", OrgUnitSourceID: "unit-1",
		JobLevel: "L1", Enabled: SourceEnableEnabled, ChangeTime: "2026-08-12T10:31:00",
	})
	if err != nil || position.Key.ObjectKind() != ObjectKindPosition || position.Code != "POST-001" || position.OrgUnitSourceID != "unit-1" {
		t.Fatalf("position normalized=%+v err=%v", position, err)
	}
	var invalidPosition HRPositionSourceDTO
	if err := json.Unmarshal([]byte(`{"postidzjkid_ignore":"position-2","postCode":"POST-002","postname":"岗位","isenable":"unknown"}`), &invalidPosition); !errors.Is(err, ErrSourceEnumInvalid) {
		t.Fatalf("invalid position enum err=%v", err)
	}
	employee, err := normalizer.NormalizeEmployeeSource(HREmployeeSourceDTO{
		SourceID: "employee-1", EmployeeNo: "EMP-001", Name: "员工", Mobile: "", Email: "",
		Enabled: SourceEnableDisabled, ChangeTime: "2026-08-12T10:32:00", EmbeddedAssignments: "[]",
	})
	if err != nil || employee.Key.ObjectKind() != ObjectKindEmployee || employee.EmployeeNo != "EMP-001" || employee.Mobile != "" || employee.Email != "" || employee.EmploymentStatus != "suspended" {
		t.Fatalf("employee normalized=%+v err=%v", employee, err)
	}
}

func TestEmployeeEmbeddedAssignmentsDefenseBoundaries(t *testing.T) {
	values := make([]string, maxEmbeddedAssignmentsCount)
	for index := range values {
		values[index] = `{}`
	}
	if err := validateEmbeddedAssignments("[" + strings.Join(values, ",") + "]"); err != nil {
		t.Fatalf("maximum assignment count rejected: %v", err)
	}
	values = append(values, `{}`)
	if err := validateEmbeddedAssignments("[" + strings.Join(values, ",") + "]"); !errors.Is(err, errEmbeddedAssignmentsInvalid) {
		t.Fatalf("assignment count overflow err=%v", err)
	}
	if err := validateEmbeddedAssignments(strings.Repeat(" ", maxEmbeddedAssignmentsBytes+1)); !errors.Is(err, errEmbeddedAssignmentsInvalid) {
		t.Fatalf("assignment bytes overflow err=%v", err)
	}
	if err := validateEmbeddedAssignments(`{"ID":"not-an-array"}`); !errors.Is(err, errEmbeddedAssignmentsInvalid) {
		t.Fatalf("assignment shape err=%v", err)
	}
}

func TestLogicalWindowClassifiesLookbackCurrentAndFuture(t *testing.T) {
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	tests := []struct {
		at   time.Time
		want WindowClassification
	}{
		{start.Add(-10 * time.Minute), WindowRecordReplay},
		{start, WindowRecordCurrent},
		{end.Add(-time.Second), WindowRecordCurrent},
		{end, WindowRecordFuture},
	}
	for _, test := range tests {
		got, err := ClassifySourceChangeTime(test.at, start, end)
		if err != nil || got != test.want {
			t.Fatalf("at=%s got=%s err=%v", test.at, got, err)
		}
	}
}

func TestOrganizationHRConsumerRegistrationsStayGatedUntilSourceContractIsExplicit(t *testing.T) {
	domain := &organizationSyncDomainStub{}
	disabled, err := integration.NewStaticSyncConsumerRegistry(DisabledConsumerRegistrations(domain)...)
	if err != nil {
		t.Fatal(err)
	}
	if got := disabled.ListMetadata(); len(got) != 0 {
		t.Fatalf("disabled production registrations leaked into selectable metadata: %+v", got)
	}
	for _, code := range []string{
		ConsumerCodeLegalEntity,
		ConsumerCodeManagementCompany,
		ConsumerCodeManagementDepartment,
		ConsumerCodeLegalDepartment,
		ConsumerCodePosition,
		ConsumerCodeEmployee,
		ConsumerCodeResignedEmployee,
	} {
		if _, err := disabled.Resolve(code, ConsumerVersionV1); err == nil {
			t.Fatalf("disabled consumer %s resolved", code)
		}
	}
	contract, err := NewExplicitSourceContract(OrganizationHRSourceSystemCode, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewExplicitSourceContract("client_selected_source", time.UTC); !errors.Is(err, ErrSourceContractInvalid) {
		t.Fatalf("expected fixed source system rejection, got %v", err)
	}
	registrations, err := EnabledConsumerRegistrations(domain, contract)
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := integration.NewStaticSyncConsumerRegistry(registrations...)
	if err != nil {
		t.Fatal(err)
	}
	if got := enabled.ListMetadata(); len(got) != 7 {
		t.Fatalf("enabled registrations=%d, want 7", len(got))
	}
	metadata, err := enabled.ValidateReference(integration.SyncConsumerReference{
		Code: ConsumerCodeEmployee, Version: ConsumerVersionV1, ContentType: "application/json",
		ResponseLimit: maxEmployeeResponseBytes, CheckpointMode: model.IntegrationSyncCheckpointTimestamp,
		RequestTimeout: time.Second, LeaseDuration: 180 * time.Second,
	})
	if err != nil || metadata.MaxResponseBytes != maxEmployeeResponseBytes {
		t.Fatalf("employee metadata=%+v err=%v", metadata, err)
	}
}

func TestOrganizationConsumerHarnessFiltersLogicalWindowAndPersistsNoBody(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:hrsync-harness?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.OrgSyncBatch{}, &model.OrgSyncRecord{}); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	body := []byte(`{"data":[{"zjkid_ignore":"lookback","pk_corp":"C0","name":"Replay","isenable":1,"changeTime":"2026-08-12T09:55:00"},{"zjkid_ignore":"inside","pk_corp":"C1","name":"Current","isenable":1,"changeTime":"2026-08-12T10:20:00"},{"zjkid_ignore":"future","pk_corp":"C2","name":"Future","isenable":1,"changeTime":"2026-08-12T11:00:00"}]}`)
	digest := sha256.Sum256(body)
	request, err := integration.NewSyncConsumptionRequest(integration.SyncConsumptionRequestInput{
		ExecutionNo: "EXEC-006B", SyncBatchNo: "SYNC-006B", TaskCode: "hr_legal_entity_sync", TaskVersion: 1,
		SliceNo: 1, WindowStart: &start, WindowEnd: &end, ContentType: "application/json", ResponseSize: int64(len(body)),
		ResponseHash: hex.EncodeToString(digest[:]), Body: body,
	})
	if err != nil {
		t.Fatal(err)
	}
	consumer := organizationConsumerHarness{db: db, normalizer: Normalizer{SourceSystemCode: "hr_source", SourceLocation: time.UTC}}
	result, err := consumer.Consume(context.Background(), request)
	if err != nil || !result.Success() || result.BusinessSuccessCount() != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var records []model.OrgSyncRecord
	if err := db.Order("source_id").Find(&records).Error; err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("records=%+v", records)
	}
	for _, record := range records {
		if record.SourceId == "inside" || record.SourceId == "lookback" || len(record.SourceId) != 24 {
			t.Fatalf("raw source ID was persisted: %+v", record)
		}
		if strings.Contains(record.ErrorMessage, "Current") || strings.Contains(record.ErrorMessage, "Replay") {
			t.Fatal("source body was persisted")
		}
	}
}

type organizationConsumerHarness struct {
	db         *gorm.DB
	normalizer Normalizer
}

type organizationSyncDomainStub struct{}

func (*organizationSyncDomainStub) SynchronizeLegalEntities(context.Context, BusinessSyncContext, []LegalEntitySyncInput, []SourceIssue) (BusinessSyncSummary, error) {
	return BusinessSyncSummary{}, nil
}

func (*organizationSyncDomainStub) SynchronizeOrgUnits(context.Context, BusinessSyncContext, ObjectKind, string, []OrgUnitSyncInput, []SourceIssue) (BusinessSyncSummary, error) {
	return BusinessSyncSummary{}, nil
}

func (*organizationSyncDomainStub) SynchronizePositions(context.Context, BusinessSyncContext, []PositionSyncInput, []SourceIssue) (BusinessSyncSummary, error) {
	return BusinessSyncSummary{}, nil
}

func (*organizationSyncDomainStub) SynchronizeEmployees(context.Context, BusinessSyncContext, []EmployeeSyncInput, []SourceIssue) (BusinessSyncSummary, error) {
	return BusinessSyncSummary{}, nil
}

func (*organizationSyncDomainStub) SynchronizeResignations(context.Context, BusinessSyncContext, []ResignationSyncInput, []SourceIssue) (BusinessSyncSummary, error) {
	return BusinessSyncSummary{}, nil
}

var _ integration.SyncResultConsumer = organizationConsumerHarness{}

func (h organizationConsumerHarness) Consume(_ context.Context, request integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
	var envelope struct {
		Data []HRCompanySourceDTO `json:"data"`
	}
	if err := json.Unmarshal(request.Body(), &envelope); err != nil {
		return integration.NewSyncConsumptionResult(false, string(ReasonEnvelopeInvalid), 0, 1, "")
	}
	batch := model.OrgSyncBatch{Basic: model.Basic{Id: 1, State: true}, BatchNo: request.SyncBatchNo(), SyncType: "incremental", ObjectScope: "legal_entity", Status: "processing"}
	if err := h.db.Create(&batch).Error; err != nil {
		return integration.SyncConsumptionResult{}, err
	}
	processed := 0
	for _, source := range envelope.Data {
		input, err := h.normalizer.NormalizeLegalEntitySource(source)
		if err != nil {
			return integration.NewSyncConsumptionResult(false, string(ReasonEnvelopeInvalid), processed, 1, batch.BatchNo)
		}
		classification, err := ClassifySourceChangeTime(input.SourceChangedAt, *request.WindowStart(), *request.WindowEnd())
		if err != nil {
			return integration.SyncConsumptionResult{}, err
		}
		if classification == WindowRecordFuture {
			continue
		}
		action := model.OrgSyncRecordActionCreate
		if classification == WindowRecordReplay {
			action = model.OrgSyncRecordActionNoop
		}
		record := model.OrgSyncRecord{BatchId: batch.Id, ObjectType: string(ObjectKindLegalEntity), SourceId: input.Key.Digest(), Action: action, Status: "success"}
		if err := h.db.Where("batch_id = ? AND object_type = ? AND source_id = ?", batch.Id, record.ObjectType, record.SourceId).FirstOrCreate(&record).Error; err != nil {
			return integration.SyncConsumptionResult{}, err
		}
		processed++
	}
	if err := h.db.Model(&batch).Updates(map[string]any{"status": "success", "total_count": processed, "success_count": processed}).Error; err != nil {
		return integration.SyncConsumptionResult{}, err
	}
	return integration.NewSyncConsumptionResult(true, "", processed, 0, batch.BatchNo)
}
