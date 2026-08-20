package service

import (
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

type dataPermissionPreflightSnapshot struct {
	resources            map[int]model.DataResource
	policies             map[int]model.DataPolicy
	grants               map[int]model.DataGrant
	grantsByResource     map[int][]model.DataGrant
	grantsByPolicy       map[int][]model.DataGrant
	rulesByPolicy        map[int][]model.DataPolicyRule
	operations           map[string]model.DataResourceOperation
	ownerships           map[string]model.DataOwnershipField
	activeOwnershipCodes map[string]map[int]struct{}
	dimensions           map[int]model.DataDimensionDefinition
	activeSubjects       map[string]map[int]struct{}
}

func (v dataPermissionConfigValidator) loadPreflightSnapshotForGrant(tx *gorm.DB, grant model.DataGrant) (*dataPermissionPreflightSnapshot, error) {
	snapshot, err := v.loadPreflightSnapshot(tx, []int{grant.ResourceId}, []int{grant.PolicyId}, nil, false, false)
	if err != nil {
		return nil, err
	}
	snapshot.grants[grant.Id] = grant
	if err := v.loadActivePreflightSubjects(tx, snapshot, map[int]model.DataGrant{grant.Id: grant}); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (v dataPermissionConfigValidator) loadPreflightSnapshot(
	tx *gorm.DB,
	resourceIDs []int,
	policyIDs []int,
	grantIDs []int,
	loadResourceGrants bool,
	loadPolicyGrants bool,
) (*dataPermissionPreflightSnapshot, error) {
	snapshot := &dataPermissionPreflightSnapshot{
		resources: map[int]model.DataResource{}, policies: map[int]model.DataPolicy{}, grants: map[int]model.DataGrant{},
		grantsByResource: map[int][]model.DataGrant{}, grantsByPolicy: map[int][]model.DataGrant{},
		rulesByPolicy: map[int][]model.DataPolicyRule{}, operations: map[string]model.DataResourceOperation{},
		ownerships: map[string]model.DataOwnershipField{}, activeOwnershipCodes: map[string]map[int]struct{}{},
		dimensions: map[int]model.DataDimensionDefinition{}, activeSubjects: map[string]map[int]struct{}{},
	}
	resourceSet, policySet := preflightIntSet(resourceIDs), preflightIntSet(policyIDs)
	grantSet := map[int]model.DataGrant{}

	if err := v.loadSnapshotResources(tx, resourceSet, snapshot); err != nil {
		return nil, err
	}
	if err := v.loadSnapshotPolicies(tx, policySet, snapshot); err != nil {
		return nil, err
	}
	if len(grantIDs) > 0 {
		items, err := v.grantRepo.FindListByFieldInWithDB(tx, "id", grantIDs)
		if err != nil {
			return nil, err
		}
		for _, grant := range items {
			grantSet[grant.Id] = grant
		}
	}
	if loadResourceGrants && len(resourceSet) > 0 {
		items, err := v.grantRepo.FindListByFieldInWithDB(tx, "resource_id", setValues(resourceSet))
		if err != nil {
			return nil, err
		}
		for _, grant := range items {
			grantSet[grant.Id] = grant
		}
	}
	if loadPolicyGrants && len(policySet) > 0 {
		items, err := v.grantRepo.FindListByFieldInWithDB(tx, "policy_id", setValues(policySet))
		if err != nil {
			return nil, err
		}
		for _, grant := range items {
			grantSet[grant.Id] = grant
		}
	}
	for _, grant := range grantSet {
		snapshot.grants[grant.Id] = grant
		snapshot.grantsByResource[grant.ResourceId] = append(snapshot.grantsByResource[grant.ResourceId], grant)
		snapshot.grantsByPolicy[grant.PolicyId] = append(snapshot.grantsByPolicy[grant.PolicyId], grant)
		resourceSet[grant.ResourceId] = struct{}{}
		policySet[grant.PolicyId] = struct{}{}
	}
	if err := v.loadSnapshotResources(tx, resourceSet, snapshot); err != nil {
		return nil, err
	}
	if err := v.loadSnapshotPolicies(tx, policySet, snapshot); err != nil {
		return nil, err
	}

	rules, err := v.ruleRepo.FindListByFieldInWithDB(tx, "policy_id", setValues(policySet))
	if err != nil {
		return nil, err
	}
	dimensionSet := map[int]struct{}{}
	ownershipCodeSet := map[string]struct{}{}
	for _, rule := range rules {
		snapshot.rulesByPolicy[rule.PolicyId] = append(snapshot.rulesByPolicy[rule.PolicyId], rule)
		dimensionSet[rule.DimensionId] = struct{}{}
		ownershipCodeSet[rule.OwnershipCode] = struct{}{}
	}

	operations, err := v.operationRepo.FindListByFieldInWithDB(tx, "resource_id", setValues(resourceSet))
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		snapshot.operations[resourceOperationKey(operation.ResourceId, operation.Operation)] = operation
	}
	ownerships, err := v.ownershipRepo.FindListByFieldInWithDB(tx, "resource_id", setValues(resourceSet))
	if err != nil {
		return nil, err
	}
	for _, ownership := range ownerships {
		snapshot.ownerships[resourceOwnershipKey(ownership.ResourceId, ownership.OwnershipCode)] = ownership
		dimensionSet[ownership.DimensionId] = struct{}{}
	}
	activeOwnerships, err := v.ownershipRepo.ListActiveByOwnershipCodesForConfigDB(tx, stringSetValues(ownershipCodeSet))
	if err != nil {
		return nil, err
	}
	for _, ownership := range activeOwnerships {
		if snapshot.activeOwnershipCodes[ownership.OwnershipCode] == nil {
			snapshot.activeOwnershipCodes[ownership.OwnershipCode] = map[int]struct{}{}
		}
		snapshot.activeOwnershipCodes[ownership.OwnershipCode][ownership.DimensionId] = struct{}{}
	}
	if err := v.loadSnapshotDimensions(tx, dimensionSet, snapshot); err != nil {
		return nil, err
	}
	if err := v.loadActivePreflightSubjects(tx, snapshot, grantSet); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (v dataPermissionConfigValidator) loadSnapshotResources(tx *gorm.DB, ids map[int]struct{}, snapshot *dataPermissionPreflightSnapshot) error {
	missing := make([]int, 0, len(ids))
	for id := range ids {
		if _, loaded := snapshot.resources[id]; !loaded {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	items, err := v.resourceRepo.FindListByFieldInWithDB(tx, "id", missing)
	if err != nil {
		return err
	}
	for _, item := range items {
		snapshot.resources[item.Id] = item
	}
	return nil
}

func (v dataPermissionConfigValidator) loadSnapshotPolicies(tx *gorm.DB, ids map[int]struct{}, snapshot *dataPermissionPreflightSnapshot) error {
	missing := make([]int, 0, len(ids))
	for id := range ids {
		if _, loaded := snapshot.policies[id]; !loaded {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	items, err := v.policyRepo.FindListByFieldInWithDB(tx, "id", missing)
	if err != nil {
		return err
	}
	for _, item := range items {
		snapshot.policies[item.Id] = item
	}
	return nil
}

func (v dataPermissionConfigValidator) loadSnapshotDimensions(tx *gorm.DB, ids map[int]struct{}, snapshot *dataPermissionPreflightSnapshot) error {
	if len(ids) == 0 {
		return nil
	}
	items, err := v.dimensionRepo.FindListByFieldInWithDB(tx, "id", setValues(ids))
	if err != nil {
		return err
	}
	for _, item := range items {
		snapshot.dimensions[item.Id] = item
	}
	return nil
}

func (v dataPermissionConfigValidator) loadActivePreflightSubjects(tx *gorm.DB, snapshot *dataPermissionPreflightSnapshot, grants map[int]model.DataGrant) error {
	idsByType := map[string]map[int]struct{}{}
	for _, grant := range grants {
		if idsByType[grant.SubjectType] == nil {
			idsByType[grant.SubjectType] = map[int]struct{}{}
		}
		idsByType[grant.SubjectType][grant.SubjectId] = struct{}{}
	}
	for subjectType, ids := range idsByType {
		active, err := v.grantRepo.FindActiveSubjectIDsForConfigDB(tx, subjectType, setValues(ids))
		if err != nil {
			return err
		}
		snapshot.activeSubjects[subjectType] = preflightIntSet(active)
	}
	return nil
}

func preflightIntSet(values []int) map[int]struct{} {
	result := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value > 0 {
			result[value] = struct{}{}
		}
	}
	return result
}

func setValues(values map[int]struct{}) []int {
	result := make([]int, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return result
}

func stringSetValues(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func resourceOperationKey(resourceID int, operation string) string {
	return fmt.Sprintf("%d:%s", resourceID, operation)
}

func resourceOwnershipKey(resourceID int, ownershipCode string) string {
	return fmt.Sprintf("%d:%s", resourceID, ownershipCode)
}
