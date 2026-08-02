package datapermission

import (
	"encoding/json"
	"strings"

	myerrors "backend/internal/errors"

	"github.com/gin-gonic/gin"
)

const resolverSummaryContextKey = "data_permission_resolver_summary"

// ResolverSummaryInput is the controlled construction boundary for one
// request-scoped resolution summary.
type ResolverSummaryInput struct {
	ResourceCode       string
	Operation          string
	Decision           DataScopeDecision
	CheckedGrantCount  int
	CheckedPolicyCount int
}

// ResolverSummary contains only safe orchestration diagnostics. Configuration
// IDs, subject details, database fields, and executable expressions are absent.
type ResolverSummary struct {
	resourceCode       string
	operation          string
	decision           DataScopeDecision
	checkedGrantCount  int
	checkedPolicyCount int
}

func NewResolverSummary(input ResolverSummaryInput) (ResolverSummary, error) {
	summary := ResolverSummary{
		resourceCode:       strings.TrimSpace(input.ResourceCode),
		operation:          strings.ToLower(strings.TrimSpace(input.Operation)),
		decision:           DataScopeDecision(strings.ToLower(strings.TrimSpace(string(input.Decision)))),
		checkedGrantCount:  input.CheckedGrantCount,
		checkedPolicyCount: input.CheckedPolicyCount,
	}
	if err := summary.Validate(); err != nil {
		return ResolverSummary{}, err
	}
	return summary, nil
}

func (summary ResolverSummary) Validate() error {
	if !dataScopeResourceCodePattern.MatchString(summary.resourceCode) {
		return myerrors.ErrDataPermissionResolverConfigConflict
	}
	if _, supported := dataScopeOperations[summary.operation]; !supported {
		return myerrors.ErrDataPermissionResolverConfigConflict
	}
	if !isDataScopeDecision(summary.decision) || summary.checkedGrantCount < 0 ||
		summary.checkedPolicyCount < 0 || summary.checkedPolicyCount > summary.checkedGrantCount {
		return myerrors.ErrDataPermissionResolverConfigConflict
	}
	return nil
}

func (summary ResolverSummary) ResourceCode() string {
	return summary.resourceCode
}

func (summary ResolverSummary) Operation() string {
	return summary.operation
}

func (summary ResolverSummary) Decision() DataScopeDecision {
	return summary.decision
}

func (summary ResolverSummary) CheckedGrantCount() int {
	return summary.checkedGrantCount
}

func (summary ResolverSummary) CheckedPolicyCount() int {
	return summary.checkedPolicyCount
}

func (summary ResolverSummary) MarshalJSON() ([]byte, error) {
	if err := summary.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ResourceCode       string            `json:"resource_code"`
		Operation          string            `json:"operation"`
		Decision           DataScopeDecision `json:"decision"`
		CheckedGrantCount  int               `json:"checked_grant_count"`
		CheckedPolicyCount int               `json:"checked_policy_count"`
	}{
		ResourceCode:       summary.resourceCode,
		Operation:          summary.operation,
		Decision:           summary.decision,
		CheckedGrantCount:  summary.checkedGrantCount,
		CheckedPolicyCount: summary.checkedPolicyCount,
	})
}

// StoreResolverSummary attaches a validated summary only to the current
// request. It deliberately provides no package-level or cross-request cache.
func StoreResolverSummary(ctx *gin.Context, summary ResolverSummary) error {
	if err := summary.Validate(); err != nil {
		return err
	}
	if ctx != nil {
		ctx.Set(resolverSummaryContextKey, summary)
	}
	return nil
}

func ResolverSummaryFromContext(ctx *gin.Context) (ResolverSummary, bool) {
	if ctx == nil {
		return ResolverSummary{}, false
	}
	value, exists := ctx.Get(resolverSummaryContextKey)
	if !exists {
		return ResolverSummary{}, false
	}
	summary, ok := value.(ResolverSummary)
	if !ok || summary.Validate() != nil {
		return ResolverSummary{}, false
	}
	return summary, true
}
