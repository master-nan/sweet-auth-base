package hrsync

import (
	"errors"
	"strings"
	"time"
)

var ErrSourceContractUnconfirmed = errors.New("org_sync_source_contract_unconfirmed")

type Normalizer struct {
	SourceSystemCode string
	SourceLocation   *time.Location
}

func (n Normalizer) NormalizeLegalEntitySource(source HRCompanySourceDTO) (LegalEntitySyncInput, error) {
	key, changedAt, status, err := n.common(ObjectKindLegalEntity, source.SourceID, source.ChangeTime, source.Enabled)
	if err != nil || strings.TrimSpace(source.SourceCode) == "" || strings.TrimSpace(source.Name) == "" {
		return LegalEntitySyncInput{}, normalizeRequiredError(err)
	}
	return LegalEntitySyncInput{Key: key, Code: strings.TrimSpace(source.SourceCode), Name: strings.TrimSpace(source.Name), ShortName: strings.TrimSpace(source.ShortName), ParentSourceID: strings.TrimSpace(source.ParentSourceID), Status: status, SourceChangedAt: changedAt}, nil
}

func (n Normalizer) NormalizeOrgUnitSource(source HRDepartmentSourceDTO, kind ObjectKind) (OrgUnitSyncInput, error) {
	if kind != ObjectKindManagementUnit && kind != ObjectKindLegalUnit {
		return OrgUnitSyncInput{}, ErrSourceKeyInvalid
	}
	key, changedAt, status, err := n.common(kind, source.SourceID, source.ChangeTime, source.Enabled)
	if err != nil || strings.TrimSpace(source.SourceCode) == "" || strings.TrimSpace(source.Name) == "" {
		return OrgUnitSyncInput{}, normalizeRequiredError(err)
	}
	return OrgUnitSyncInput{Key: key, Code: strings.TrimSpace(source.SourceCode), Name: strings.TrimSpace(source.Name), ParentSourceID: strings.TrimSpace(source.ParentSourceID), LegalEntitySourceID: strings.TrimSpace(source.LegalEntitySourceID), Status: status, SourceChangedAt: changedAt}, nil
}

func (n Normalizer) NormalizePositionSource(source HRPositionSourceDTO) (PositionSyncInput, error) {
	key, changedAt, status, err := n.common(ObjectKindPosition, source.SourceID, source.ChangeTime, source.Enabled)
	if err != nil || strings.TrimSpace(source.SourceCode) == "" || strings.TrimSpace(source.Name) == "" || strings.TrimSpace(source.OrgUnitSourceID) == "" {
		return PositionSyncInput{}, normalizeRequiredError(err)
	}
	return PositionSyncInput{Key: key, Code: strings.TrimSpace(source.SourceCode), Name: strings.TrimSpace(source.Name), OrgUnitSourceID: strings.TrimSpace(source.OrgUnitSourceID), JobLevel: strings.TrimSpace(source.JobLevel), Status: status, SourceChangedAt: changedAt}, nil
}

func (n Normalizer) NormalizeEmployeeSource(source HREmployeeSourceDTO) (EmployeeSyncInput, error) {
	key, changedAt, status, err := n.common(ObjectKindEmployee, source.SourceID, source.ChangeTime, source.Enabled)
	if err != nil || strings.TrimSpace(source.EmployeeNo) == "" || strings.TrimSpace(source.Name) == "" {
		return EmployeeSyncInput{}, normalizeRequiredError(err)
	}
	return EmployeeSyncInput{Key: key, EmployeeNo: strings.TrimSpace(source.EmployeeNo), Name: strings.TrimSpace(source.Name), Status: status, SourceChangedAt: changedAt}, nil
}

func (n Normalizer) NormalizeAssignmentSource(HRAssignmentSourceDTO) (AssignmentSyncInput, error) {
	return AssignmentSyncInput{}, ErrSourceContractUnconfirmed
}

func (n Normalizer) common(kind ObjectKind, sourceID, changedAt string, enabled SourceEnableStatus) (SourceKey, time.Time, CanonicalStatus, error) {
	key, err := NewSourceKey(n.SourceSystemCode, kind, sourceID)
	if err != nil {
		return SourceKey{}, time.Time{}, "", err
	}
	if n.SourceLocation == nil || enabled == SourceEnableUnknown {
		return SourceKey{}, time.Time{}, "", ErrSourceContractUnconfirmed
	}
	parsed, err := time.ParseInLocation("2006-01-02T15:04:05", strings.TrimSpace(changedAt), n.SourceLocation)
	if err != nil {
		return SourceKey{}, time.Time{}, "", err
	}
	status := CanonicalStatusDisabled
	if enabled == SourceEnableEnabled {
		status = CanonicalStatusEnabled
	}
	return key, parsed.UTC(), status, nil
}

func normalizeRequiredError(err error) error {
	if err != nil {
		return err
	}
	return ErrSourceKeyInvalid
}
