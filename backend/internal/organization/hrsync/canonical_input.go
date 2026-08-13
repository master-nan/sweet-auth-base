package hrsync

import "time"

type CanonicalStatus string

const (
	CanonicalStatusEnabled  CanonicalStatus = "enabled"
	CanonicalStatusDisabled CanonicalStatus = "disabled"
)

type LegalEntitySyncInput struct {
	Key             SourceKey
	SourceCode      string
	Code            string
	Name            string
	ShortName       string
	ParentSourceID  string
	Status          CanonicalStatus
	SourceChangedAt time.Time
}

type OrgUnitSyncInput struct {
	Key                 SourceKey
	SourceCode          string
	Code                string
	Name                string
	ParentSourceID      string
	LegalEntitySourceID string
	Status              CanonicalStatus
	SourceChangedAt     time.Time
	Level               int
	Sort                int
}

type PositionSyncInput struct {
	Key             SourceKey
	Code            string
	Name            string
	OrgUnitSourceID string
	JobLevel        string
	Status          CanonicalStatus
	SourceChangedAt time.Time
}

type EmployeeSyncInput struct {
	Key              SourceKey
	EmployeeNo       string
	Name             string
	Mobile           string
	Email            string
	EmploymentStatus string
	SourceChangedAt  time.Time
}

type AssignmentSyncInput struct {
	Key SourceKey
}
