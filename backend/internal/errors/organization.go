package errors

import "net/http"

const (
	ErrorCodeOrgLegalEntityNotFound = 110001
	ErrorCodeOrgLegalEntityCycle    = 110002
)

var (
	ErrOrgLegalEntityNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeOrgLegalEntityNotFound,
		"法人主体不存在",
	)
	ErrOrgLegalEntityCycle = NewBusinessError(
		http.StatusConflict,
		ErrorCodeOrgLegalEntityCycle,
		"法人层级存在循环关系",
	)
)
