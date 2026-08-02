package service

import (
	"backend/internal/datapermission"
	"backend/model"
)

type policyResolverPolicyConfig struct {
	policy model.DataPolicy
	rules  []model.DataPolicyRule
}

func (config policyResolverPolicyConfig) clone() policyResolverPolicyConfig {
	config.rules = append([]model.DataPolicyRule(nil), config.rules...)
	return config
}

// policyResolverRequestContext is created for every Resolve call. Its caches
// cannot outlive or be shared across requests.
type policyResolverRequestContext struct {
	input              datapermission.ResolverInput
	resources          map[string]model.DataResource
	policies           map[int]policyResolverPolicyConfig
	dimensionValues    map[string]datapermission.DimensionValues
	checkedGrantCount  int
	checkedPolicyCount int
}

func newPolicyResolverRequestContext(
	input datapermission.ResolverInput,
) *policyResolverRequestContext {
	return &policyResolverRequestContext{
		input:           input,
		resources:       make(map[string]model.DataResource),
		policies:        make(map[int]policyResolverPolicyConfig),
		dimensionValues: make(map[string]datapermission.DimensionValues),
	}
}

func (request *policyResolverRequestContext) summary(
	result datapermission.DataScopeResult,
) (datapermission.ResolverSummary, error) {
	return datapermission.NewResolverSummary(datapermission.ResolverSummaryInput{
		ResourceCode:       request.input.ResourceCode(),
		Operation:          request.input.Operation(),
		Decision:           result.Decision(),
		CheckedGrantCount:  request.checkedGrantCount,
		CheckedPolicyCount: request.checkedPolicyCount,
	})
}
