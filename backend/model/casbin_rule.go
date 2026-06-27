package model

// CasbinRule 对应 casbin_rule 表
// 说明：由 casbin gorm-adapter 使用
// https://github.com/casbin/gorm-adapter

type CasbinRule struct {
	Id    uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	PType string `gorm:"column:ptype;size:100;index:idx_casbin_rule" json:"ptype"`
	V0    string `gorm:"column:v0;size:100" json:"v0"`
	V1    string `gorm:"column:v1;size:100" json:"v1"`
	V2    string `gorm:"column:v2;size:100" json:"v2"`
	V3    string `gorm:"column:v3;size:100" json:"v3"`
	V4    string `gorm:"column:v4;size:100" json:"v4"`
	V5    string `gorm:"column:v5;size:100" json:"v5"`
}

func (CasbinRule) TableName() string {
	return "casbin_rule"
}
