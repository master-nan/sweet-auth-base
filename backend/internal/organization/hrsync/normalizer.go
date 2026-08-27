package hrsync

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var ErrSourceContractUnconfirmed = errors.New("org_sync_source_contract_unconfirmed")
var ErrSourceDateInvalid = errors.New("org_sync_source_date_invalid")

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
	sourceCode := strings.TrimSpace(source.SourceCode)
	if err != nil || sourceCode == "" || strings.TrimSpace(source.Name) == "" {
		return OrgUnitSyncInput{}, normalizeRequiredError(err)
	}
	return OrgUnitSyncInput{
		Key: key, SourceCode: sourceCode, Code: key.PersistenceID(),
		Name: strings.TrimSpace(source.Name), ParentSourceID: strings.TrimSpace(source.ParentSourceID),
		Status: status, SourceChangedAt: changedAt, Level: source.Level,
	}, nil
}

func (n Normalizer) NormalizeOrgUnitSource(source HRDepartmentSourceDTO, kind ObjectKind) (OrgUnitSyncInput, error) {
	if kind != ObjectKindManagementUnit && kind != ObjectKindLegalUnit {
		return OrgUnitSyncInput{}, ErrSourceKeyInvalid
	}
	key, changedAt, status, err := n.common(kind, source.SourceID, source.ChangeTime, source.Enabled)
	sourceCode := strings.TrimSpace(source.SourceCode)
	if err != nil || strings.TrimSpace(source.Name) == "" || len(sourceCode) > 128 {
		return OrgUnitSyncInput{}, normalizeRequiredError(err)
	}
	sort, err := parseSourceSort(source.Sort)
	if err != nil {
		return OrgUnitSyncInput{}, err
	}
	return OrgUnitSyncInput{Key: key, SourceCode: sourceCode, Code: key.PersistenceID(), Name: strings.TrimSpace(source.Name), ParentSourceID: strings.TrimSpace(source.ParentSourceID), LegalEntitySourceID: strings.TrimSpace(source.LegalEntitySourceID), LegalEntityCode: strings.TrimSpace(source.LegalEntityCode), Status: status, SourceChangedAt: changedAt, Level: source.Level, Sort: sort}, nil
}

func (n Normalizer) NormalizePositionSource(source HRPositionSourceDTO) (PositionSyncInput, error) {
	key, changedAt, status, err := n.common(ObjectKindPosition, source.SourceID, source.ChangeTime, source.Enabled)
	sourceCode := strings.TrimSpace(source.SourceCode)
	name := strings.TrimSpace(source.Name)
	orgUnitSourceID := strings.TrimSpace(source.OrgUnitSourceID)
	jobLevel := strings.TrimSpace(source.JobLevel)
	if err != nil || name == "" {
		return PositionSyncInput{}, normalizeRequiredError(err)
	}
	if len(sourceCode) > 128 || len(name) > 255 || len(orgUnitSourceID) > MaxRawSourceIDLength || len(jobLevel) > 64 {
		return PositionSyncInput{}, ErrSourceKeyInvalid
	}
	return PositionSyncInput{Key: key, SourceCode: sourceCode, Code: key.PersistenceID(), Name: name, OrgUnitSourceID: orgUnitSourceID, JobLevel: jobLevel, Status: status, SourceChangedAt: changedAt}, nil
}

func (n Normalizer) NormalizeEmployeeSource(source HREmployeeSourceDTO) (EmployeeSyncInput, error) {
	key, changedAt, status, err := n.common(ObjectKindEmployee, source.SourceID, source.ChangeTime, source.Enabled)
	employeeNo := strings.TrimSpace(source.EmployeeNo)
	name := strings.TrimSpace(source.Name)
	mobile := strings.TrimSpace(source.Mobile)
	email := strings.TrimSpace(source.Email)
	if err != nil || employeeNo == "" || name == "" {
		return EmployeeSyncInput{}, normalizeRequiredError(err)
	}
	if len(employeeNo) > 128 || len(name) > 128 || len(mobile) > 64 || len(email) > 128 {
		return EmployeeSyncInput{}, ErrSourceKeyInvalid
	}
	employmentStatus := "suspended"
	if status == CanonicalStatusEnabled {
		employmentStatus = "active"
	}
	return EmployeeSyncInput{
		Key: key, EmployeeNo: employeeNo, Name: name, Mobile: mobile, Email: email,
		EmploymentStatus: employmentStatus, SourceChangedAt: changedAt,
	}, nil
}

func (n Normalizer) NormalizeResignedEmployeeSource(source HRResignedEmployeeSourceDTO) (ResignationSyncInput, error) {
	key, changedAt, err := n.sourceFact(ObjectKindEmployee, source.SourceID, source.ChangeTime)
	if err != nil {
		return ResignationSyncInput{}, err
	}
	resignedOn, err := parseSourceLocalDate(source.ResignedAt, n.SourceLocation)
	if err != nil {
		return ResignationSyncInput{}, err
	}
	return ResignationSyncInput{Key: key, ResignedOn: resignedOn, SourceChangedAt: changedAt}, nil
}

func (n Normalizer) NormalizeAssignmentSource(HRAssignmentSourceDTO) (AssignmentSyncInput, error) {
	return AssignmentSyncInput{}, ErrSourceContractUnconfirmed
}

func (n Normalizer) common(kind ObjectKind, sourceID, changedAt string, enabled SourceEnableStatus) (SourceKey, time.Time, CanonicalStatus, error) {
	key, parsed, err := n.sourceFact(kind, sourceID, changedAt)
	if err != nil {
		return SourceKey{}, time.Time{}, "", err
	}
	if enabled == SourceEnableUnknown {
		return SourceKey{}, time.Time{}, "", ErrSourceContractUnconfirmed
	}
	status := CanonicalStatusDisabled
	if enabled == SourceEnableEnabled {
		status = CanonicalStatusEnabled
	}
	return key, parsed.UTC(), status, nil
}

func (n Normalizer) sourceFact(kind ObjectKind, sourceID, changedAt string) (SourceKey, time.Time, error) {
	key, err := NewSourceKey(n.SourceSystemCode, kind, sourceID)
	if err != nil {
		return SourceKey{}, time.Time{}, err
	}
	if n.SourceLocation == nil {
		return SourceKey{}, time.Time{}, ErrSourceContractUnconfirmed
	}
	parsed, err := time.ParseInLocation("2006-01-02T15:04:05", strings.TrimSpace(changedAt), n.SourceLocation)
	if err != nil {
		return SourceKey{}, time.Time{}, err
	}
	return key, parsed.UTC(), nil
}

func parseSourceLocalDate(value string, location *time.Location) (time.Time, error) {
	if location == nil {
		return time.Time{}, ErrSourceContractUnconfirmed
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, ErrSourceDateInvalid
	}
	// 来源字段是LocalDate而非时间点；Canonical存储使用UTC零点载体，不经过来源时区偏移。
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil || parsed.Year() < 1900 || parsed.Year() > 9998 {
		return time.Time{}, ErrSourceDateInvalid
	}
	return parsed.UTC(), nil
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
