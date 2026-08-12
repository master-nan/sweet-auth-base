package hrsync

import (
	"backend/internal/integration"
	"backend/model"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

const maxOrganizationSourceRecords = 100000

type LegalEntityConsumer struct {
	domain     OrganizationSyncDomain
	normalizer Normalizer
}

type ManagementCompanyConsumer struct {
	domain     OrganizationSyncDomain
	normalizer Normalizer
}

type ManagementDepartmentConsumer struct {
	domain     OrganizationSyncDomain
	normalizer Normalizer
}

type LegalDepartmentConsumer struct {
	domain     OrganizationSyncDomain
	normalizer Normalizer
}

type PositionConsumer struct {
	domain     OrganizationSyncDomain
	normalizer Normalizer
}

var (
	_ integration.SyncResultConsumer = (*LegalEntityConsumer)(nil)
	_ integration.SyncResultConsumer = (*ManagementCompanyConsumer)(nil)
	_ integration.SyncResultConsumer = (*ManagementDepartmentConsumer)(nil)
	_ integration.SyncResultConsumer = (*LegalDepartmentConsumer)(nil)
	_ integration.SyncResultConsumer = (*PositionConsumer)(nil)
)

func NewLegalEntityConsumer(domain OrganizationSyncDomain, contract SourceContract) *LegalEntityConsumer {
	return &LegalEntityConsumer{domain: domain, normalizer: contract.normalizer()}
}

func NewManagementCompanyConsumer(domain OrganizationSyncDomain, contract SourceContract) *ManagementCompanyConsumer {
	return &ManagementCompanyConsumer{domain: domain, normalizer: contract.normalizer()}
}

func NewManagementDepartmentConsumer(domain OrganizationSyncDomain, contract SourceContract) *ManagementDepartmentConsumer {
	return &ManagementDepartmentConsumer{domain: domain, normalizer: contract.normalizer()}
}

func NewLegalDepartmentConsumer(domain OrganizationSyncDomain, contract SourceContract) *LegalDepartmentConsumer {
	return &LegalDepartmentConsumer{domain: domain, normalizer: contract.normalizer()}
}

func NewPositionConsumer(domain OrganizationSyncDomain, contract SourceContract) *PositionConsumer {
	return &PositionConsumer{domain: domain, normalizer: contract.normalizer()}
}

func (c *LegalEntityConsumer) Consume(ctx context.Context, request integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
	inputs, issues := normalizeWindowedSources(request, ObjectKindLegalEntity, c.normalizer.NormalizeLegalEntitySource)
	summary, err := c.domain.SynchronizeLegalEntities(ctx, NewBusinessSyncContext(request), inputs, issues)
	return consumptionResult(summary, err)
}

func (c *ManagementCompanyConsumer) Consume(ctx context.Context, request integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
	inputs, issues := normalizeWindowedSources(request, ObjectKindManagementCompany, c.normalizer.NormalizeManagementCompanySource)
	summary, err := c.domain.SynchronizeOrgUnits(ctx, NewBusinessSyncContext(request), ObjectKindManagementCompany, model.OrgStructureTypeManagement, inputs, issues)
	return consumptionResult(summary, err)
}

func (c *ManagementDepartmentConsumer) Consume(ctx context.Context, request integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
	inputs, issues := normalizeWindowedSources(request, ObjectKindManagementUnit, func(source HRDepartmentSourceDTO) (OrgUnitSyncInput, error) {
		return c.normalizer.NormalizeOrgUnitSource(source, ObjectKindManagementUnit)
	})
	summary, err := c.domain.SynchronizeOrgUnits(ctx, NewBusinessSyncContext(request), ObjectKindManagementUnit, model.OrgStructureTypeManagement, inputs, issues)
	return consumptionResult(summary, err)
}

func (c *LegalDepartmentConsumer) Consume(ctx context.Context, request integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
	inputs, issues := normalizeWindowedSources(request, ObjectKindLegalUnit, func(source HRDepartmentSourceDTO) (OrgUnitSyncInput, error) {
		return c.normalizer.NormalizeOrgUnitSource(source, ObjectKindLegalUnit)
	})
	summary, err := c.domain.SynchronizeOrgUnits(ctx, NewBusinessSyncContext(request), ObjectKindLegalUnit, model.OrgStructureTypeLegal, inputs, issues)
	return consumptionResult(summary, err)
}

func (c *PositionConsumer) Consume(ctx context.Context, request integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
	inputs, issues := normalizeWindowedSources(request, ObjectKindPosition, c.normalizer.NormalizePositionSource)
	summary, err := c.domain.SynchronizePositions(ctx, NewBusinessSyncContext(request), inputs, issues)
	return consumptionResult(summary, err)
}

func EnabledConsumerRegistrations(domain OrganizationSyncDomain, contract SourceContract) ([]integration.SyncConsumerRegistration, error) {
	if domain == nil || !contract.valid() {
		return nil, ErrSourceContractInvalid
	}
	return consumerRegistrations(domain, contract, integration.SyncConsumerStatusEnabled), nil
}

// DisabledConsumerRegistrations 保留固定 code/version，但不会出现在可选列表或被 Runtime 解析。
func DisabledConsumerRegistrations(domain OrganizationSyncDomain) []integration.SyncConsumerRegistration {
	return consumerRegistrations(domain, SourceContract{}, integration.SyncConsumerStatusDisabled)
}

func consumerRegistrations(domain OrganizationSyncDomain, contract SourceContract, status string) []integration.SyncConsumerRegistration {
	return []integration.SyncConsumerRegistration{
		consumerRegistration(ConsumerCodeLegalEntity, "HR 法人公司", 4<<20, status, NewLegalEntityConsumer(domain, contract)),
		consumerRegistration(ConsumerCodeManagementCompany, "HR 管理公司", 4<<20, status, NewManagementCompanyConsumer(domain, contract)),
		consumerRegistration(ConsumerCodeManagementDepartment, "HR 管理部门", 8<<20, status, NewManagementDepartmentConsumer(domain, contract)),
		consumerRegistration(ConsumerCodeLegalDepartment, "HR 法人部门", 8<<20, status, NewLegalDepartmentConsumer(domain, contract)),
		consumerRegistration(ConsumerCodePosition, "HR 岗位", 8<<20, status, NewPositionConsumer(domain, contract)),
	}
}

func consumerRegistration(code, name string, maxResponse int64, status string, consumer integration.SyncResultConsumer) integration.SyncConsumerRegistration {
	return integration.SyncConsumerRegistration{Metadata: integration.SyncConsumerMetadata{
		Code: code, Version: ConsumerVersionV1, Name: name, Status: status,
		ContentTypes: []string{"application/json"}, MaxResponseBytes: maxResponse, MaxDuration: 60 * time.Second,
		CheckpointModes: []string{model.IntegrationSyncCheckpointTimestamp},
	}, Consumer: consumer}
}

func consumptionResult(summary BusinessSyncSummary, err error) (integration.SyncConsumptionResult, error) {
	if err != nil {
		return integration.SyncConsumptionResult{}, err
	}
	if summary.Success() {
		return integration.NewSyncConsumptionResult(true, "", summary.SuccessCount, 0, summary.BatchNo)
	}
	reason := summary.ReasonCode
	if reason == "" {
		reason = ReasonBusinessConflict
	}
	failed := summary.FailedCount
	if failed < 1 {
		failed = 1
	}
	return integration.NewSyncConsumptionResult(false, string(reason), summary.SuccessCount, failed, summary.BatchNo)
}

type changedSource interface {
	LegalEntitySyncInput | OrgUnitSyncInput | PositionSyncInput
}

func normalizeWindowedSources[S HRCompanySourceDTO | HRDepartmentSourceDTO | HRPositionSourceDTO, T changedSource](
	request integration.SyncConsumptionRequest,
	kind ObjectKind,
	normalize func(S) (T, error),
) ([]T, []SourceIssue) {
	start, end := request.WindowStart(), request.WindowEnd()
	if start == nil || end == nil || !end.After(*start) {
		return nil, []SourceIssue{{ReasonCode: ReasonEnvelopeInvalid, Action: model.OrgSyncRecordActionError}}
	}
	inputs := make([]T, 0)
	issues := make([]SourceIssue, 0)
	err := decodeSourceEnvelope(request.Body(), func(source S) error {
		input, err := normalize(source)
		if err != nil {
			issues = append(issues, sourceIssue(source, kind, err))
			return nil
		}
		changedAt := sourceChangedAt(input)
		classification, err := ClassifySourceChangeTime(changedAt, *start, *end)
		if err != nil {
			issues = append(issues, SourceIssue{ReasonCode: ReasonEnvelopeInvalid, Action: model.OrgSyncRecordActionError})
			return nil
		}
		if classification != WindowRecordFuture {
			inputs = append(inputs, input)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrSourceEnumInvalid) {
			return nil, []SourceIssue{{ObjectKind: kind, ReasonCode: ReasonEnumUnknown, Action: model.OrgSyncRecordActionError}}
		}
		return nil, []SourceIssue{{ReasonCode: ReasonEnvelopeInvalid, Action: model.OrgSyncRecordActionError}}
	}
	return inputs, issues
}

func sourceChangedAt[T changedSource](value T) time.Time {
	switch typed := any(value).(type) {
	case LegalEntitySyncInput:
		return typed.SourceChangedAt
	case OrgUnitSyncInput:
		return typed.SourceChangedAt
	case PositionSyncInput:
		return typed.SourceChangedAt
	default:
		return time.Time{}
	}
}

func sourceIssue[S HRCompanySourceDTO | HRDepartmentSourceDTO | HRPositionSourceDTO](source S, kind ObjectKind, err error) SourceIssue {
	var sourceID string
	switch value := any(source).(type) {
	case HRCompanySourceDTO:
		sourceID = strings.TrimSpace(value.SourceID)
	case HRDepartmentSourceDTO:
		sourceID = strings.TrimSpace(value.SourceID)
	case HRPositionSourceDTO:
		sourceID = strings.TrimSpace(value.SourceID)
	}
	reason := ReasonEnvelopeInvalid
	if sourceID == "" {
		reason = ReasonSourceIDMissing
	} else if errors.Is(err, ErrSourceContractUnconfirmed) {
		reason = ReasonEnumUnknown
	}
	issue := SourceIssue{ObjectKind: kind, ReasonCode: reason, Action: model.OrgSyncRecordActionError}
	if sourceID != "" {
		if key, keyErr := NewSourceKey(OrganizationHRSourceSystemCode, kind, sourceID); keyErr == nil {
			issue.SourceIDSummary = key.Digest()
		}
	}
	return issue
}

func decodeSourceEnvelope[T any](body []byte, consume func(T) error) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return ErrSourceContractUnconfirmed
	}
	var success *bool
	dataSeen := false
	count := 0
	for decoder.More() {
		nameToken, err := decoder.Token()
		if err != nil {
			return err
		}
		name, ok := nameToken.(string)
		if !ok {
			return ErrSourceContractUnconfirmed
		}
		switch name {
		case "success":
			var value bool
			if err := decoder.Decode(&value); err != nil {
				return err
			}
			success = &value
		case "data":
			if dataSeen {
				return ErrSourceContractUnconfirmed
			}
			dataSeen = true
			open, err := decoder.Token()
			if err != nil || open != json.Delim('[') {
				return ErrSourceContractUnconfirmed
			}
			for decoder.More() {
				count++
				if count > maxOrganizationSourceRecords {
					return ErrSourceContractUnconfirmed
				}
				var value T
				if err := decoder.Decode(&value); err != nil {
					return err
				}
				if err := consume(value); err != nil {
					return err
				}
			}
			if closeToken, err := decoder.Token(); err != nil || closeToken != json.Delim(']') {
				return ErrSourceContractUnconfirmed
			}
		default:
			var discard json.RawMessage
			if err := decoder.Decode(&discard); err != nil {
				return err
			}
		}
	}
	if closeToken, err := decoder.Token(); err != nil || closeToken != json.Delim('}') || success == nil || !*success || !dataSeen {
		return ErrSourceContractUnconfirmed
	}
	if _, err := decoder.Token(); err != io.EOF {
		return ErrSourceContractUnconfirmed
	}
	return nil
}
