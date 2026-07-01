package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
)

type ReportDefinitionRepositoryImpl struct {
	*BasicRepositoryImpl[model.ReportDefinition]
}

func NewReportDefinitionRepositoryImpl(primaryDB *database.PrimaryDB) *ReportDefinitionRepositoryImpl {
	return &ReportDefinitionRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.ReportDefinition{}),
	}
}

func (r *ReportDefinitionRepositoryImpl) GetReportDefinitionList(basic *request.Basic, table model.SysTable) (response.ListResult[model.ReportDefinition], error) {
	var result response.ListResult[model.ReportDefinition]
	var items []model.ReportDefinition
	total, err := r.PaginateAndCountAsync(basic, &items, table)
	result.Data = items
	result.Total = int(total)
	return result, err
}

type ReportExecutionLogRepositoryImpl struct {
	*BasicRepositoryImpl[model.ReportExecutionLog]
}

func NewReportExecutionLogRepositoryImpl(primaryDB *database.PrimaryDB) *ReportExecutionLogRepositoryImpl {
	return &ReportExecutionLogRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.ReportExecutionLog{}),
	}
}
