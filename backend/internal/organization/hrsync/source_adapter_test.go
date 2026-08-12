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
	if len(records) != 2 || records[0].SourceId != "inside" || records[1].SourceId != "lookback" {
		t.Fatalf("records=%+v", records)
	}
	for _, record := range records {
		if strings.Contains(record.ErrorMessage, "Current") || strings.Contains(record.ErrorMessage, "Replay") {
			t.Fatal("source body was persisted")
		}
	}
}

type organizationConsumerHarness struct {
	db         *gorm.DB
	normalizer Normalizer
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
		record := model.OrgSyncRecord{BatchId: batch.Id, ObjectType: string(ObjectKindLegalEntity), SourceId: input.Key.RawSourceID(), Action: action, Status: "success"}
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
