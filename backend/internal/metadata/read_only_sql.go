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

// ValidateReadOnlyQuery 只接受单条SELECT/WITH语句，并拒绝任何可能改变数据库状态的SQL。
// 调用方执行查询时仍须自行应用字段白名单和行级权限。
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
