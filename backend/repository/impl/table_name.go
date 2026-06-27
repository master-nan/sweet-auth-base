package impl

import "gorm.io/gorm"

func tableNameForModel(tx *gorm.DB, model interface{}) string {
	stmt := &gorm.Statement{DB: tx}
	if err := stmt.Parse(model); err != nil || stmt.Schema == nil {
		return ""
	}
	return stmt.Schema.Table
}
