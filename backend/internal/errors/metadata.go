package errors

const (
	ErrorCodeTableExist             = 90001
	ErrorCodeTableFieldExist        = 90002
	ErrorCodeTableFieldNoChange     = 90003
	ErrorCodeTableInit              = 90004
	ErrorCodeTableViewSQLEmpty      = 90005
	ErrorCodeTableViewFieldNoAdd    = 90006
	ErrorCodeTableNotFound          = 90007
	ErrorCodeTableViewFieldNoDelete = 90008
	ErrorCodeTableRelationMigration = 90009
)

var (
	ErrTableExist             = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeTableExist, "表已存在")
	ErrTableFieldExist        = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeTableFieldExist, "字段已存在")
	ErrTableFieldNoChange     = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeTableFieldNoChange, "字段无变化，无需更新")
	ErrTableInit              = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeTableInit, "表已初始化，请勿重复操作")
	ErrTableViewSQLEmpty      = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeTableViewSQLEmpty, "视图类型视图SQL不能为空")
	ErrTableViewFieldNoAdd    = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeTableViewFieldNoAdd, "视图字段不可新增，请修改视图SQL后同步字段")
	ErrTableNotFound          = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeTableNotFound, "表不存在，请先初始化表元数据")
	ErrTableViewFieldNoDelete = newApplicationError(KindInvalidArgument, CategoryBusiness, ErrorCodeTableViewFieldNoDelete, "视图字段不可删除，请修改视图SQL后同步字段")
	ErrTableRelationMigration = newApplicationError(KindConflict, CategoryBusiness, ErrorCodeTableRelationMigration, "关系物理结构变更需要通过显式Migration完成")
)
