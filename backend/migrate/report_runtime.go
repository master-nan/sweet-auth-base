package main

import (
	"backend/enum"
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

const (
	legacyReportRuntimeComponent  = "pages/report-v2/runtime/ReportRuntimePage.vue"
	currentReportRuntimeComponent = "pages/report/runtime/ReportRuntimePage.vue"
)

func unifyReportRuntimeComponent(db *gorm.DB) error {
	if err := db.Model(&model.SysMenu{}).
		Where("page_type = ? AND component = ?", enum.MenuPageTypeReport, legacyReportRuntimeComponent).
		Update("component", currentReportRuntimeComponent).Error; err != nil {
		return fmt.Errorf("update published report runtime component: %w", err)
	}
	return nil
}
