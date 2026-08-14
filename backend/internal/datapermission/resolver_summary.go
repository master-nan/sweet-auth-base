package datapermission

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	myerrors "backend/internal/errors"
)

type resolverSummaryContextKey struct{}

type resolverSummaryHolder struct {
	mu      sync.RWMutex
	summary ResolverSummary
	set     bool
}

// ResolverSummaryInput 是单次请求级解析摘要的受控构造边界。
type ResolverSummaryInput struct {
	ResourceCode       string
	Operation          string
	Decision           DataScopeDecision
	CheckedGrantCount  int
	CheckedPolicyCount int
}

// ResolverSummary 仅包含安全的编排诊断信息。
// 其中不包含配置 ID、主体明细、数据库字段或可执行表达式。
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

// WithResolverSummaryContext 为单次解析创建并发安全的诊断摘要容器。
// 容器只存储白名单计数和决策，不包含主体或配置明细。
func WithResolverSummaryContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, resolverSummaryContextKey{}, &resolverSummaryHolder{})
}

// StoreResolverSummary 仅将已校验摘要附加到当前标准 Context。
// 未安装摘要容器时安全忽略，不提供包级或跨请求缓存。
func StoreResolverSummary(ctx context.Context, summary ResolverSummary) error {
	if err := summary.Validate(); err != nil {
		return err
	}
	if ctx == nil {
		return nil
	}
	holder, ok := ctx.Value(resolverSummaryContextKey{}).(*resolverSummaryHolder)
	if !ok || holder == nil {
		return nil
	}
	holder.mu.Lock()
	holder.summary = summary
	holder.set = true
	holder.mu.Unlock()
	return nil
}

func ResolverSummaryFromContext(ctx context.Context) (ResolverSummary, bool) {
	if ctx == nil {
		return ResolverSummary{}, false
	}
	holder, ok := ctx.Value(resolverSummaryContextKey{}).(*resolverSummaryHolder)
	if !ok || holder == nil {
		return ResolverSummary{}, false
	}
	holder.mu.RLock()
	summary, set := holder.summary, holder.set
	holder.mu.RUnlock()
	if !set || summary.Validate() != nil {
		return ResolverSummary{}, false
	}
	return summary, true
}
