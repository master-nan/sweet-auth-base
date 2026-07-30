package datapermission

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	ErrOwnershipFieldRegistrationInvalid      = errors.New("ownership field registration is invalid")
	ErrOwnershipFieldRegistrationDuplicate    = errors.New("ownership field registration already exists")
	ErrOwnershipFieldRegistrationNotFound     = errors.New("ownership field registration not found")
	ErrOwnershipFieldResourceMismatch         = errors.New("ownership field registration resource mismatch")
	ErrOwnershipFieldValueTypeMismatch        = errors.New("ownership field registration value type mismatch")
	ErrOwnershipFieldDimensionUnsupported     = errors.New("ownership field registration dimension unsupported")
	ErrOwnershipFieldOperationUnsupported     = errors.New("ownership field registration operation unsupported")
	ownershipFieldRegistrationCodePattern     = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._][a-z0-9]+)*$`)
	ownershipFieldRegistrationValueTypeLookup = map[string]struct{}{
		"bigint": {},
		"string": {},
	}
	ownershipFieldRegistrationOperationLookup = map[string]struct{}{
		"query":  {},
		"detail": {},
		"create": {},
		"update": {},
		"delete": {},
		"export": {},
		"run":    {},
	}
)

// OwnershipFieldRegistration is a server-owned declaration for one fixed
// business field. It contains no database expression and is never populated
// from an API request.
type OwnershipFieldRegistration struct {
	ResourceCode        string
	OwnershipCode       string
	AdapterFieldCode    string
	ValueType           string
	SupportedDimensions []string
	SupportedOperations []string
}

type OwnershipFieldBindingValidation struct {
	ResourceCode     string
	OwnershipCode    string
	AdapterFieldCode string
	ValueType        string
	DimensionCode    string
}

type OwnershipFieldOperationValidation struct {
	ResourceCode  string
	OwnershipCode string
	Operation     string
}

type OwnershipFieldBindingValidator interface {
	ValidateBinding(OwnershipFieldBindingValidation) error
}

// OwnershipFieldRegistry is an in-memory, process-local registry initialized
// from reviewed module declarations during application construction.
type OwnershipFieldRegistry struct {
	mu      sync.RWMutex
	entries map[string]ownershipFieldRegistrationEntry
}

type ownershipFieldRegistrationEntry struct {
	registration OwnershipFieldRegistration
	dimensions   map[string]struct{}
	operations   map[string]struct{}
}

func NewOwnershipFieldRegistry(
	registrations ...OwnershipFieldRegistration,
) (*OwnershipFieldRegistry, error) {
	registry := &OwnershipFieldRegistry{
		entries: make(map[string]ownershipFieldRegistrationEntry, len(registrations)),
	}
	for _, registration := range registrations {
		if err := registry.Register(registration); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *OwnershipFieldRegistry) Register(registration OwnershipFieldRegistration) error {
	if r == nil {
		return ErrOwnershipFieldRegistrationInvalid
	}
	normalized, err := normalizeOwnershipFieldRegistration(registration)
	if err != nil {
		return err
	}
	entry := ownershipFieldRegistrationEntry{
		registration: normalized,
		dimensions:   stringSet(normalized.SupportedDimensions),
		operations:   stringSet(normalized.SupportedOperations),
	}
	key := ownershipFieldRegistrationKey(
		normalized.ResourceCode,
		normalized.OwnershipCode,
		normalized.AdapterFieldCode,
	)

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.entries == nil {
		r.entries = make(map[string]ownershipFieldRegistrationEntry)
	}
	for existingKey, existing := range r.entries {
		if existingKey == key ||
			existing.registration.ResourceCode == normalized.ResourceCode &&
				(existing.registration.OwnershipCode == normalized.OwnershipCode ||
					existing.registration.AdapterFieldCode == normalized.AdapterFieldCode) {
			return ErrOwnershipFieldRegistrationDuplicate
		}
	}
	r.entries[key] = entry
	return nil
}

func (r *OwnershipFieldRegistry) ValidateBinding(
	validation OwnershipFieldBindingValidation,
) error {
	if r == nil {
		return ErrOwnershipFieldRegistrationNotFound
	}
	validation = normalizeOwnershipFieldBindingValidation(validation)

	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, exists := r.entries[ownershipFieldRegistrationKey(
		validation.ResourceCode,
		validation.OwnershipCode,
		validation.AdapterFieldCode,
	)]
	if !exists {
		if r.adapterFieldRegisteredForAnotherResource(validation) {
			return ErrOwnershipFieldResourceMismatch
		}
		return ErrOwnershipFieldRegistrationNotFound
	}
	if entry.registration.ValueType != validation.ValueType {
		return ErrOwnershipFieldValueTypeMismatch
	}
	if _, supported := entry.dimensions[validation.DimensionCode]; !supported {
		return ErrOwnershipFieldDimensionUnsupported
	}
	return nil
}

func (r *OwnershipFieldRegistry) ValidateOperation(
	validation OwnershipFieldOperationValidation,
) error {
	if r == nil {
		return ErrOwnershipFieldRegistrationNotFound
	}
	resourceCode := strings.TrimSpace(validation.ResourceCode)
	ownershipCode := strings.TrimSpace(validation.OwnershipCode)
	operation := strings.ToLower(strings.TrimSpace(validation.Operation))

	r.mu.RLock()
	defer r.mu.RUnlock()
	var matched bool
	for _, entry := range r.entries {
		if entry.registration.ResourceCode != resourceCode ||
			entry.registration.OwnershipCode != ownershipCode {
			continue
		}
		matched = true
		if _, supported := entry.operations[operation]; supported {
			return nil
		}
	}
	if !matched {
		return ErrOwnershipFieldRegistrationNotFound
	}
	return ErrOwnershipFieldOperationUnsupported
}

func (r *OwnershipFieldRegistry) adapterFieldRegisteredForAnotherResource(
	validation OwnershipFieldBindingValidation,
) bool {
	for _, entry := range r.entries {
		if entry.registration.AdapterFieldCode == validation.AdapterFieldCode &&
			entry.registration.OwnershipCode == validation.OwnershipCode &&
			entry.registration.ResourceCode != validation.ResourceCode {
			return true
		}
	}
	return false
}

func normalizeOwnershipFieldRegistration(
	registration OwnershipFieldRegistration,
) (OwnershipFieldRegistration, error) {
	registration.ResourceCode = strings.TrimSpace(registration.ResourceCode)
	registration.OwnershipCode = strings.TrimSpace(registration.OwnershipCode)
	registration.AdapterFieldCode = strings.TrimSpace(registration.AdapterFieldCode)
	registration.ValueType = strings.ToLower(strings.TrimSpace(registration.ValueType))
	registration.SupportedDimensions = normalizedUniqueValues(registration.SupportedDimensions)
	registration.SupportedOperations = normalizedUniqueValues(registration.SupportedOperations)

	if !ownershipFieldRegistrationCodePattern.MatchString(registration.ResourceCode) ||
		!ownershipFieldRegistrationCodePattern.MatchString(registration.OwnershipCode) ||
		!ownershipFieldRegistrationCodePattern.MatchString(registration.AdapterFieldCode) {
		return OwnershipFieldRegistration{}, fmt.Errorf(
			"%w: invalid registration code",
			ErrOwnershipFieldRegistrationInvalid,
		)
	}
	if _, ok := ownershipFieldRegistrationValueTypeLookup[registration.ValueType]; !ok {
		return OwnershipFieldRegistration{}, fmt.Errorf(
			"%w: unsupported value type",
			ErrOwnershipFieldRegistrationInvalid,
		)
	}
	if len(registration.SupportedDimensions) == 0 ||
		len(registration.SupportedOperations) == 0 {
		return OwnershipFieldRegistration{}, fmt.Errorf(
			"%w: dimensions and operations are required",
			ErrOwnershipFieldRegistrationInvalid,
		)
	}
	for _, value := range registration.SupportedDimensions {
		if !ownershipFieldRegistrationCodePattern.MatchString(value) {
			return OwnershipFieldRegistration{}, fmt.Errorf(
				"%w: invalid supported dimension",
				ErrOwnershipFieldRegistrationInvalid,
			)
		}
	}
	for _, operation := range registration.SupportedOperations {
		if _, supported := ownershipFieldRegistrationOperationLookup[operation]; !supported {
			return OwnershipFieldRegistration{}, fmt.Errorf(
				"%w: invalid supported operation",
				ErrOwnershipFieldRegistrationInvalid,
			)
		}
	}
	return registration, nil
}

func normalizeOwnershipFieldBindingValidation(
	validation OwnershipFieldBindingValidation,
) OwnershipFieldBindingValidation {
	validation.ResourceCode = strings.TrimSpace(validation.ResourceCode)
	validation.OwnershipCode = strings.TrimSpace(validation.OwnershipCode)
	validation.AdapterFieldCode = strings.TrimSpace(validation.AdapterFieldCode)
	validation.ValueType = strings.ToLower(strings.TrimSpace(validation.ValueType))
	validation.DimensionCode = strings.ToLower(strings.TrimSpace(validation.DimensionCode))
	return validation
}

func ownershipFieldRegistrationKey(resourceCode, ownershipCode, adapterFieldCode string) string {
	return resourceCode + "\x00" + ownershipCode + "\x00" + adapterFieldCode
}

func normalizedUniqueValues(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
