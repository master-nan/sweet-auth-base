package service

import (
	"backend/dto/request"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/security"
	"backend/model"
	"backend/repository"
	"context"
	"fmt"
	"maps"

	"gorm.io/gorm"
)

type DocumentSection struct {
	Relation model.SysTableRelation
	Table    model.SysTable
	Changes  request.DocumentChanges
}

func (gs *GeneralizationService) ResolveDocumentSection(ctx context.Context, master model.SysTable, relationID int) (DocumentSection, error) {
	for _, relation := range master.TableRelations {
		if relation.Id != relationID || relation.DocumentTab == "" || relation.RelationType != enum.OneToMany {
			continue
		}
		child, err := gs.metadataRuntime.GetTableByID(ctx, relation.RelatedTableId)
		if err != nil {
			return DocumentSection{}, err
		}
		return DocumentSection{Relation: relation, Table: child.QueryModel()}, nil
	}
	return DocumentSection{}, myerrors.NewValidationError("单据明细关系无效，请刷新页面")
}

func (gs *GeneralizationService) ReadDocumentRows(ctx context.Context, section DocumentSection, master map[string]interface{}, forUpdate bool) ([]map[string]interface{}, error) {
	repo, ok := gs.generalizationRepo.(repository.DocumentRepository)
	if !ok {
		return nil, myerrors.ErrDataPermissionRuntimeFailed
	}
	value := master[section.Relation.ReferenceKey]
	if value == nil || value == "" {
		return []map[string]interface{}{}, nil
	}
	access, err := gs.ResolveDataPermission(ctx, section.Table, model.DataPermissionOperationDetail)
	if err != nil {
		return nil, err
	}
	rows, err := repo.DocumentRows(ctx, section.Table, section.Relation.ForeignKey, value, access)
	if err != nil {
		return nil, err
	}
	visible := map[string]bool{"id": true}
	for _, f := range section.Table.TableFields {
		if f.FieldCode == section.Relation.ForeignKey || f.IsPrimaryKey || security.IsManagedMetadataField(f.FieldCode) || security.IsSensitiveFieldName(f.FieldCode) {
			continue
		}
		if forUpdate {
			visible[f.FieldCode] = f.IsUpdateShow
		} else if f.DetailVisible != nil {
			visible[f.FieldCode] = *f.DetailVisible
		} else {
			visible[f.FieldCode] = f.IsInsertShow || f.IsUpdateShow
		}
	}
	for _, row := range rows {
		maps.DeleteFunc(row, func(k string, _ any) bool { return !visible[k] })
	}
	return rows, nil
}

func (gs *GeneralizationService) SaveDocument(ctx context.Context, table model.SysTable, id int, data map[string]interface{}, sections []DocumentSection) (int, error) {
	repo, ok := gs.generalizationRepo.(repository.DocumentRepository)
	if !ok {
		return 0, myerrors.ErrDataPermissionRuntimeFailed
	}
	err := RunInTransaction(ctx, repo.DBWithContext(ctx), func(tx *gorm.DB) error {
		records := repo.WithDocumentDB(tx)
		service := *gs
		service.generalizationRepo, service.permissionRepo = records, records
		if id == 0 {
			newID, err := service.createRow(ctx, table, data, nil)
			if err != nil {
				return err
			}
			id = newID
		} else {
			access, err := service.ResolveDataPermission(ctx, table, model.DataPermissionOperationDetail)
			if err != nil {
				return err
			}
			original, err := records.LockDocumentRow(ctx, table, id, nil, access)
			if err != nil {
				return err
			}
			for _, relation := range table.TableRelations {
				if relation.DocumentTab == "" {
					continue
				}
				key := relation.ReferenceKey
				if v, exists := data[key]; exists && fmt.Sprint(v) != fmt.Sprint(original[key]) {
					return myerrors.NewValidationError("单据关联键不能在编辑时更改")
				}
			}
			if err := service.UpdateWithDataPermission(ctx, table, id, data); err != nil {
				return err
			}
		}
		master, err := records.GetById(ctx, table, id)
		if err != nil {
			return err
		}
		for _, section := range sections {
			if err := service.saveDocumentSection(ctx, records, section, master); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (gs *GeneralizationService) saveDocumentSection(ctx context.Context, records repository.DocumentRepository, section DocumentSection, master map[string]interface{}) error {
	relation, child, changes := section.Relation, section.Table, section.Changes
	key := master[relation.ReferenceKey]
	if key == nil || key == "" {
		return myerrors.NewValidationError("主表关联字段不能为空")
	}
	linked := map[string]interface{}{relation.ForeignKey: key}
	for _, row := range changes.Added {
		if _, err := gs.createRow(ctx, child, row, linked); err != nil {
			return err
		}
	}
	if len(changes.Updated)+len(changes.Deleted) == 0 {
		return nil
	}
	access, err := gs.ResolveDataPermission(ctx, child, model.DataPermissionOperationDetail)
	if err != nil {
		return err
	}
	seen := make(map[int]bool)
	lock := func(id int) error {
		if id <= 0 || seen[id] {
			return myerrors.ErrParamInvalid
		}
		seen[id] = true
		_, err := records.LockDocumentRow(ctx, child, id, linked, access)
		return err
	}
	for _, row := range changes.Updated {
		if err := lock(row.ID); err != nil {
			return err
		}
		data := maps.Clone(row.Data)
		delete(data, relation.ForeignKey)
		if err := gs.UpdateWithDataPermission(ctx, child, row.ID, data); err != nil {
			return err
		}
	}
	for _, id := range changes.Deleted {
		if err := lock(id); err != nil {
			return err
		}
		if err := gs.DeleteWithDataPermission(ctx, child, id); err != nil {
			return err
		}
	}
	return nil
}
