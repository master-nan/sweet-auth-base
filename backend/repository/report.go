package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"

	"gorm.io/gorm"
)

type ReportDefinitionRepository interface {
	BasicRepository[model.ReportDefinition]
	GetReportDefinitionList(*request.Basic, model.SysTable) (response.ListResult[model.ReportDefinition], error)
}

type ReportExecutionLogRepository interface {
	BasicRepository[model.ReportExecutionLog]
}

type ReportDefinitionVersionRepository interface {
	BasicRepository[model.ReportDefinitionVersion]
	GetMaxVersionNo(*gorm.DB, int) (int, error)
	FindByReportAndId(int, int) (model.ReportDefinitionVersion, error)
	ListByReportId(int) ([]model.ReportDefinitionVersion, error)
	ArchiveByReportId(*gorm.DB, int) error
}
