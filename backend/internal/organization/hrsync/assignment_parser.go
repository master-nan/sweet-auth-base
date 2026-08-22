package hrsync

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

const (
	MaxAssignmentSourceBytes = 256 << 10
	MaxAssignmentSourceItems = 100
)

var (
	ErrAssignmentSourceInvalid    = errors.New("org_sync_assignment_invalid")
	ErrAssignmentPeriodInvalid    = errors.New("org_sync_assignment_period_invalid")
	ErrAssignmentStatusConflict   = errors.New("org_sync_assignment_status_conflict")
	ErrAssignmentSourceConflict   = errors.New("org_sync_assignment_source_conflict")
	ErrSourceCrosswalkUnavailable = errors.New("org_sync_source_crosswalk_unavailable")
)

type AssignmentPeriodState string

const (
	AssignmentPeriodCurrent    AssignmentPeriodState = "current"
	AssignmentPeriodHistorical AssignmentPeriodState = "historical"
	AssignmentPeriodFuture     AssignmentPeriodState = "future"
)

type AssignmentPeriod struct {
	ValidFrom time.Time
	ValidTo   *time.Time
	State     AssignmentPeriodState
}

// AssignmentSourceCandidate 只表示Source Adapter输出；
// 在来源合同未确认前不能作为可持久化的Organization任职输入。
type AssignmentSourceCandidate struct {
	RelationID      string
	LegalEntityNCID string
	OrgUnitNCID     string
	PositionNCID    string
	StructureType   int
	AssignmentType  string
	Period          AssignmentPeriod
}

type AssignmentSourceParser struct {
	location *time.Location
	asOf     time.Time
}

func NewAssignmentSourceParser(location *time.Location, asOf time.Time) (AssignmentSourceParser, error) {
	if location == nil || asOf.IsZero() {
		return AssignmentSourceParser{}, ErrSourceContractInvalid
	}
	return AssignmentSourceParser{location: location, asOf: asOf.UTC()}, nil
}

func (p AssignmentSourceParser) Parse(raw string) ([]AssignmentSourceCandidate, error) {
	sources, err := ParseAssignmentSourceDTOs(raw)
	if err != nil {
		return nil, err
	}
	result := make([]AssignmentSourceCandidate, 0, len(sources))
	for _, source := range sources {
		period, err := NormalizeAssignmentPeriod(source, p.location, p.asOf)
		if err != nil {
			return nil, err
		}
		legalEntityID := strings.TrimSpace(source.LegalEntityID)
		orgUnitID := strings.TrimSpace(source.OrgUnitID)
		positionID := strings.TrimSpace(source.PositionID)
		if legalEntityID == "" || orgUnitID == "" || len(legalEntityID) > MaxRawSourceIDLength ||
			len(orgUnitID) > MaxRawSourceIDLength || len(positionID) > MaxRawSourceIDLength {
			return nil, ErrAssignmentSourceInvalid
		}
		result = append(result, AssignmentSourceCandidate{
			RelationID: strings.TrimSpace(source.SourceID), LegalEntityNCID: legalEntityID, OrgUnitNCID: orgUnitID,
			PositionNCID: positionID, StructureType: source.StructureType,
			AssignmentType: strings.TrimSpace(source.AssignmentTypeID), Period: period,
		})
	}
	return result, nil
}

// ParseAssignmentSourceDTOs 只执行有界JSON结构校验；
// 任职周期和交叉引用语义由独立、显式的规范化步骤处理。
func ParseAssignmentSourceDTOs(raw string) ([]HRAssignmentSourceDTO, error) {
	if len(raw) > MaxAssignmentSourceBytes {
		return nil, ErrAssignmentSourceInvalid
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	open, err := decoder.Token()
	if err != nil || open != json.Delim('[') {
		return nil, ErrAssignmentSourceInvalid
	}
	result := make([]HRAssignmentSourceDTO, 0)
	seen := make(map[string]struct{})
	for decoder.More() {
		if len(result) >= MaxAssignmentSourceItems {
			return nil, ErrAssignmentSourceInvalid
		}
		var source HRAssignmentSourceDTO
		if err := decoder.Decode(&source); err != nil {
			return nil, ErrAssignmentSourceInvalid
		}
		relationID := strings.TrimSpace(source.SourceID)
		if relationID == "" || len(relationID) > MaxRawSourceIDLength {
			return nil, ErrAssignmentSourceInvalid
		}
		if _, duplicate := seen[relationID]; duplicate {
			return nil, ErrAssignmentSourceConflict
		}
		seen[relationID] = struct{}{}
		result = append(result, source)
	}
	if closeToken, err := decoder.Token(); err != nil || closeToken != json.Delim(']') {
		return nil, ErrAssignmentSourceInvalid
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, ErrAssignmentSourceInvalid
	}
	return result, nil
}

func NormalizeAssignmentPeriod(source HRAssignmentSourceDTO, location *time.Location, asOf time.Time) (AssignmentPeriod, error) {
	if location == nil || asOf.IsZero() {
		return AssignmentPeriod{}, ErrSourceContractInvalid
	}
	start, err := parseAssignmentDateTime(source.StartedAt, location)
	if err != nil {
		return AssignmentPeriod{}, ErrAssignmentPeriodInvalid
	}
	var end *time.Time
	if strings.TrimSpace(source.EndedAt) != "" {
		parsed, err := parseAssignmentDateTime(source.EndedAt, location)
		if err != nil || parsed.Before(start) {
			return AssignmentPeriod{}, ErrAssignmentPeriodInvalid
		}
		end = &parsed
	}

	onPost := strings.ToUpper(strings.TrimSpace(source.OnPost))
	ended := strings.ToUpper(strings.TrimSpace(source.Ended))
	switch {
	case onPost == "Y" && ended == "N":
		if end != nil && !asOf.Before(*end) {
			return AssignmentPeriod{}, ErrAssignmentStatusConflict
		}
		state := AssignmentPeriodCurrent
		if start.After(asOf) {
			state = AssignmentPeriodFuture
		}
		return AssignmentPeriod{ValidFrom: start, ValidTo: end, State: state}, nil
	case onPost == "N" && ended == "Y":
		if end == nil {
			return AssignmentPeriod{}, ErrAssignmentPeriodInvalid
		}
		if end.After(asOf) {
			return AssignmentPeriod{}, ErrAssignmentStatusConflict
		}
		return AssignmentPeriod{ValidFrom: start, ValidTo: end, State: AssignmentPeriodHistorical}, nil
	default:
		return AssignmentPeriod{}, ErrAssignmentStatusConflict
	}
}

func parseAssignmentDateTime(value string, location *time.Location) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", time.DateOnly} {
		if parsed, err := time.ParseInLocation(layout, value, location); err == nil {
			if parsed.Year() >= 1900 && parsed.Year() <= 9998 {
				return parsed.UTC(), nil
			}
		}
	}
	return time.Time{}, ErrAssignmentPeriodInvalid
}

type OrganizationSourceReference struct {
	SourceSystemCode string
	ObjectKind       ObjectKind
	NCID             string
}

// OrganizationSourceCrosswalkResolver 是显式Adapter端口。
// 当前没有生产可用的NCID到BIP映射来源，因此默认实现始终拒绝解析。
type OrganizationSourceCrosswalkResolver interface {
	Resolve(context.Context, OrganizationSourceReference) (SourceKey, error)
}

type UnavailableOrganizationSourceCrosswalkResolver struct{}

func (UnavailableOrganizationSourceCrosswalkResolver) Resolve(context.Context, OrganizationSourceReference) (SourceKey, error) {
	return SourceKey{}, ErrSourceCrosswalkUnavailable
}
