package service

import (
	"backend/internal/organization/hrsync"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	organizationManagementStructureCode = "hr_management"
	organizationLegalStructureCode      = "hr_legal"
	organizationManagedSourceCode       = "platform"
)

var ErrOrganizationHRSyncInvalid = errors.New("organization hr sync context invalid")

type OrganizationHRSyncService struct {
	repository repository.OrganizationHRSyncRepository
	sf         *utils.Snowflake
}

var _ hrsync.OrganizationSyncDomain = (*OrganizationHRSyncService)(nil)

func NewOrganizationHRSyncService(repository repository.OrganizationHRSyncRepository, sf *utils.Snowflake) *OrganizationHRSyncService {
	return &OrganizationHRSyncService{repository: repository, sf: sf}
}

type organizationBusinessBatch struct {
	value model.OrgSyncBatch
	now   time.Time
}

type organizationSyncOutcome struct {
	kind           hrsync.ObjectKind
	sourceSummary  string
	localID        *int
	action         string
	status         string
	reason         hrsync.ReasonCode
	dependencyType string
	dependencyKey  string
}

func (s *OrganizationHRSyncService) SynchronizeLegalEntities(
	ctx context.Context,
	business hrsync.BusinessSyncContext,
	inputs []hrsync.LegalEntitySyncInput,
	issues []hrsync.SourceIssue,
) (hrsync.BusinessSyncSummary, error) {
	inputs, issues = dedupeLegalEntityInputs(inputs, issues)
	batch, err := s.beginBusinessBatch(ctx, business, string(hrsync.ObjectKindLegalEntity), hrsync.ConsumerCodeLegalEntity)
	if err != nil {
		return hrsync.BusinessSyncSummary{}, err
	}
	outcomes := sourceIssueOutcomes(issues, hrsync.ObjectKindLegalEntity)
	validInputs := make([]hrsync.LegalEntitySyncInput, 0, len(inputs))
	for start := 0; start < len(inputs); start += hrsync.OrganizationHRSyncChunkSize {
		end := min(start+hrsync.OrganizationHRSyncChunkSize, len(inputs))
		chunkOutcomes, chunkInputs, chunkErr := s.upsertLegalEntityChunk(ctx, batch, inputs[start:end])
		if chunkErr != nil {
			outcomes = append(outcomes, persistenceFailureOutcomes(inputs[start:end], hrsync.ObjectKindLegalEntity)...)
			continue
		}
		outcomes = append(outcomes, chunkOutcomes...)
		validInputs = append(validInputs, chunkInputs...)
	}
	relationOutcomes, err := s.resolveLegalEntityParents(ctx, batch, validInputs)
	if err != nil {
		outcomes = append(outcomes, organizationSyncOutcome{kind: hrsync.ObjectKindLegalEntity, sourceSummary: "relation", action: model.OrgSyncRecordActionError, status: "failed", reason: hrsync.ReasonPersistenceFailed})
	} else {
		outcomes = mergeOrganizationOutcomes(outcomes, relationOutcomes)
	}
	return s.finishBusinessBatch(ctx, batch, outcomes)
}

func (s *OrganizationHRSyncService) SynchronizeOrgUnits(
	ctx context.Context,
	business hrsync.BusinessSyncContext,
	kind hrsync.ObjectKind,
	structureType string,
	inputs []hrsync.OrgUnitSyncInput,
	issues []hrsync.SourceIssue,
) (hrsync.BusinessSyncSummary, error) {
	consumerCode, unitType, err := organizationConsumerContract(kind, structureType)
	if err != nil {
		return hrsync.BusinessSyncSummary{}, err
	}
	inputs, issues = dedupeOrgUnitInputs(inputs, issues, kind)
	batch, err := s.beginBusinessBatch(ctx, business, string(kind), consumerCode)
	if err != nil {
		return hrsync.BusinessSyncSummary{}, err
	}
	structure, err := s.ensureStructure(ctx, structureType)
	if err != nil {
		outcomes := append(sourceIssueOutcomes(issues, kind), organizationSyncOutcome{kind: kind, sourceSummary: "structure", action: model.OrgSyncRecordActionError, status: "failed", reason: hrsync.ReasonPersistenceFailed})
		return s.finishBusinessBatch(ctx, batch, outcomes)
	}
	outcomes := sourceIssueOutcomes(issues, kind)
	validInputs := make([]hrsync.OrgUnitSyncInput, 0, len(inputs))
	for start := 0; start < len(inputs); start += hrsync.OrganizationHRSyncChunkSize {
		end := min(start+hrsync.OrganizationHRSyncChunkSize, len(inputs))
		chunkOutcomes, chunkInputs, chunkErr := s.upsertOrgUnitChunk(ctx, batch, structure, kind, unitType, inputs[start:end])
		if chunkErr != nil {
			outcomes = append(outcomes, persistenceFailureOutcomes(inputs[start:end], kind)...)
			continue
		}
		outcomes = append(outcomes, chunkOutcomes...)
		validInputs = append(validInputs, chunkInputs...)
	}
	relationOutcomes, relationErr := s.resolveStructureRelations(ctx, batch, structure, kind, validInputs)
	if relationErr != nil {
		outcomes = append(outcomes, organizationSyncOutcome{kind: kind, sourceSummary: "relation", action: model.OrgSyncRecordActionError, status: "failed", reason: hrsync.ReasonPersistenceFailed})
	} else {
		outcomes = mergeOrganizationOutcomes(outcomes, relationOutcomes)
	}
	return s.finishBusinessBatch(ctx, batch, outcomes)
}

func organizationConsumerContract(kind hrsync.ObjectKind, structureType string) (string, string, error) {
	switch {
	case kind == hrsync.ObjectKindManagementCompany && structureType == model.OrgStructureTypeManagement:
		return hrsync.ConsumerCodeManagementCompany, "business_unit", nil
	case kind == hrsync.ObjectKindManagementUnit && structureType == model.OrgStructureTypeManagement:
		return hrsync.ConsumerCodeManagementDepartment, "department", nil
	case kind == hrsync.ObjectKindLegalUnit && structureType == model.OrgStructureTypeLegal:
		return hrsync.ConsumerCodeLegalDepartment, "department", nil
	default:
		return "", "", ErrOrganizationHRSyncInvalid
	}
}

func (s *OrganizationHRSyncService) beginBusinessBatch(ctx context.Context, business hrsync.BusinessSyncContext, objectScope, consumerCode string) (organizationBusinessBatch, error) {
	var result organizationBusinessBatch
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		execution, err := s.repository.FindExecutionByNo(tx, strings.TrimSpace(business.ExecutionNo))
		if err != nil || execution.SyncBatchID == nil || execution.SyncSliceNo == nil || execution.SyncConsumerVersion == nil ||
			execution.SyncConsumerCode != consumerCode || *execution.SyncConsumerVersion != hrsync.ConsumerVersionV1 ||
			*execution.SyncSliceNo != business.SliceNo {
			return ErrOrganizationHRSyncInvalid
		}
		integrationBatch, err := s.repository.FindIntegrationSyncBatchByID(tx, *execution.SyncBatchID)
		if err != nil || integrationBatch.BatchNo != business.SyncBatchNo || integrationBatch.TaskCode != business.TaskCode ||
			integrationBatch.TaskVersion != business.TaskVersion || integrationBatch.ConsumerCode != consumerCode ||
			integrationBatch.ConsumerVersion != hrsync.ConsumerVersionV1 {
			return ErrOrganizationHRSyncInvalid
		}
		now, err := s.repository.CurrentDatabaseTime(tx)
		if err != nil {
			return err
		}
		batch, err := s.repository.FindSyncBatchByExecutionForUpdate(tx, execution.Id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			id, generateErr := s.nextID()
			if generateErr != nil {
				return generateErr
			}
			batch = model.OrgSyncBatch{
				Basic: model.Basic{Id: id, State: true}, BatchNo: fmt.Sprintf("ORG-%d", id), ExecutionId: &execution.Id,
				SyncType: "incremental", ObjectScope: objectScope, StartedAt: &now, Status: "processing",
			}
			if err := s.repository.CreateSyncBatch(tx, &batch); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if batch.ObjectScope != objectScope {
				return ErrOrganizationHRSyncInvalid
			}
			if err := s.repository.UpdateSyncBatch(tx, batch.Id, map[string]any{
				"status": "processing", "completed_at": nil, "error_summary": "",
			}); err != nil {
				return err
			}
		}
		result = organizationBusinessBatch{value: batch, now: now.UTC()}
		return nil
	})
	return result, err
}

func (s *OrganizationHRSyncService) upsertLegalEntityChunk(ctx context.Context, batch organizationBusinessBatch, inputs []hrsync.LegalEntitySyncInput) ([]organizationSyncOutcome, []hrsync.LegalEntitySyncInput, error) {
	outcomes := make([]organizationSyncOutcome, 0, len(inputs))
	valid := make([]hrsync.LegalEntitySyncInput, 0, len(inputs))
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		for _, input := range inputs {
			outcome, ok, err := s.upsertLegalEntity(tx, batch, input)
			if err != nil {
				return err
			}
			outcomes = append(outcomes, outcome)
			if ok {
				valid = append(valid, input)
			}
			if err := s.upsertSyncRecord(tx, batch.value, outcome); err != nil {
				return err
			}
		}
		return nil
	})
	return outcomes, valid, err
}

func (s *OrganizationHRSyncService) upsertLegalEntity(tx *gorm.DB, batch organizationBusinessBatch, input hrsync.LegalEntitySyncInput) (organizationSyncOutcome, bool, error) {
	outcome := successOutcome(input.Key, hrsync.ObjectKindLegalEntity)
	identity := input.Key.PersistenceID()
	if err := s.repository.LockSourceIdentity(tx, input.Key.SourceSystemCode()+"|legal_entity|"+identity); err != nil {
		return outcome, false, err
	}
	existing, err := s.repository.FindLegalEntityBySource(tx, input.Key.SourceSystemCode(), identity)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		byCode, codeErr := s.repository.FindLegalEntityByCode(tx, input.Key.SourceSystemCode(), input.Code)
		if codeErr == nil && byCode.SourceId != identity {
			return failedOutcome(input.Key, hrsync.ObjectKindLegalEntity, hrsync.ReasonSourceIDConflict), false, nil
		}
		if codeErr != nil && !errors.Is(codeErr, gorm.ErrRecordNotFound) {
			return outcome, false, codeErr
		}
		id, generateErr := s.nextID()
		if generateErr != nil {
			return outcome, false, generateErr
		}
		value := model.OrgLegalEntity{
			Basic: model.Basic{Id: id, State: true}, SourceSystemCode: input.Key.SourceSystemCode(), SourceId: identity,
			SourceCode: input.SourceCode, Code: input.Code, Name: input.Name, ShortName: input.ShortName,
			EntityType: "legal_company", Status: string(input.Status), SourceUpdatedAt: organizationHRTimePointer(input.SourceChangedAt),
			SourceVersion: input.SourceChangedAt.Format(time.RFC3339Nano), LastSyncAt: &batch.now,
			SourceStatus: string(input.Status), SourceDeleted: false, SyncStatus: "synced",
		}
		if err := s.repository.CreateLegalEntity(tx, &value); err != nil {
			return outcome, false, err
		}
		outcome.localID, outcome.action = &value.Id, objectAction("", string(input.Status), true)
		return outcome, true, nil
	}
	if err != nil {
		return outcome, false, err
	}
	if existing.EntityType != "legal_company" {
		return failedOutcome(input.Key, hrsync.ObjectKindLegalEntity, hrsync.ReasonSourceIDConflict), false, nil
	}
	if stale, conflict := staleOrganizationFact(existing.SourceUpdatedAt, input.SourceChangedAt, legalEntityFactsEqual(existing, input)); stale {
		if conflict {
			return failedOutcome(input.Key, hrsync.ObjectKindLegalEntity, hrsync.ReasonSourceIDConflict), false, nil
		}
		outcome.localID, outcome.action = &existing.Id, model.OrgSyncRecordActionNoop
		return outcome, existing.SourceUpdatedAt != nil && input.SourceChangedAt.Equal(*existing.SourceUpdatedAt), nil
	}
	byCode, codeErr := s.repository.FindLegalEntityByCode(tx, input.Key.SourceSystemCode(), input.Code)
	if codeErr == nil && byCode.Id != existing.Id {
		return failedOutcome(input.Key, hrsync.ObjectKindLegalEntity, hrsync.ReasonSourceIDConflict), false, nil
	}
	if codeErr != nil && !errors.Is(codeErr, gorm.ErrRecordNotFound) {
		return outcome, false, codeErr
	}
	changed := !legalEntityFactsEqual(existing, input)
	if err := s.repository.UpdateLegalEntity(tx, existing.Id, map[string]any{
		"source_code": input.SourceCode, "code": input.Code, "name": input.Name, "short_name": input.ShortName,
		"status": string(input.Status), "source_updated_at": input.SourceChangedAt, "source_version": input.SourceChangedAt.Format(time.RFC3339Nano),
		"last_sync_at": batch.now, "source_status": string(input.Status), "source_deleted": false, "sync_status": "synced", "last_error": "",
	}); err != nil {
		return outcome, false, err
	}
	outcome.localID, outcome.action = &existing.Id, objectAction(existing.Status, string(input.Status), changed)
	return outcome, true, nil
}

func (s *OrganizationHRSyncService) upsertOrgUnitChunk(ctx context.Context, batch organizationBusinessBatch, structure model.OrgStructure, kind hrsync.ObjectKind, unitType string, inputs []hrsync.OrgUnitSyncInput) ([]organizationSyncOutcome, []hrsync.OrgUnitSyncInput, error) {
	outcomes := make([]organizationSyncOutcome, 0, len(inputs))
	valid := make([]hrsync.OrgUnitSyncInput, 0, len(inputs))
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		for _, input := range inputs {
			outcome, unit, ok, err := s.upsertOrgUnit(tx, batch, kind, unitType, input)
			if err != nil {
				return err
			}
			outcomes = append(outcomes, outcome)
			if ok {
				if err := s.ensureStructureNodePlaceholder(tx, structure, input, unit); err != nil {
					return err
				}
				valid = append(valid, input)
			}
			if err := s.upsertSyncRecord(tx, batch.value, outcome); err != nil {
				return err
			}
		}
		return nil
	})
	return outcomes, valid, err
}

func (s *OrganizationHRSyncService) upsertOrgUnit(tx *gorm.DB, batch organizationBusinessBatch, kind hrsync.ObjectKind, unitType string, input hrsync.OrgUnitSyncInput) (organizationSyncOutcome, model.OrgUnit, bool, error) {
	outcome := successOutcome(input.Key, kind)
	identity := input.Key.PersistenceID()
	if err := s.repository.LockSourceIdentity(tx, input.Key.SourceSystemCode()+"|org_unit|"+identity); err != nil {
		return outcome, model.OrgUnit{}, false, err
	}
	existing, err := s.repository.FindOrgUnitBySource(tx, input.Key.SourceSystemCode(), identity)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		byCode, codeErr := s.repository.FindOrgUnitByCode(tx, input.Key.SourceSystemCode(), input.Code)
		if codeErr == nil && byCode.SourceId != identity {
			return failedOutcome(input.Key, kind, hrsync.ReasonSourceIDConflict), model.OrgUnit{}, false, nil
		}
		if codeErr != nil && !errors.Is(codeErr, gorm.ErrRecordNotFound) {
			return outcome, model.OrgUnit{}, false, codeErr
		}
		id, generateErr := s.nextID()
		if generateErr != nil {
			return outcome, model.OrgUnit{}, false, generateErr
		}
		value := model.OrgUnit{
			Basic: model.Basic{Id: id, State: true}, SourceSystemCode: input.Key.SourceSystemCode(), SourceId: identity,
			SourceCode: input.SourceCode, Code: input.Code, Name: input.Name, UnitType: unitType,
			Status: string(input.Status), SourceUpdatedAt: organizationHRTimePointer(input.SourceChangedAt), SourceVersion: input.SourceChangedAt.Format(time.RFC3339Nano),
			LastSyncAt: &batch.now, SourceStatus: string(input.Status), SourceDeleted: false, SyncStatus: "synced",
		}
		if err := s.repository.CreateOrgUnit(tx, &value); err != nil {
			return outcome, model.OrgUnit{}, false, err
		}
		outcome.localID, outcome.action = &value.Id, objectAction("", string(input.Status), true)
		return outcome, value, true, nil
	}
	if err != nil {
		return outcome, model.OrgUnit{}, false, err
	}
	if existing.UnitType != unitType {
		return failedOutcome(input.Key, kind, hrsync.ReasonSourceIDConflict), existing, false, nil
	}
	if stale, conflict := staleOrganizationFact(existing.SourceUpdatedAt, input.SourceChangedAt, orgUnitFactsEqual(existing, input, unitType)); stale {
		if conflict {
			return failedOutcome(input.Key, kind, hrsync.ReasonSourceIDConflict), existing, false, nil
		}
		outcome.localID, outcome.action = &existing.Id, model.OrgSyncRecordActionNoop
		return outcome, existing, existing.SourceUpdatedAt != nil && input.SourceChangedAt.Equal(*existing.SourceUpdatedAt), nil
	}
	byCode, codeErr := s.repository.FindOrgUnitByCode(tx, input.Key.SourceSystemCode(), input.Code)
	if codeErr == nil && byCode.Id != existing.Id {
		return failedOutcome(input.Key, kind, hrsync.ReasonSourceIDConflict), existing, false, nil
	}
	if codeErr != nil && !errors.Is(codeErr, gorm.ErrRecordNotFound) {
		return outcome, model.OrgUnit{}, false, codeErr
	}
	changed := !orgUnitFactsEqual(existing, input, unitType)
	if err := s.repository.UpdateOrgUnit(tx, existing.Id, map[string]any{
		"source_code": input.SourceCode, "code": input.Code, "name": input.Name, "unit_type": unitType,
		"status": string(input.Status), "source_updated_at": input.SourceChangedAt, "source_version": input.SourceChangedAt.Format(time.RFC3339Nano),
		"last_sync_at": batch.now, "source_status": string(input.Status), "source_deleted": false, "sync_status": "synced", "last_error": "",
	}); err != nil {
		return outcome, model.OrgUnit{}, false, err
	}
	previousStatus := existing.Status
	existing.SourceCode, existing.Code, existing.Name, existing.Status = input.SourceCode, input.Code, input.Name, string(input.Status)
	outcome.localID, outcome.action = &existing.Id, objectAction(previousStatus, string(input.Status), changed)
	return outcome, existing, true, nil
}

func (s *OrganizationHRSyncService) ensureStructureNodePlaceholder(tx *gorm.DB, structure model.OrgStructure, input hrsync.OrgUnitSyncInput, unit model.OrgUnit) error {
	sourceID, err := structureNodeSourceID(input.Key.SourceSystemCode(), structure.StructureType, input.Key.PersistenceID())
	if err != nil {
		return err
	}
	node, err := s.repository.FindStructureNodeBySource(tx, input.Key.SourceSystemCode(), sourceID)
	if err == nil {
		if node.StructureId != structure.Id || node.OrgUnitId != unit.Id {
			return ErrOrganizationHRSyncInvalid
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	id, err := s.nextID()
	if err != nil {
		return err
	}
	value := model.OrgStructureNode{
		Basic: model.Basic{Id: id, State: true}, StructureId: structure.Id, OrgUnitId: unit.Id,
		SourceSystemCode: input.Key.SourceSystemCode(), SourceId: sourceID, Path: fmt.Sprintf("/%d/", id), Level: 1,
		Sort: input.Sort, Status: "disabled", SourceDeleted: false, SyncStatus: "dependency_waiting",
	}
	return s.repository.CreateStructureNode(tx, &value)
}

func (s *OrganizationHRSyncService) ensureStructure(ctx context.Context, structureType string) (model.OrgStructure, error) {
	code, name := organizationManagementStructureCode, "HR 管理架构"
	if structureType == model.OrgStructureTypeLegal {
		code, name = organizationLegalStructureCode, "HR 法人架构"
	} else if structureType != model.OrgStructureTypeManagement {
		return model.OrgStructure{}, ErrOrganizationHRSyncInvalid
	}
	var result model.OrgStructure
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		value, err := s.repository.FindStructureByCode(tx, code)
		if err == nil {
			if value.StructureType != structureType {
				return ErrOrganizationHRSyncInvalid
			}
			result = value
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		id, err := s.nextID()
		if err != nil {
			return err
		}
		value = model.OrgStructure{
			Basic: model.Basic{Id: id, State: true}, Code: code, Name: name, StructureType: structureType,
			SourceSystemCode: organizationManagedSourceCode, Status: "enabled", IsDefault: structureType == model.OrgStructureTypeManagement,
			SyncStatus: "synced",
		}
		if err := s.repository.CreateStructure(tx, &value); err != nil {
			return err
		}
		result = value
		return nil
	})
	return result, err
}

func (s *OrganizationHRSyncService) resolveLegalEntityParents(
	ctx context.Context,
	batch organizationBusinessBatch,
	inputs []hrsync.LegalEntitySyncInput,
) ([]organizationSyncOutcome, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	tx := s.repository.DBWithContext(ctx)
	entities, err := s.repository.ListLegalEntities(tx, hrsync.OrganizationHRSourceSystemCode)
	if err != nil {
		return nil, err
	}
	bySource := make(map[string]model.OrgLegalEntity, len(entities))
	parents := make(map[int]*int, len(entities))
	for _, entity := range entities {
		bySource[entity.SourceId] = entity
		parents[entity.Id] = entity.ParentId
	}

	type relation struct {
		input    hrsync.LegalEntitySyncInput
		entityID int
		parentID *int
		reason   hrsync.ReasonCode
	}
	relations := make([]relation, 0, len(inputs))
	for _, input := range inputs {
		entity, ok := bySource[input.Key.PersistenceID()]
		if !ok {
			continue
		}
		relation := relation{input: input, entityID: entity.Id}
		parentSourceID := strings.TrimSpace(input.ParentSourceID)
		if parentSourceID != "" {
			if parentSourceID == input.Key.RawSourceID() {
				relation.reason = hrsync.ReasonParentSelfReference
			} else if parentKey, keyErr := hrsync.NewSourceKey(input.Key.SourceSystemCode(), hrsync.ObjectKindLegalEntity, parentSourceID); keyErr != nil {
				relation.reason = hrsync.ReasonParentInvalid
			} else if parent, exists := bySource[parentKey.PersistenceID()]; !exists {
				relation.reason = hrsync.ReasonParentUnresolved
			} else {
				parentID := parent.Id
				relation.parentID = &parentID
			}
		}
		if entity.SourceUpdatedAt != nil && input.SourceChangedAt.Equal(*entity.SourceUpdatedAt) && entity.SyncStatus == "synced" &&
			(relation.reason != "" || !organizationIntPointersEqual(entity.ParentId, relation.parentID)) {
			relation.reason = hrsync.ReasonSourceIDConflict
		}
		if relation.reason == "" {
			parents[entity.Id] = relation.parentID
		}
		relations = append(relations, relation)
	}
	cycleIDs := cyclicNodeIDs(parents)
	for index := range relations {
		if relations[index].reason == "" && cycleIDs[relations[index].entityID] {
			relations[index].reason = hrsync.ReasonParentCycle
		}
	}

	outcomes := make([]organizationSyncOutcome, 0)
	for start := 0; start < len(relations); start += hrsync.OrganizationHRSyncChunkSize {
		end := min(start+hrsync.OrganizationHRSyncChunkSize, len(relations))
		err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
			for _, relation := range relations[start:end] {
				outcome := successOutcome(relation.input.Key, hrsync.ObjectKindLegalEntity)
				if relation.reason != "" {
					outcome = relationFailureOutcome(relation.input.Key, hrsync.ObjectKindLegalEntity, relation.reason, "legal_entity_parent", relation.input.ParentSourceID)
					if relation.reason != hrsync.ReasonSourceIDConflict {
						if err := s.repository.UpdateLegalEntity(tx, relation.entityID, map[string]any{"sync_status": syncStatusForOutcome(outcome), "last_error": string(relation.reason)}); err != nil {
							return err
						}
					}
					if err := s.upsertSyncRecord(tx, batch.value, outcome); err != nil {
						return err
					}
					outcomes = append(outcomes, outcome)
					continue
				}
				if err := s.repository.UpdateLegalEntity(tx, relation.entityID, map[string]any{"parent_id": relation.parentID, "sync_status": "synced", "last_error": ""}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return outcomes, err
		}
	}
	return outcomes, nil
}

func (s *OrganizationHRSyncService) resolveStructureRelations(
	ctx context.Context,
	batch organizationBusinessBatch,
	structure model.OrgStructure,
	kind hrsync.ObjectKind,
	inputs []hrsync.OrgUnitSyncInput,
) ([]organizationSyncOutcome, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	tx := s.repository.DBWithContext(ctx)
	nodes, err := s.repository.ListStructureNodes(tx, structure.Id)
	if err != nil {
		return nil, err
	}
	allSourceNodes, err := s.repository.ListStructureNodesBySource(tx, hrsync.OrganizationHRSourceSystemCode)
	if err != nil {
		return nil, err
	}
	units, err := s.repository.ListOrgUnits(tx, hrsync.OrganizationHRSourceSystemCode)
	if err != nil {
		return nil, err
	}
	nodeBySource := make(map[string]model.OrgStructureNode, len(nodes))
	nodeByID := make(map[int]model.OrgStructureNode, len(nodes))
	unitByID := make(map[int]model.OrgUnit, len(units))
	for _, node := range nodes {
		nodeBySource[node.SourceId] = node
		nodeByID[node.Id] = node
	}
	allNodeSources := make(map[string]model.OrgStructureNode, len(allSourceNodes))
	for _, node := range allSourceNodes {
		allNodeSources[node.SourceId] = node
	}
	for _, unit := range units {
		unitByID[unit.Id] = unit
	}

	type relation struct {
		input       hrsync.OrgUnitSyncInput
		node        model.OrgStructureNode
		parentID    *int
		legalEntity *int
		reason      hrsync.ReasonCode
		dependency  string
	}
	relations := make([]relation, 0, len(inputs))
	desiredParents := make(map[int]*int, len(nodes))
	currentNodeIDs := make(map[int]bool, len(inputs))
	for _, node := range nodes {
		desiredParents[node.Id] = node.ParentNodeId
	}
	for _, input := range inputs {
		sourceID, sourceErr := structureNodeSourceID(input.Key.SourceSystemCode(), structure.StructureType, input.Key.PersistenceID())
		if sourceErr == nil {
			if node, exists := nodeBySource[sourceID]; exists {
				currentNodeIDs[node.Id] = true
			}
		}
	}
	for _, input := range inputs {
		sourceID, sourceErr := structureNodeSourceID(input.Key.SourceSystemCode(), structure.StructureType, input.Key.PersistenceID())
		node, exists := nodeBySource[sourceID]
		if sourceErr != nil || !exists {
			continue
		}
		relation := relation{input: input, node: node}
		if kind == hrsync.ObjectKindLegalUnit {
			legalID := strings.TrimSpace(input.LegalEntitySourceID)
			if legalID == "" {
				relation.reason, relation.dependency = hrsync.ReasonReferenceMissing, "legal_entity"
			} else if legalKey, keyErr := hrsync.NewSourceKey(input.Key.SourceSystemCode(), hrsync.ObjectKindLegalEntity, legalID); keyErr != nil {
				relation.reason, relation.dependency = hrsync.ReasonReferenceMissing, "legal_entity"
			} else if legalEntity, findErr := s.repository.FindLegalEntityBySource(tx, input.Key.SourceSystemCode(), legalKey.PersistenceID()); findErr != nil {
				if !errors.Is(findErr, gorm.ErrRecordNotFound) {
					return nil, findErr
				}
				relation.reason, relation.dependency = hrsync.ReasonReferenceMissing, "legal_entity"
			} else {
				legalEntityID := legalEntity.Id
				relation.legalEntity = &legalEntityID
			}
		}
		if relation.reason == "" {
			parentID, reason, dependency := resolveStructureParent(input, kind, structure, nodeBySource, allNodeSources, currentNodeIDs)
			relation.parentID, relation.reason, relation.dependency = parentID, reason, dependency
		}
		unit := unitByID[node.OrgUnitId]
		expectedParentSource := safeDependencyDigest(input.Key.SourceSystemCode(), kind, input.ParentSourceID)
		if unit.SourceUpdatedAt != nil && input.SourceChangedAt.Equal(*unit.SourceUpdatedAt) && node.SyncStatus == "synced" &&
			(relation.reason != "" || !organizationIntPointersEqual(node.ParentNodeId, relation.parentID) || node.SourceParentId != expectedParentSource) {
			relation.reason, relation.dependency = hrsync.ReasonSourceIDConflict, "structure_parent"
		}
		if relation.reason == "" {
			desiredParents[node.Id] = relation.parentID
		}
		relations = append(relations, relation)
	}
	cycleIDs := cyclicNodeIDs(desiredParents)
	for index := range relations {
		if relations[index].reason == "" && cycleIDs[relations[index].node.Id] {
			relations[index].reason = hrsync.ReasonParentCycle
			relations[index].dependency = "structure_parent"
		}
	}
	invalidNodes := make(map[int]bool, len(relations))
	for _, relation := range relations {
		if relation.reason != "" {
			invalidNodes[relation.node.Id] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for index := range relations {
			relation := &relations[index]
			if relation.reason == "" && relation.parentID != nil && invalidNodes[*relation.parentID] {
				relation.reason, relation.dependency = hrsync.ReasonParentUnresolved, "structure_parent"
				invalidNodes[relation.node.Id] = true
				changed = true
			}
		}
	}

	paths, levels := deriveStructurePaths(nodeByID, desiredParents)
	outcomes := make([]organizationSyncOutcome, 0)
	for start := 0; start < len(relations); start += hrsync.OrganizationHRSyncChunkSize {
		end := min(start+hrsync.OrganizationHRSyncChunkSize, len(relations))
		err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
			for _, relation := range relations[start:end] {
				outcome := successOutcome(relation.input.Key, kind)
				if relation.reason != "" {
					outcome = relationFailureOutcome(relation.input.Key, kind, relation.reason, relation.dependency, relation.input.ParentSourceID)
					if relation.reason != hrsync.ReasonSourceIDConflict {
						if err := s.repository.UpdateStructureNode(tx, relation.node.Id, map[string]any{
							"source_parent_id": safeDependencyDigest(relation.input.Key.SourceSystemCode(), kind, relation.input.ParentSourceID),
							"status":           "disabled", "sync_status": syncStatusForOutcome(outcome),
						}); err != nil {
							return err
						}
						if err := s.repository.UpdateOrgUnit(tx, relation.node.OrgUnitId, map[string]any{"sync_status": syncStatusForOutcome(outcome), "last_error": string(relation.reason)}); err != nil {
							return err
						}
					}
					if err := s.upsertSyncRecord(tx, batch.value, outcome); err != nil {
						return err
					}
					outcomes = append(outcomes, outcome)
					continue
				}
				unit := unitByID[relation.node.OrgUnitId]
				status := unit.Status
				if status != "enabled" {
					status = "disabled"
				}
				if err := s.repository.UpdateStructureNode(tx, relation.node.Id, map[string]any{
					"parent_node_id": relation.parentID, "source_parent_id": safeDependencyDigest(inputKeySourceSystem(relation.input.Key), kind, relation.input.ParentSourceID),
					"path": paths[relation.node.Id], "level": levels[relation.node.Id], "sort": relation.input.Sort,
					"status": status, "source_deleted": false, "sync_status": "synced",
				}); err != nil {
					return err
				}
				values := map[string]any{"sync_status": "synced", "last_error": ""}
				if relation.legalEntity != nil {
					values["primary_legal_entity_id"] = relation.legalEntity
				}
				if err := s.repository.UpdateOrgUnit(tx, relation.node.OrgUnitId, values); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return outcomes, err
		}
	}
	return outcomes, nil
}

func resolveStructureParent(
	input hrsync.OrgUnitSyncInput,
	kind hrsync.ObjectKind,
	structure model.OrgStructure,
	nodes map[string]model.OrgStructureNode,
	allNodes map[string]model.OrgStructureNode,
	currentNodeIDs map[int]bool,
) (*int, hrsync.ReasonCode, string) {
	parentRawID := strings.TrimSpace(input.ParentSourceID)
	if parentRawID == "" {
		return nil, "", ""
	}
	if parentRawID == input.Key.RawSourceID() {
		return nil, hrsync.ReasonParentSelfReference, "structure_parent"
	}
	candidateKinds := []hrsync.ObjectKind{kind}
	if kind == hrsync.ObjectKindManagementUnit {
		candidateKinds = []hrsync.ObjectKind{hrsync.ObjectKindManagementUnit, hrsync.ObjectKindManagementCompany}
	}
	var resolved *int
	for _, candidateKind := range candidateKinds {
		key, err := hrsync.NewSourceKey(input.Key.SourceSystemCode(), candidateKind, parentRawID)
		if err != nil {
			return nil, hrsync.ReasonParentInvalid, "structure_parent"
		}
		sourceID, err := structureNodeSourceID(input.Key.SourceSystemCode(), structure.StructureType, key.PersistenceID())
		if err != nil {
			return nil, hrsync.ReasonParentInvalid, "structure_parent"
		}
		if node, ok := nodes[sourceID]; ok {
			if node.SyncStatus != "synced" && !currentNodeIDs[node.Id] {
				return nil, hrsync.ReasonParentUnresolved, "structure_parent"
			}
			if resolved != nil && *resolved != node.Id {
				return nil, hrsync.ReasonParentInvalid, "structure_parent"
			}
			parentID := node.Id
			resolved = &parentID
		}
	}
	if resolved == nil {
		if structureParentExistsOutsideStructure(input, parentRawID, structure, allNodes) {
			return nil, hrsync.ReasonParentInvalid, "structure_parent"
		}
		return nil, hrsync.ReasonParentUnresolved, "structure_parent"
	}
	return resolved, "", ""
}

func structureParentExistsOutsideStructure(input hrsync.OrgUnitSyncInput, parentRawID string, structure model.OrgStructure, allNodes map[string]model.OrgStructureNode) bool {
	for _, candidateKind := range []hrsync.ObjectKind{hrsync.ObjectKindManagementCompany, hrsync.ObjectKindManagementUnit, hrsync.ObjectKindLegalUnit} {
		key, err := hrsync.NewSourceKey(input.Key.SourceSystemCode(), candidateKind, parentRawID)
		if err != nil {
			continue
		}
		for _, structureType := range []string{model.OrgStructureTypeManagement, model.OrgStructureTypeLegal} {
			sourceID, sourceErr := structureNodeSourceID(input.Key.SourceSystemCode(), structureType, key.PersistenceID())
			if sourceErr != nil {
				continue
			}
			if node, exists := allNodes[sourceID]; exists && node.StructureId != structure.Id {
				return true
			}
		}
	}
	return false
}

func cyclicNodeIDs(parents map[int]*int) map[int]bool {
	result := make(map[int]bool)
	state := make(map[int]uint8, len(parents))
	stack := make([]int, 0, len(parents))
	positions := make(map[int]int, len(parents))
	var visit func(int)
	visit = func(id int) {
		switch state[id] {
		case 1:
			if start, ok := positions[id]; ok {
				for _, cycleID := range stack[start:] {
					result[cycleID] = true
				}
			}
			return
		case 2:
			return
		}
		state[id] = 1
		positions[id] = len(stack)
		stack = append(stack, id)
		if parent := parents[id]; parent != nil {
			if _, exists := parents[*parent]; exists {
				visit(*parent)
			}
		}
		stack = stack[:len(stack)-1]
		delete(positions, id)
		state[id] = 2
	}
	for id := range parents {
		visit(id)
	}
	return result
}

func deriveStructurePaths(nodes map[int]model.OrgStructureNode, parents map[int]*int) (map[int]string, map[int]int) {
	paths := make(map[int]string, len(nodes))
	levels := make(map[int]int, len(nodes))
	visiting := make(map[int]bool)
	var derive func(int) (string, int)
	derive = func(id int) (string, int) {
		if path, ok := paths[id]; ok {
			return path, levels[id]
		}
		node := nodes[id]
		if visiting[id] {
			return node.Path, max(node.Level, 1)
		}
		visiting[id] = true
		path, level := fmt.Sprintf("/%d/", id), 1
		if parent := parents[id]; parent != nil {
			if _, exists := nodes[*parent]; exists {
				parentPath, parentLevel := derive(*parent)
				path, level = strings.TrimRight(parentPath, "/")+fmt.Sprintf("/%d/", id), parentLevel+1
			}
		}
		delete(visiting, id)
		paths[id], levels[id] = path, level
		return path, level
	}
	for id := range nodes {
		derive(id)
	}
	return paths, levels
}

func (s *OrganizationHRSyncService) finishBusinessBatch(
	ctx context.Context,
	batch organizationBusinessBatch,
	outcomes []organizationSyncOutcome,
) (hrsync.BusinessSyncSummary, error) {
	outcomes = compactOrganizationOutcomes(outcomes)
	summary := hrsync.BusinessSyncSummary{BatchNo: batch.value.BatchNo}
	for _, outcome := range outcomes {
		if outcome.status == "success" {
			summary.SuccessCount++
		} else {
			summary.FailedCount++
			if summary.ReasonCode == "" {
				summary.ReasonCode = outcome.reason
			}
		}
	}
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		for _, outcome := range outcomes {
			if err := s.upsertSyncRecord(tx, batch.value, outcome); err != nil {
				return err
			}
		}
		now, err := s.repository.CurrentDatabaseTime(tx)
		if err != nil {
			return err
		}
		status := "success"
		errorSummary := ""
		if summary.FailedCount > 0 {
			status = "failed"
			errorSummary = string(summary.ReasonCode)
		}
		return s.repository.UpdateSyncBatch(tx, batch.value.Id, map[string]any{
			"completed_at": now.UTC(), "total_count": len(outcomes), "success_count": summary.SuccessCount,
			"failed_count": summary.FailedCount, "skipped_count": 0, "status": status, "error_summary": errorSummary,
		})
	})
	if err != nil {
		return hrsync.BusinessSyncSummary{}, err
	}
	return summary, nil
}

func (s *OrganizationHRSyncService) upsertSyncRecord(tx *gorm.DB, batch model.OrgSyncBatch, outcome organizationSyncOutcome) error {
	if outcome.action == "" {
		outcome.action = model.OrgSyncRecordActionNoop
	}
	if outcome.status == "" {
		outcome.status = "success"
	}
	existing, err := s.repository.FindSyncRecordForUpdate(tx, batch.Id, string(outcome.kind), outcome.sourceSummary)
	values := map[string]any{
		"execution_id": batch.ExecutionId, "local_id": outcome.localID, "action": outcome.action, "status": outcome.status,
		"error_code": string(outcome.reason), "error_message": "", "dependency_type": outcome.dependencyType,
		"dependency_key": outcome.dependencyKey, "retry_count": 0, "last_retry_at": nil,
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, generateErr := s.nextID()
		if generateErr != nil {
			return generateErr
		}
		value := model.OrgSyncRecord{
			Basic: model.Basic{Id: id, State: true}, BatchId: batch.Id, ExecutionId: batch.ExecutionId,
			ObjectType: string(outcome.kind), SourceId: outcome.sourceSummary, LocalId: outcome.localID,
			Action: outcome.action, Status: outcome.status, ErrorCode: string(outcome.reason),
			DependencyType: outcome.dependencyType, DependencyKey: outcome.dependencyKey,
		}
		return s.repository.CreateSyncRecord(tx, &value)
	}
	if err != nil {
		return err
	}
	return s.repository.UpdateSyncRecord(tx, existing.Id, values)
}

func dedupeLegalEntityInputs(inputs []hrsync.LegalEntitySyncInput, issues []hrsync.SourceIssue) ([]hrsync.LegalEntitySyncInput, []hrsync.SourceIssue) {
	seen := make(map[string]hrsync.LegalEntitySyncInput, len(inputs))
	blocked := make(map[string]bool)
	order := make([]string, 0, len(inputs))
	for _, input := range inputs {
		identity := input.Key.PersistenceID()
		if blocked[identity] {
			continue
		}
		if existing, ok := seen[identity]; ok {
			if !reflect.DeepEqual(existing, input) {
				issues = append(issues, sourceConflictIssue(input.Key, hrsync.ObjectKindLegalEntity))
				delete(seen, identity)
				blocked[identity] = true
			}
			continue
		}
		seen[identity] = input
		order = append(order, identity)
	}
	result := make([]hrsync.LegalEntitySyncInput, 0, len(seen))
	for _, identity := range order {
		if input, ok := seen[identity]; ok {
			result = append(result, input)
		}
	}
	return result, issues
}

func dedupeOrgUnitInputs(inputs []hrsync.OrgUnitSyncInput, issues []hrsync.SourceIssue, kind hrsync.ObjectKind) ([]hrsync.OrgUnitSyncInput, []hrsync.SourceIssue) {
	seen := make(map[string]hrsync.OrgUnitSyncInput, len(inputs))
	blocked := make(map[string]bool)
	order := make([]string, 0, len(inputs))
	for _, input := range inputs {
		identity := input.Key.PersistenceID()
		if blocked[identity] {
			continue
		}
		if existing, ok := seen[identity]; ok {
			if !reflect.DeepEqual(existing, input) {
				issues = append(issues, sourceConflictIssue(input.Key, kind))
				delete(seen, identity)
				blocked[identity] = true
			}
			continue
		}
		seen[identity] = input
		order = append(order, identity)
	}
	result := make([]hrsync.OrgUnitSyncInput, 0, len(seen))
	for _, identity := range order {
		if input, ok := seen[identity]; ok {
			result = append(result, input)
		}
	}
	return result, issues
}

func sourceConflictIssue(key hrsync.SourceKey, kind hrsync.ObjectKind) hrsync.SourceIssue {
	return hrsync.SourceIssue{ObjectKind: kind, SourceIDSummary: key.Digest(), ReasonCode: hrsync.ReasonSourceIDConflict, Action: model.OrgSyncRecordActionError}
}

func sourceIssueOutcomes(issues []hrsync.SourceIssue, defaultKind hrsync.ObjectKind) []organizationSyncOutcome {
	result := make([]organizationSyncOutcome, 0, len(issues))
	for index, issue := range issues {
		kind := issue.ObjectKind
		if kind == "" {
			kind = defaultKind
		}
		summary := issue.SourceIDSummary
		if summary == "" {
			summary = fmt.Sprintf("issue-%d", index+1)
		}
		action := issue.Action
		if action == "" {
			action = model.OrgSyncRecordActionError
		}
		status := "failed"
		if action == model.OrgSyncRecordActionDeferred {
			status = "dependency_waiting"
		}
		result = append(result, organizationSyncOutcome{kind: kind, sourceSummary: summary, action: action, status: status, reason: issue.ReasonCode, dependencyType: issue.DependencyType, dependencyKey: issue.DependencyKey})
	}
	return result
}

func persistenceFailureOutcomes[T hrsync.LegalEntitySyncInput | hrsync.OrgUnitSyncInput](inputs []T, kind hrsync.ObjectKind) []organizationSyncOutcome {
	result := make([]organizationSyncOutcome, 0, len(inputs))
	for _, input := range inputs {
		var key hrsync.SourceKey
		switch typed := any(input).(type) {
		case hrsync.LegalEntitySyncInput:
			key = typed.Key
		case hrsync.OrgUnitSyncInput:
			key = typed.Key
		}
		result = append(result, failedOutcome(key, kind, hrsync.ReasonPersistenceFailed))
	}
	return result
}

func compactOrganizationOutcomes(outcomes []organizationSyncOutcome) []organizationSyncOutcome {
	result := make([]organizationSyncOutcome, 0, len(outcomes))
	positions := make(map[string]int, len(outcomes))
	for _, outcome := range outcomes {
		key := string(outcome.kind) + "\x00" + outcome.sourceSummary
		if index, ok := positions[key]; ok {
			if outcome.status != "success" || result[index].status == "success" {
				result[index] = outcome
			}
			continue
		}
		positions[key] = len(result)
		result = append(result, outcome)
	}
	return result
}

func mergeOrganizationOutcomes(base, updates []organizationSyncOutcome) []organizationSyncOutcome {
	return compactOrganizationOutcomes(append(base, updates...))
}

func successOutcome(key hrsync.SourceKey, kind hrsync.ObjectKind) organizationSyncOutcome {
	return organizationSyncOutcome{kind: kind, sourceSummary: key.Digest(), action: model.OrgSyncRecordActionNoop, status: "success"}
}

func failedOutcome(key hrsync.SourceKey, kind hrsync.ObjectKind, reason hrsync.ReasonCode) organizationSyncOutcome {
	return organizationSyncOutcome{kind: kind, sourceSummary: key.Digest(), action: model.OrgSyncRecordActionError, status: "failed", reason: reason}
}

func dependencyOutcome(key hrsync.SourceKey, kind hrsync.ObjectKind, reason hrsync.ReasonCode, dependencyType, dependencyRaw string) organizationSyncOutcome {
	return organizationSyncOutcome{
		kind: kind, sourceSummary: key.Digest(), action: model.OrgSyncRecordActionDeferred, status: "dependency_waiting", reason: reason,
		dependencyType: dependencyType, dependencyKey: safeDependencyDigest(key.SourceSystemCode(), kind, dependencyRaw),
	}
}

func relationFailureOutcome(key hrsync.SourceKey, kind hrsync.ObjectKind, reason hrsync.ReasonCode, dependencyType, dependencyRaw string) organizationSyncOutcome {
	if reason == hrsync.ReasonParentUnresolved || reason == hrsync.ReasonReferenceMissing {
		return dependencyOutcome(key, kind, reason, dependencyType, dependencyRaw)
	}
	outcome := failedOutcome(key, kind, reason)
	outcome.dependencyType = dependencyType
	outcome.dependencyKey = safeDependencyDigest(key.SourceSystemCode(), kind, dependencyRaw)
	return outcome
}

func syncStatusForOutcome(outcome organizationSyncOutcome) string {
	if outcome.status == "dependency_waiting" {
		return "dependency_waiting"
	}
	return "failed"
}

func safeDependencyDigest(sourceSystem string, kind hrsync.ObjectKind, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	key, err := hrsync.NewSourceKey(sourceSystem, kind, raw)
	if err != nil {
		return "invalid"
	}
	return key.Digest()
}

func structureNodeSourceID(sourceSystem, structureType, unitPersistenceID string) (string, error) {
	if strings.TrimSpace(sourceSystem) == "" || (structureType != model.OrgStructureTypeManagement && structureType != model.OrgStructureTypeLegal) || strings.TrimSpace(unitPersistenceID) == "" {
		return "", ErrOrganizationHRSyncInvalid
	}
	value := structureType + ":" + unitPersistenceID
	if len(value) <= hrsync.MaxRawSourceIDLength {
		return value, nil
	}
	digest := sha256.Sum256([]byte(value))
	return structureType + ":sha256:" + hex.EncodeToString(digest[:]), nil
}

func inputKeySourceSystem(key hrsync.SourceKey) string { return key.SourceSystemCode() }

func legalEntityFactsEqual(existing model.OrgLegalEntity, input hrsync.LegalEntitySyncInput) bool {
	return existing.SourceCode == input.SourceCode && existing.Code == input.Code && existing.Name == input.Name &&
		existing.ShortName == input.ShortName && existing.Status == string(input.Status)
}

func orgUnitFactsEqual(existing model.OrgUnit, input hrsync.OrgUnitSyncInput, unitType string) bool {
	return existing.SourceCode == input.SourceCode && existing.Code == input.Code && existing.Name == input.Name &&
		existing.UnitType == unitType && existing.Status == string(input.Status)
}

func staleOrganizationFact(existing *time.Time, incoming time.Time, equal bool) (bool, bool) {
	if existing == nil || incoming.After(*existing) {
		return false, false
	}
	return true, !equal
}

func objectAction(previousStatus, currentStatus string, changed bool) string {
	if previousStatus == "" {
		if currentStatus == "disabled" {
			return model.OrgSyncRecordActionDisable
		}
		return model.OrgSyncRecordActionCreate
	}
	if currentStatus == "disabled" && previousStatus != "disabled" {
		return model.OrgSyncRecordActionDisable
	}
	if changed {
		return model.OrgSyncRecordActionUpdate
	}
	return model.OrgSyncRecordActionNoop
}

func organizationIntPointersEqual(left, right *int) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}

func organizationHRTimePointer(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}

func (s *OrganizationHRSyncService) nextID() (int, error) {
	value, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, err
	}
	return int(value), nil
}
