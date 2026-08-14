package errors

const (
	ErrorCodeOrgLegalEntityNotFound          = 110001
	ErrorCodeOrgLegalEntityCycle             = 110002
	ErrorCodeOrgStructureNotFound            = 110003
	ErrorCodeOrgStructureInactive            = 110004
	ErrorCodeOrgUnitNotFound                 = 110005
	ErrorCodeOrgStructureNodeMissing         = 110006
	ErrorCodeOrgStructureCycle               = 110007
	ErrorCodeOrgTreeRootAmbiguous            = 110008
	ErrorCodeOrgTreeTooLarge                 = 110009
	ErrorCodeOrgEmployeeNotFound             = 110010
	ErrorCodeOrgPositionNotFound             = 110011
	ErrorCodeOrgAssignmentNotFound           = 110012
	ErrorCodeOrgAssignmentTooLarge           = 110013
	ErrorCodeOrgUserNotFound                 = 110014
	ErrorCodeOrgEmployeeAlreadyBound         = 110015
	ErrorCodeOrgUserAlreadyBound             = 110016
	ErrorCodeOrgEmployeeInactive             = 110017
	ErrorCodeOrgLegalEntityInactive          = 110018
	ErrorCodeOrgUnitInactive                 = 110019
	ErrorCodeOrgStructureMembershipNotFound  = 110020
	ErrorCodeOrgStructureMembershipAmbiguous = 110021
	ErrorCodeOrgSyncBatchNotFound            = 110022
	ErrorCodeOrgSyncRecordNotFound           = 110023
)

var (
	ErrOrgLegalEntityNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgLegalEntityNotFound,
		"法人主体不存在",
	)
	ErrOrgLegalEntityCycle = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgLegalEntityCycle,
		"法人层级存在循环关系",
	)
	ErrOrgStructureNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgStructureNotFound,
		"管理架构不存在",
	)
	ErrOrgStructureInactive = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgStructureInactive,
		"管理架构在指定时间不可用",
	)
	ErrOrgUnitNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgUnitNotFound,
		"组织单元不存在",
	)
	ErrOrgStructureNodeMissing = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgStructureNodeMissing,
		"管理架构节点不存在",
	)
	ErrOrgStructureCycle = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgStructureCycle,
		"管理架构存在循环关系",
	)
	ErrOrgTreeRootAmbiguous = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgTreeRootAmbiguous,
		"组织单元在当前架构中对应多个节点，请使用structure_node_id定位",
	)
	ErrOrgTreeTooLarge = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeOrgTreeTooLarge,
		"组织树节点数量超过单次查询上限，请缩小查询范围",
	)
	ErrOrgEmployeeNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgEmployeeNotFound,
		"企业人员不存在",
	)
	ErrOrgPositionNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgPositionNotFound,
		"岗位不存在",
	)
	ErrOrgAssignmentNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgAssignmentNotFound,
		"任职记录不存在",
	)
	ErrOrgAssignmentResultTooLarge = newApplicationError(KindUnprocessable, CategoryBusiness,
		ErrorCodeOrgAssignmentTooLarge,
		"当前任职数量超过单次摘要上限",
	)
	ErrOrgUserNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgUserNotFound,
		"系统账号不存在",
	)
	ErrOrgEmployeeAlreadyBound = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgEmployeeAlreadyBound,
		"企业人员已绑定系统账号",
	)
	ErrOrgUserAlreadyBound = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgUserAlreadyBound,
		"系统账号已绑定其他企业人员",
	)
	ErrOrgEmployeeInactive = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgEmployeeInactive,
		"企业人员在指定时间不可用",
	)
	ErrOrgLegalEntityInactive = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgLegalEntityInactive,
		"法人主体在指定时间不可用",
	)
	ErrOrgUnitInactive = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgUnitInactive,
		"组织单元在指定时间不可用",
	)
	ErrOrgStructureMembershipNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgStructureMembershipNotFound,
		"组织单元不在指定管理架构中",
	)
	ErrOrgStructureMembershipAmbiguous = newApplicationError(KindConflict, CategoryBusiness,
		ErrorCodeOrgStructureMembershipAmbiguous,
		"组织单元在指定管理架构中存在多个有效节点",
	)
	ErrOrgSyncBatchNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgSyncBatchNotFound,
		"组织同步批次不存在",
	)
	ErrOrgSyncRecordNotFound = newApplicationError(KindNotFound, CategoryBusiness,
		ErrorCodeOrgSyncRecordNotFound,
		"组织同步记录不存在",
	)
)
