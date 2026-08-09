package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

// RetryPolicyRepository 只保留版本、启用冲突和引用保护等领域查询。
type RetryPolicyRepository interface {
	BasicRepository[model.RetryPolicy]
	NextVersion(*gorm.DB, string) (int, error)
	HasEnabledVersion(*gorm.DB, string, int) (bool, error)
	CountEnabledInterfaceReferences(*gorm.DB, int) (int64, error)
}
