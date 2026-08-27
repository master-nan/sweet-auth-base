package request

// OrgReadScopeReq 定义组织镜像读取的共享可见性控制。
// OnlyEffective 为 nil 时由 OrgService 规范化为 true。
type OrgReadScopeReq struct {
	OnlyEffective   *bool  `form:"only_effective" json:"only_effective"`
	IncludeDisabled bool   `form:"include_disabled" json:"include_disabled"`
	IncludeHistory  bool   `form:"include_history" json:"include_history"`
	AsOfDate        string `form:"as_of_date" json:"as_of_date" binding:"omitempty,datetime=2006-01-02"`
}

// OrgSelectorOptionsReq 是 Organization 选择器选项 API 的共享请求协议。
// SelectedIds 仅用于回显：停用记录可以返回展示，但不能用于新选择。
type OrgSelectorOptionsReq struct {
	OrgReadScopeReq
	Page        int    `form:"page" json:"page"`
	Num         int    `form:"num" json:"num"`
	Keyword     string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
	SelectedIds []int  `form:"selected_ids" json:"selected_ids" binding:"omitempty,max=100,dive,gt=0"`
}

// OrgLegalEntityReadScopeReq 是法人主体 API 使用的读取范围请求。
type OrgLegalEntityReadScopeReq = OrgReadScopeReq

// OrgLegalEntityQueryReq 定义法人主体的 Repository 安全查询字段。
// 面向用户的查询 DTO 不包含来源身份和同步内部字段。
type OrgLegalEntityQueryReq struct {
	Basic
	OrgLegalEntityReadScopeReq
	SourceSystemCode string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	EntityType       string `form:"entity_type" json:"entity_type" binding:"omitempty,oneof=group legal_company branch accounting_unit"`
	ParentId         *int   `form:"parent_id" json:"parent_id" binding:"omitempty,gt=0"`
	Status           string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgLegalEntityDetailReq 控制指定内部 legal_entity_id 的可见性。
type OrgLegalEntityDetailReq struct {
	OrgLegalEntityReadScopeReq
}

// OrgLegalEntityTreeReq 请求仅按 org_legal_entity.parent_id 构建的法人主体树。
type OrgLegalEntityTreeReq struct {
	OrgLegalEntityReadScopeReq
	RootId *int `form:"root_id" json:"root_id" binding:"omitempty,gt=0"`
}

// OrgLegalEntityOptionsReq 支持远程选项搜索和已保存 ID 回显。
// SelectedIds 不改变选项值契约，Value 始终为内部 legal_entity_id。
type OrgLegalEntityOptionsReq struct {
	OrgSelectorOptionsReq
}

// OrgUnitQueryReq 定义管理组织的 Repository 安全查询字段。
type OrgUnitQueryReq struct {
	Basic
	OrgReadScopeReq
	SourceSystemCode     string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	UnitType             string `form:"unit_type" json:"unit_type" binding:"omitempty,oneof=business_unit region center department team project_group"`
	LegalEntityId        *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	PrimaryLegalEntityId *int   `form:"primary_legal_entity_id" json:"primary_legal_entity_id" binding:"omitempty,gt=0"`
	Status               string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgUnitDetailReq 控制指定内部 org_unit_id 的可见性。
type OrgUnitDetailReq struct {
	OrgReadScopeReq
}

// OrgUnitOptionsReq 支持远程搜索和已保存 org_unit_id 回显。
// StructureId 通过 org_structure_node 限制候选项。
type OrgUnitOptionsReq struct {
	OrgSelectorOptionsReq
	LegalEntityId *int `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	StructureId   *int `form:"structure_id" json:"structure_id" binding:"omitempty,gt=0"`
}

// OrgStructureQueryReq 定义法人架构和管理架构的 Repository 安全查询字段。
type OrgStructureQueryReq struct {
	Basic
	OrgReadScopeReq
	SourceSystemCode string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	LegalEntityId    *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	StructureType    string `form:"structure_type" json:"structure_type" binding:"omitempty,oneof=management legal"`
	Status           string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
	IsDefault        *bool  `form:"is_default" json:"is_default"`
}

// OrgStructureDetailReq 控制指定内部 structure_id 的可见性。
type OrgStructureDetailReq struct {
	OrgReadScopeReq
}

// OrgStructureOptionsReq 支持远程选项搜索和已保存 structure_id 回显。
type OrgStructureOptionsReq struct {
	OrgReadScopeReq
	Page          int    `form:"page" json:"page"`
	Num           int    `form:"num" json:"num"`
	Keyword       string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
	LegalEntityId *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	SelectedIds   []int  `form:"selected_ids" json:"selected_ids" binding:"omitempty,max=100,dive,gt=0"`
}

// OrgStructureOrgTreeReq 请求法人组织树或管理组织树。
// RootNodeId 标识架构中的一个节点；RootOrgUnitId 用于按业务组织便捷查找，
// 且必须精确解析为一个可见节点。
type OrgStructureOrgTreeReq struct {
	OrgReadScopeReq
	StructureId   int    `form:"structure_id" json:"structure_id" binding:"required,gt=0"`
	RootNodeId    *int   `form:"root_node_id" json:"root_node_id" binding:"omitempty,gt=0"`
	RootOrgUnitId *int   `form:"root_org_unit_id" json:"root_org_unit_id" binding:"omitempty,gt=0"`
	Keyword       string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
}

// OrgStructureNodeQueryReq 定义运行时组织树节点的 Repository 安全查询字段。
// Path 是内部查询加速字段，不接受普通客户端提交。
type OrgStructureNodeQueryReq struct {
	Basic
	StructureId  *int   `form:"structure_id" json:"structure_id" binding:"omitempty,gt=0"`
	OrgUnitId    *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	ParentNodeId *int   `form:"parent_node_id" json:"parent_node_id" binding:"omitempty,gt=0"`
	Status       string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgEmployeeQueryReq 不将手机号、邮箱、来源身份或来源版本作为普通高级查询字段开放。
// Repository 通过一条匹配的 org_assignment 记录解析关系过滤条件。
type OrgEmployeeQueryReq struct {
	Basic
	OrgReadScopeReq
	SourceSystemCode     string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	EmploymentStatus     string `form:"employment_status" json:"employment_status" binding:"omitempty,oneof=active probation suspended resigned retired"`
	PrimaryLegalEntityId *int   `form:"primary_legal_entity_id" json:"primary_legal_entity_id" binding:"omitempty,gt=0"`
	BoundUserId          *int   `form:"user_id" json:"user_id" binding:"omitempty,gt=0"`
	LegalEntityId        *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId            *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	PositionId           *int   `form:"position_id" json:"position_id" binding:"omitempty,gt=0"`
	BoundStatus          string `form:"bound_status" json:"bound_status" binding:"omitempty,oneof=all bound unbound"`
}

// OrgEmployeeDetailReq 控制指定内部 employee_id 的可见性。
type OrgEmployeeDetailReq struct {
	OrgReadScopeReq
}

// OrgEmployeeOptionsReq 支持远程搜索和已保存 employee_id 回显。
// Value 不使用 user_id、姓名或联系方式。
type OrgEmployeeOptionsReq struct {
	OrgSelectorOptionsReq
	LegalEntityId *int `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId     *int `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	PositionId    *int `form:"position_id" json:"position_id" binding:"omitempty,gt=0"`
}

// OrgEmployeeUserOptionsReq 为显式员工账号绑定流程提供远程账号查询。
// 它不接受联系方式字段或模糊身份匹配规则。
type OrgEmployeeUserOptionsReq struct {
	Page    int    `form:"page" json:"page" binding:"required,gt=0"`
	Num     int    `form:"num" json:"num" binding:"required,gt=0,lte=50"`
	Keyword string `form:"keyword" json:"keyword" binding:"omitempty,max=128"`
}

// OrgEmployeeBindUserReq 将显式选择的 Sweet Platform 账号绑定到员工。
// 任何账号属性都不能用于隐式匹配。
type OrgEmployeeBindUserReq struct {
	EmployeeId int `form:"-" json:"-" binding:"required,gt=0"`
	UserId     int `form:"user_id" json:"user_id" binding:"required,gt=0"`
}

// OrgEmployeeUnbindUserReq 通过路由标识员工。
// 解绑时不得根据账号名称或联系方式推断员工。
type OrgEmployeeUnbindUserReq struct {
	EmployeeId int `form:"-" json:"-" binding:"required,gt=0"`
}

// OrgPositionQueryReq 定义岗位的 Repository 安全查询字段。
type OrgPositionQueryReq struct {
	Basic
	OrgReadScopeReq
	SourceSystemCode  string `form:"source_system_code" json:"source_system_code" binding:"omitempty,max=64"`
	LegalEntityId     *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId         *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	PositionType      string `form:"position_type" json:"position_type" binding:"omitempty,oneof=management professional technical operation service"`
	IsManagerPosition *bool  `form:"is_manager_position" json:"is_manager_position"`
	Status            string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// OrgPositionDetailReq 控制指定内部 position_id 的可见性。
type OrgPositionDetailReq struct {
	OrgReadScopeReq
}

// OrgPositionOptionsReq 支持远程搜索和已保存 position_id 回显。
type OrgPositionOptionsReq struct {
	OrgSelectorOptionsReq
	LegalEntityId *int `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId     *int `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
}

const (
	OrgAssignmentScopeCurrent  = "current"
	OrgAssignmentScopeHistory  = "history"
	OrgAssignmentScopeFuture   = "future"
	OrgAssignmentScopeTimeline = "timeline"
)

// OrgAssignmentQueryReq 定义人员任职的只读查询。
// TimeScope 由 OrgService 规范化为 current，不选择主任职，也不特殊处理第一条任职。
type OrgAssignmentQueryReq struct {
	Basic
	EmployeeId     *int   `form:"employee_id" json:"employee_id" binding:"required,gt=0"`
	LegalEntityId  *int   `form:"legal_entity_id" json:"legal_entity_id" binding:"omitempty,gt=0"`
	OrgUnitId      *int   `form:"org_unit_id" json:"org_unit_id" binding:"omitempty,gt=0"`
	PositionId     *int   `form:"position_id" json:"position_id" binding:"omitempty,gt=0"`
	AssignmentType string `form:"assignment_type" json:"assignment_type" binding:"omitempty,oneof=standard primary part_time temporary project"`
	IsPrimary      *bool  `form:"is_primary" json:"is_primary"`
	IsManager      *bool  `form:"is_manager" json:"is_manager"`
	Status         string `form:"status" json:"status" binding:"omitempty,oneof=enabled disabled"`
	TimeScope      string `form:"time_scope" json:"time_scope" binding:"omitempty,oneof=current history future timeline"`
	AsOfDate       string `form:"as_of_date" json:"as_of_date" binding:"omitempty,datetime=2006-01-02"`
}

// OrgAssignmentDetailReq 有意保持为空，任职详情仅通过 Sweet Platform 内部 assignment_id 定位。
type OrgAssignmentDetailReq struct{}

// OrgEmployeeCurrentAssignmentSummaryReq 选择有效日期快照，
// 用于聚合员工当前的全部组织和岗位。
type OrgEmployeeCurrentAssignmentSummaryReq struct {
	AsOfDate string `form:"as_of_date" json:"as_of_date" binding:"omitempty,datetime=2006-01-02"`
}

// OrgSyncBatchQueryReq 是组织领域同步批次的只读查询契约。
// 集成载荷字段不属于此 DTO。
type OrgSyncBatchQueryReq struct {
	Basic
	ExecutionId *int   `form:"execution_id" json:"execution_id" binding:"omitempty,gt=0"`
	SyncType    string `form:"sync_type" json:"sync_type" binding:"omitempty,oneof=full incremental manual_retry"`
	ObjectScope string `form:"object_scope" json:"object_scope" binding:"omitempty,max=64"`
	Status      string `form:"status" json:"status" binding:"omitempty,oneof=pending processing success failed dependency_waiting ignored"`
}

// OrgSyncRecordQueryReq 开放诊断分类字段，但不开放原始来源身份、依赖键或错误消息。
type OrgSyncRecordQueryReq struct {
	Basic
	BatchId             *int   `form:"batch_id" json:"batch_id" binding:"omitempty,gt=0"`
	ExecutionId         *int   `form:"execution_id" json:"execution_id" binding:"omitempty,gt=0"`
	ObjectType          string `form:"object_type" json:"object_type" binding:"omitempty,max=64"`
	LocalId             *int   `form:"local_id" json:"local_id" binding:"omitempty,gt=0"`
	Action              string `form:"action" json:"action" binding:"omitempty,oneof=create update disable close noop error deferred"`
	Status              string `form:"status" json:"status" binding:"omitempty,oneof=pending processing success failed dependency_waiting ignored"`
	DependencyType      string `form:"dependency_type" json:"dependency_type" binding:"omitempty,oneof=legal_entity org_unit structure_node employee position assignment"`
	LocalHandlingStatus string `form:"local_handling_status" json:"local_handling_status" binding:"omitempty,max=32"`
}
