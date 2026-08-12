package impl

import (
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrganizationHRSyncRepositoryImpl struct {
	db *gorm.DB
}

var _ repository.OrganizationHRSyncRepository = (*OrganizationHRSyncRepositoryImpl)(nil)

func NewOrganizationHRSyncRepositoryImpl(primary *database.PrimaryDB) *OrganizationHRSyncRepositoryImpl {
	return &OrganizationHRSyncRepositoryImpl{db: primary.DB}
}

func (r *OrganizationHRSyncRepositoryImpl) DBWithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return r.db
	}
	return r.db.WithContext(ctx)
}

func (r *OrganizationHRSyncRepositoryImpl) CurrentDatabaseTime(tx *gorm.DB) (time.Time, error) {
	var now time.Time
	if tx.Dialector.Name() == "postgres" {
		return now, tx.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&now).Error
	}
	return time.Now().UTC(), nil
}

func (r *OrganizationHRSyncRepositoryImpl) LockSourceIdentity(tx *gorm.DB, identity string) error {
	if tx.Dialector.Name() != "postgres" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", identity).Error
}

func (r *OrganizationHRSyncRepositoryImpl) FindExecutionByNo(tx *gorm.DB, executionNo string) (model.IntegrationExecution, error) {
	var value model.IntegrationExecution
	err := tx.Where("execution_no = ?", executionNo).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) FindIntegrationSyncBatchByID(tx *gorm.DB, id int) (model.IntegrationSyncBatch, error) {
	var value model.IntegrationSyncBatch
	err := tx.First(&value, id).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) FindSyncBatchByExecutionForUpdate(tx *gorm.DB, executionID int) (model.OrgSyncBatch, error) {
	var value model.OrgSyncBatch
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("execution_id = ?", executionID).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) CreateSyncBatch(tx *gorm.DB, value *model.OrgSyncBatch) error {
	return tx.Create(value).Error
}

func (r *OrganizationHRSyncRepositoryImpl) UpdateSyncBatch(tx *gorm.DB, id int, values map[string]any) error {
	return tx.Model(&model.OrgSyncBatch{}).Where("id = ?", id).Updates(values).Error
}

func (r *OrganizationHRSyncRepositoryImpl) FindLegalEntityBySource(tx *gorm.DB, sourceSystem, sourceID string) (model.OrgLegalEntity, error) {
	var value model.OrgLegalEntity
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_system_code = ? AND source_id = ?", sourceSystem, sourceID).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) FindLegalEntityByCode(tx *gorm.DB, sourceSystem, code string) (model.OrgLegalEntity, error) {
	var value model.OrgLegalEntity
	err := tx.Where("source_system_code = ? AND code = ?", sourceSystem, code).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) CreateLegalEntity(tx *gorm.DB, value *model.OrgLegalEntity) error {
	return tx.Create(value).Error
}

func (r *OrganizationHRSyncRepositoryImpl) UpdateLegalEntity(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgLegalEntity{}, id, values, orgLegalEntitySourceFields)
}

func (r *OrganizationHRSyncRepositoryImpl) ListLegalEntities(tx *gorm.DB, sourceSystem string) ([]model.OrgLegalEntity, error) {
	var values []model.OrgLegalEntity
	err := tx.Where("source_system_code = ?", sourceSystem).Find(&values).Error
	return values, err
}

func (r *OrganizationHRSyncRepositoryImpl) FindOrgUnitBySource(tx *gorm.DB, sourceSystem, sourceID string) (model.OrgUnit, error) {
	var value model.OrgUnit
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_system_code = ? AND source_id = ?", sourceSystem, sourceID).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) FindOrgUnitByCode(tx *gorm.DB, sourceSystem, code string) (model.OrgUnit, error) {
	var value model.OrgUnit
	err := tx.Where("source_system_code = ? AND code = ?", sourceSystem, code).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) CreateOrgUnit(tx *gorm.DB, value *model.OrgUnit) error {
	return tx.Create(value).Error
}

func (r *OrganizationHRSyncRepositoryImpl) UpdateOrgUnit(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgUnit{}, id, values, orgUnitSourceFields)
}

func (r *OrganizationHRSyncRepositoryImpl) ListOrgUnits(tx *gorm.DB, sourceSystem string) ([]model.OrgUnit, error) {
	var values []model.OrgUnit
	err := tx.Where("source_system_code = ?", sourceSystem).Find(&values).Error
	return values, err
}

func (r *OrganizationHRSyncRepositoryImpl) FindPositionBySource(tx *gorm.DB, sourceSystem, sourceID string) (model.OrgPosition, error) {
	var value model.OrgPosition
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_system_code = ? AND source_id = ?", sourceSystem, sourceID).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) FindPositionByCode(tx *gorm.DB, sourceSystem, code string) (model.OrgPosition, error) {
	var value model.OrgPosition
	err := tx.Where("source_system_code = ? AND code = ?", sourceSystem, code).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) CreatePosition(tx *gorm.DB, value *model.OrgPosition) error {
	return tx.Create(value).Error
}

func (r *OrganizationHRSyncRepositoryImpl) UpdatePosition(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgPosition{}, id, values, orgPositionSourceFields)
}

func (r *OrganizationHRSyncRepositoryImpl) FindStructureByCode(tx *gorm.DB, code string) (model.OrgStructure, error) {
	var value model.OrgStructure
	err := tx.Where("code = ?", code).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) CreateStructure(tx *gorm.DB, value *model.OrgStructure) error {
	return tx.Create(value).Error
}

func (r *OrganizationHRSyncRepositoryImpl) FindStructureNodeBySource(tx *gorm.DB, sourceSystem, sourceID string) (model.OrgStructureNode, error) {
	var value model.OrgStructureNode
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_system_code = ? AND source_id = ?", sourceSystem, sourceID).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) CreateStructureNode(tx *gorm.DB, value *model.OrgStructureNode) error {
	return tx.Create(value).Error
}

func (r *OrganizationHRSyncRepositoryImpl) UpdateStructureNode(tx *gorm.DB, id int, values map[string]any) error {
	return updateOrganizationFields(tx, &model.OrgStructureNode{}, id, values, orgStructureNodeSourceFields)
}

func (r *OrganizationHRSyncRepositoryImpl) ListStructureNodes(tx *gorm.DB, structureID int) ([]model.OrgStructureNode, error) {
	var values []model.OrgStructureNode
	err := tx.Where("structure_id = ?", structureID).Find(&values).Error
	return values, err
}

func (r *OrganizationHRSyncRepositoryImpl) ListStructureNodesBySource(tx *gorm.DB, sourceSystem string) ([]model.OrgStructureNode, error) {
	var values []model.OrgStructureNode
	err := tx.Where("source_system_code = ?", sourceSystem).Find(&values).Error
	return values, err
}

func (r *OrganizationHRSyncRepositoryImpl) FindSyncRecordForUpdate(tx *gorm.DB, batchID int, objectType, sourceID string) (model.OrgSyncRecord, error) {
	var value model.OrgSyncRecord
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("batch_id = ? AND object_type = ? AND source_id = ?", batchID, objectType, sourceID).First(&value).Error
	return value, err
}

func (r *OrganizationHRSyncRepositoryImpl) CreateSyncRecord(tx *gorm.DB, value *model.OrgSyncRecord) error {
	return tx.Create(value).Error
}

func (r *OrganizationHRSyncRepositoryImpl) UpdateSyncRecord(tx *gorm.DB, id int, values map[string]any) error {
	return tx.Model(&model.OrgSyncRecord{}).Where("id = ?", id).Updates(values).Error
}
