package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"

	"gorm.io/gorm"
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

type ReportDefinitionVersionRepositoryImpl struct {
	*BasicRepositoryImpl[model.ReportDefinitionVersion]
}

func NewReportDefinitionVersionRepositoryImpl(primaryDB *database.PrimaryDB) *ReportDefinitionVersionRepositoryImpl {
	return &ReportDefinitionVersionRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.ReportDefinitionVersion{}),
	}
}

func (r *ReportDefinitionVersionRepositoryImpl) GetMaxVersionNo(tx *gorm.DB, reportId int) (int, error) {
	var maxVersionNo int
	err := tx.Model(&model.ReportDefinitionVersion{}).
		Where("report_id = ?", reportId).
		Select("COALESCE(MAX(version_no), 0)").
		Scan(&maxVersionNo).Error
	return maxVersionNo, err
}

func (r *ReportDefinitionVersionRepositoryImpl) FindByReportAndId(reportId int, versionId int) (model.ReportDefinitionVersion, error) {
	var version model.ReportDefinitionVersion
	err := r.db.Model(&model.ReportDefinitionVersion{}).
		Where("report_id = ? AND id = ?", reportId, versionId).
		First(&version).Error
	return version, err
}

func (r *ReportDefinitionVersionRepositoryImpl) ListByReportId(reportId int) ([]model.ReportDefinitionVersion, error) {
	var versions []model.ReportDefinitionVersion
	err := r.db.Model(&model.ReportDefinitionVersion{}).
		Where("report_id = ?", reportId).
		Order("version_no DESC").
		Find(&versions).Error
	return versions, err
}

func (r *ReportDefinitionVersionRepositoryImpl) ArchiveByReportId(tx *gorm.DB, reportId int) error {
	return tx.Model(&model.ReportDefinitionVersion{}).
		Where("report_id = ? AND status = ?", reportId, "published").
		Updates(map[string]any{
			"status": "archived",
		}).Error
}
