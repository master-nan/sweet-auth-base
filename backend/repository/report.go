package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
)

type ReportDefinitionRepository interface {
	BasicRepository[model.ReportDefinition]
	GetReportDefinitionList(*request.Basic, model.SysTable) (response.ListResult[model.ReportDefinition], error)
}

type ReportExecutionLogRepository interface {
	BasicRepository[model.ReportExecutionLog]
}
