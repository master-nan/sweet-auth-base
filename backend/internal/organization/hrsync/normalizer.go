package hrsync

import (
	"errors"
	"strconv"
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
	return LegalEntitySyncInput{Key: key, SourceCode: strings.TrimSpace(source.SourceRecordID), Code: strings.TrimSpace(source.SourceCode), Name: strings.TrimSpace(source.Name), ShortName: strings.TrimSpace(source.ShortName), ParentSourceID: strings.TrimSpace(source.ParentSourceID), Status: status, SourceChangedAt: changedAt}, nil
}

func (n Normalizer) NormalizeManagementCompanySource(source HRCompanySourceDTO) (OrgUnitSyncInput, error) {
	key, changedAt, status, err := n.common(ObjectKindManagementCompany, source.SourceID, source.ChangeTime, source.Enabled)
	if err != nil || strings.TrimSpace(source.SourceCode) == "" || strings.TrimSpace(source.Name) == "" {
		return OrgUnitSyncInput{}, normalizeRequiredError(err)
	}
	return OrgUnitSyncInput{
		Key: key, SourceCode: strings.TrimSpace(source.SourceRecordID), Code: strings.TrimSpace(source.SourceCode),
		Name: strings.TrimSpace(source.Name), ParentSourceID: strings.TrimSpace(source.ParentSourceID),
		Status: status, SourceChangedAt: changedAt, Level: source.Level,
	}, nil
}

func (n Normalizer) NormalizeOrgUnitSource(source HRDepartmentSourceDTO, kind ObjectKind) (OrgUnitSyncInput, error) {
	if kind != ObjectKindManagementUnit && kind != ObjectKindLegalUnit {
		return OrgUnitSyncInput{}, ErrSourceKeyInvalid
	}
	key, changedAt, status, err := n.common(kind, source.SourceID, source.ChangeTime, source.Enabled)
	if err != nil || strings.TrimSpace(source.SourceCode) == "" || strings.TrimSpace(source.Name) == "" {
		return OrgUnitSyncInput{}, normalizeRequiredError(err)
	}
	sort, err := parseSourceSort(source.Sort)
	if err != nil {
		return OrgUnitSyncInput{}, err
	}
	return OrgUnitSyncInput{Key: key, SourceCode: strings.TrimSpace(source.SourceRecordID), Code: strings.TrimSpace(source.SourceCode), Name: strings.TrimSpace(source.Name), ParentSourceID: strings.TrimSpace(source.ParentSourceID), LegalEntitySourceID: strings.TrimSpace(source.LegalEntitySourceID), Status: status, SourceChangedAt: changedAt, Level: source.Level, Sort: sort}, nil
}

func (n Normalizer) NormalizePositionSource(source HRPositionSourceDTO) (PositionSyncInput, error) {
	key, changedAt, status, err := n.common(ObjectKindPosition, source.SourceID, source.ChangeTime, source.Enabled)
	code := strings.TrimSpace(source.SourceCode)
	name := strings.TrimSpace(source.Name)
	orgUnitSourceID := strings.TrimSpace(source.OrgUnitSourceID)
	jobLevel := strings.TrimSpace(source.JobLevel)
	if err != nil || code == "" || name == "" {
		return PositionSyncInput{}, normalizeRequiredError(err)
	}
	if len(code) > 128 || len(name) > 255 || len(orgUnitSourceID) > MaxRawSourceIDLength || len(jobLevel) > 64 {
		return PositionSyncInput{}, ErrSourceKeyInvalid
	}
	return PositionSyncInput{Key: key, Code: code, Name: name, OrgUnitSourceID: orgUnitSourceID, JobLevel: jobLevel, Status: status, SourceChangedAt: changedAt}, nil
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

func parseSourceSort(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, ErrSourceContractUnconfirmed
	}
	return parsed, nil
}
