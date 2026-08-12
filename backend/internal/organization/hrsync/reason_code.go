package hrsync

type ReasonCode string

const (
	ReasonEnvelopeInvalid            ReasonCode = "org_sync_envelope_invalid"
	ReasonSourceIDMissing            ReasonCode = "org_sync_source_id_missing"
	ReasonSourceIDConflict           ReasonCode = "org_sync_source_id_conflict"
	ReasonParentUnresolved           ReasonCode = "org_sync_parent_unresolved"
	ReasonParentSelfReference        ReasonCode = "org_sync_parent_self_reference"
	ReasonParentCycle                ReasonCode = "org_sync_parent_cycle"
	ReasonParentInvalid              ReasonCode = "org_sync_parent_invalid"
	ReasonReferenceMissing           ReasonCode = "org_sync_reference_missing"
	ReasonAssignmentInvalid          ReasonCode = "org_sync_assignment_invalid"
	ReasonAssignmentPeriodInvalid    ReasonCode = "org_sync_assignment_period_invalid"
	ReasonAssignmentStatusConflict   ReasonCode = "org_sync_assignment_status_conflict"
	ReasonPrimaryAssignmentAmbiguous ReasonCode = "org_sync_primary_assignment_ambiguous"
	ReasonAssignmentSourceConflict   ReasonCode = "org_sync_assignment_source_conflict"
	ReasonEnumUnknown                ReasonCode = "org_sync_enum_unknown"
	ReasonEmploymentStateConflict    ReasonCode = "org_sync_employment_state_conflict"
	ReasonBusinessConflict           ReasonCode = "org_sync_business_conflict"
	ReasonPersistenceFailed          ReasonCode = "org_sync_persistence_failed"
)

var reasonCodes = map[ReasonCode]struct{}{
	ReasonEnvelopeInvalid: {}, ReasonSourceIDMissing: {}, ReasonSourceIDConflict: {},
	ReasonParentUnresolved: {}, ReasonParentSelfReference: {}, ReasonParentCycle: {}, ReasonParentInvalid: {},
	ReasonReferenceMissing: {}, ReasonAssignmentInvalid: {}, ReasonAssignmentPeriodInvalid: {},
	ReasonAssignmentStatusConflict: {}, ReasonPrimaryAssignmentAmbiguous: {}, ReasonAssignmentSourceConflict: {},
	ReasonEnumUnknown: {}, ReasonEmploymentStateConflict: {}, ReasonBusinessConflict: {}, ReasonPersistenceFailed: {},
}

func (code ReasonCode) Valid() bool {
	_, ok := reasonCodes[code]
	return ok
}
