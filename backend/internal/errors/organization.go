package errors

import "net/http"

const (
	ErrorCodeOrgLegalEntityNotFound  = 110001
	ErrorCodeOrgLegalEntityCycle     = 110002
	ErrorCodeOrgStructureNotFound    = 110003
	ErrorCodeOrgStructureInactive    = 110004
	ErrorCodeOrgUnitNotFound         = 110005
	ErrorCodeOrgStructureNodeMissing = 110006
	ErrorCodeOrgStructureCycle       = 110007
	ErrorCodeOrgTreeRootAmbiguous    = 110008
	ErrorCodeOrgTreeTooLarge         = 110009
	ErrorCodeOrgEmployeeNotFound     = 110010
	ErrorCodeOrgPositionNotFound     = 110011
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
	ErrOrgStructureNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeOrgStructureNotFound,
		"管理架构不存在",
	)
	ErrOrgStructureInactive = NewBusinessError(
		http.StatusConflict,
		ErrorCodeOrgStructureInactive,
		"管理架构在指定时间不可用",
	)
	ErrOrgUnitNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeOrgUnitNotFound,
		"组织单元不存在",
	)
	ErrOrgStructureNodeMissing = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeOrgStructureNodeMissing,
		"管理架构节点不存在",
	)
	ErrOrgStructureCycle = NewBusinessError(
		http.StatusConflict,
		ErrorCodeOrgStructureCycle,
		"管理架构存在循环关系",
	)
	ErrOrgTreeRootAmbiguous = NewBusinessError(
		http.StatusConflict,
		ErrorCodeOrgTreeRootAmbiguous,
		"组织单元在当前架构中对应多个节点，请使用structure_node_id定位",
	)
	ErrOrgTreeTooLarge = NewBusinessError(
		http.StatusUnprocessableEntity,
		ErrorCodeOrgTreeTooLarge,
		"组织树节点数量超过单次查询上限，请缩小查询范围",
	)
	ErrOrgEmployeeNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeOrgEmployeeNotFound,
		"企业人员不存在",
	)
	ErrOrgPositionNotFound = NewBusinessError(
		http.StatusNotFound,
		ErrorCodeOrgPositionNotFound,
		"岗位不存在",
	)
)
