package queryscheme

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Subject struct {
	UserID     int
	RoleIDs    []int
	EmployeeID *int
}

type QuickPreset struct {
	Code    string               `json:"code"`
	Label   string               `json:"label"`
	Payload QuerySchemePayloadV1 `json:"payload"`
}

type ScopeConfig struct {
	TableCode                string
	QuickDateField           string
	QuickPresets             []QuickPreset
	AllowedVirtualSortFields []string
	AllowedDynamicBindings   []BindingKind
}

func (config ScopeConfig) AllowsBinding(kind BindingKind) bool {
	for _, allowed := range config.AllowedDynamicBindings {
		if kind == allowed {
			return true
		}
	}
	return false
}

func (config ScopeConfig) AllowsSort(field string) bool {
	for _, allowed := range config.AllowedVirtualSortFields {
		if field == allowed {
			return true
		}
	}
	return false
}

type ScopeReader interface {
	Get(context.Context, string) (ScopeConfig, bool)
}

type Registry struct {
	mu      sync.RWMutex
	configs map[string]ScopeConfig
}

func NewRegistry() *Registry {
	registry := &Registry{configs: make(map[string]ScopeConfig)}
	for _, declaration := range FixedScopeDeclarations() {
		if err := registry.Register(declaration.ScopeCode, declaration.Config); err != nil {
			panic(err)
		}
	}
	return registry
}

func (registry *Registry) Register(scopeCode string, config ScopeConfig) error {
	scopeCode = strings.TrimSpace(scopeCode)
	if !ValidScopeCode(scopeCode) || strings.TrimSpace(config.TableCode) == "" {
		return fmt.Errorf("invalid query scope registration")
	}
	for _, binding := range config.AllowedDynamicBindings {
		if !binding.Valid() {
			return fmt.Errorf("invalid query scope binding registration")
		}
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.configs[scopeCode]; exists {
		return fmt.Errorf("duplicate query scope registration: %s", scopeCode)
	}
	config.AllowedDynamicBindings = append([]BindingKind(nil), config.AllowedDynamicBindings...)
	config.AllowedVirtualSortFields = append([]string(nil), config.AllowedVirtualSortFields...)
	config.QuickPresets = append([]QuickPreset(nil), config.QuickPresets...)
	registry.configs[scopeCode] = config
	return nil
}

func (registry *Registry) Get(_ context.Context, scopeCode string) (ScopeConfig, bool) {
	if registry == nil {
		return ScopeConfig{}, false
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	config, exists := registry.configs[strings.TrimSpace(scopeCode)]
	if !exists {
		return ScopeConfig{}, false
	}
	config.AllowedDynamicBindings = append([]BindingKind(nil), config.AllowedDynamicBindings...)
	config.AllowedVirtualSortFields = append([]string(nil), config.AllowedVirtualSortFields...)
	config.QuickPresets = append([]QuickPreset(nil), config.QuickPresets...)
	return config, true
}

func (registry *Registry) ScopeCodes() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	result := make([]string, 0, len(registry.configs))
	for code := range registry.configs {
		result = append(result, code)
	}
	sort.Strings(result)
	return result
}

type ScopeDeclaration struct {
	MenuName  string
	ScopeCode string
	Config    ScopeConfig
}

var dateBindings = []BindingKind{
	BindingToday, BindingStartOfWeek, BindingEndOfWeek, BindingStartOfMonth, BindingEndOfMonth,
}

func FixedScopeDeclarations() []ScopeDeclaration {
	dateAndUser := append(append([]BindingKind(nil), dateBindings...), BindingCurrentUser)
	dateUserEmployee := append(append([]BindingKind(nil), dateAndUser...), BindingCurrentEmployee)
	return []ScopeDeclaration{
		fixedScope("system_application", "system.application.list", "application", dateAndUser),
		fixedScope("system_user", "system.user.list", "sys_user", dateAndUser),
		fixedScope("system_role", "system.role.list", "sys_role", dateAndUser),
		fixedScope("system_sms", "system.sms.list", "sms_template", dateAndUser),
		fixedScope("system_audit", "system.audit.list", "access_log", dateAndUser),
		fixedScope("organization_employee", "organization.employee.list", "org_employee", dateUserEmployee, "gmt_modify"),
		fixedScope("organization_position", "organization.position.list", "org_position", dateUserEmployee, "gmt_modify"),
		fixedScope("organization_sync_batch", "organization.sync_batch.list", "org_sync_batch", dateAndUser, "gmt_modify"),
		fixedScope("organization_sync_error", "organization.sync_error.list", "org_sync_record", dateAndUser, "gmt_modify"),
		fixedScope("integration_external_system", "integration.external_system.list", "integration_external_system", dateAndUser),
		fixedScope("integration_interface_definition", "integration.interface_definition.list", "integration_interface_definition", dateAndUser),
		fixedScope("integration_credential", "integration.credential.list", "integration_credential", dateAndUser),
		fixedScope("integration_retry_policy", "integration.retry_policy.list", "integration_retry_policy", dateAndUser),
		fixedScope("integration_sync_task", "integration.sync_task.list", "integration_sync_task", dateAndUser),
		fixedScope("integration_sync_batch", "integration.sync_batch.list", "integration_sync_batch", dateAndUser),
		fixedScope("integration_execution", "integration.execution.list", "integration_execution", dateAndUser),
		fixedScope("integration_log", "integration.log.list", "integration_log", dateAndUser),
		fixedScope("develop_dictionary", "develop.dictionary.master", "sys_dict", dateAndUser),
	}
}

func fixedScope(menuName, scopeCode, tableCode string, bindings []BindingKind, virtualSortFields ...string) ScopeDeclaration {
	return ScopeDeclaration{
		MenuName:  menuName,
		ScopeCode: scopeCode,
		Config: ScopeConfig{
			TableCode:                tableCode,
			AllowedDynamicBindings:   append([]BindingKind(nil), bindings...),
			AllowedVirtualSortFields: append([]string(nil), virtualSortFields...),
		},
	}
}

func ValidScopeCode(value string) bool {
	if len(value) == 0 || len(value) > 128 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for index := range value {
		char := value[index]
		if (char >= 'a' && char <= 'z') || (index > 0 && char >= '0' && char <= '9') || char == '_' || char == '.' || char == '-' {
			continue
		}
		return false
	}
	return true
}
