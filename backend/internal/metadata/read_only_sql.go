package metadata

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrReadOnlyQueryEmpty     = errors.New("read-only query is empty")
	ErrReadOnlyQueryMultiple  = errors.New("read-only query must contain one statement")
	ErrReadOnlyQueryRequired  = errors.New("read-only query must start with select or with")
	ErrReadOnlyQueryForbidden = errors.New("read-only query contains a forbidden operation")

	readOnlySQLForbiddenPattern = regexp.MustCompile(`(?i)\b(insert|update|delete|merge|truncate|drop|alter|create|grant|revoke|replace|call|execute|exec|copy|vacuum|reindex|attach|detach|pragma|pg_sleep|benchmark|sleep)\b`)
	readOnlySQLForbiddenPhrase  = regexp.MustCompile(`(?i)\bexplain\s+analyze\b`)
)

// ValidateReadOnlyQuery accepts one SELECT/WITH statement and rejects SQL
// capable of changing database state. Callers still apply their own field and
// row authorization when executing the returned query.
func ValidateReadOnlyQuery(raw string) (string, error) {
	query := strings.TrimSpace(raw)
	if query == "" {
		return "", ErrReadOnlyQueryEmpty
	}
	if strings.Contains(query, ";") {
		return "", ErrReadOnlyQueryMultiple
	}
	lower := strings.ToLower(query)
	if !strings.HasPrefix(lower, "select") && !strings.HasPrefix(lower, "with") {
		return "", ErrReadOnlyQueryRequired
	}
	if readOnlySQLForbiddenPattern.MatchString(query) || readOnlySQLForbiddenPhrase.MatchString(query) {
		return "", ErrReadOnlyQueryForbidden
	}
	return query, nil
}
