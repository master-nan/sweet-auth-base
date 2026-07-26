package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/database"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository/impl"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestOrgServiceQueryLegalEntitiesUsesPlatformQueryAndEffectiveScope(t *testing.T) {
	service, db := newOrgServiceTestSubject(t)
	now := model.Now()
	past := now.AddDate(0, 0, -30)
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	fixtures := []model.OrgLegalEntity{
		orgServiceLegalEntity(1, "LE-001", "Alpha Legal", "Alpha", "enabled", nil, nil),
		orgServiceLegalEntity(2, "LE-002", "Beta Legal", "Beta", "enabled", nil, nil),
		orgServiceLegalEntity(3, "LE-003", "Disabled Legal", "Disabled", "disabled", nil, nil),
		orgServiceLegalEntity(4, "LE-004", "Expired Legal", "Expired", "enabled", &past, &yesterday),
		orgServiceLegalEntity(5, "LE-005", "Future Legal", "Future", "enabled", &tomorrow, nil),
		orgServiceLegalEntity(6, "LE-006", "Deleted Legal", "Deleted", "enabled", nil, nil),
	}
	fixtures[5].SourceDeleted = true
	testutil.MustCreate(t, db, &fixtures)

	result, err := service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
		Basic: request.Basic{
			Page: 1,
			Num:  1,
			Order: request.Order{
				Field: "code",
				IsAsc: true,
			},
		},
	}, orgServiceLegalEntityTable())
	if err != nil {
		t.Fatalf("query default legal entities: %v", err)
	}
	if result.Total != 2 || len(result.Data) != 1 || result.Data[0].Id != fixtures[0].Id {
		t.Fatalf("unexpected default query result: %+v", result)
	}

	quickResult, err := service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
		Basic: request.Basic{
			QuickQuery: &request.QuickQuery{Keyword: "Alpha"},
		},
	}, orgServiceLegalEntityTable())
	if err != nil {
		t.Fatalf("quick query legal entities: %v", err)
	}
	if quickResult.Total != 1 || quickResult.Data[0].Id != fixtures[0].Id {
		t.Fatalf("unexpected quick query result: %+v", quickResult)
	}

	expressionResult, err := service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
		Basic: request.Basic{
			Expressions: []request.ExpressionGroup{{
				Logic: enum.And,
				Rules: []request.QueryRule{{
					Field:          "short_name",
					ExpressionType: enum.Like,
					Value:          "Bet",
				}},
			}},
		},
	}, orgServiceLegalEntityTable())
	if err != nil {
		t.Fatalf("advanced query legal entities: %v", err)
	}
	if expressionResult.Total != 1 || expressionResult.Data[0].Id != fixtures[1].Id {
		t.Fatalf("unexpected advanced query result: %+v", expressionResult)
	}

	disabledResult, err := service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
		OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{
			IncludeDisabled: true,
		},
	}, orgServiceLegalEntityTable())
	if err != nil {
		t.Fatalf("include disabled legal entities: %v", err)
	}
	if disabledResult.Total != 3 {
		t.Fatalf("include_disabled total = %d, want 3", disabledResult.Total)
	}

	historyResult, err := service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
		OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{
			IncludeHistory: true,
		},
	}, orgServiceLegalEntityTable())
	if err != nil {
		t.Fatalf("include legal entity history: %v", err)
	}
	if historyResult.Total != 5 {
		t.Fatalf("include_history total = %d, want 5", historyResult.Total)
	}

	allResult, err := service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
		OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{
			OnlyEffective: orgServiceBoolPointer(false),
		},
	}, orgServiceLegalEntityTable())
	if err != nil {
		t.Fatalf("query explicit non-effective scope: %v", err)
	}
	if allResult.Total != len(fixtures) {
		t.Fatalf("only_effective=false total = %d, want %d", allResult.Total, len(fixtures))
	}
}

func TestOrgServiceQueryLegalEntitiesHonorsAsOfDateInclusiveBoundaries(t *testing.T) {
	service, db := newOrgServiceTestSubject(t)
	validFrom := orgServiceDate(t, "2026-01-01")
	validTo := orgServiceDate(t, "2026-01-31")
	entity := orgServiceLegalEntity(
		1,
		"LE-BOUNDARY",
		"Boundary Legal",
		"Boundary",
		"enabled",
		&validFrom,
		&validTo,
	)
	testutil.MustCreate(t, db, &entity)

	for _, asOf := range []string{"2026-01-01", "2026-01-31"} {
		result, err := service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
			OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{AsOfDate: asOf},
		}, orgServiceLegalEntityTable())
		if err != nil {
			t.Fatalf("query as of %s: %v", asOf, err)
		}
		if result.Total != 1 {
			t.Fatalf("as_of_date %s total = %d, want 1", asOf, result.Total)
		}
	}

	result, err := service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
		OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{AsOfDate: "2026-02-01"},
	}, orgServiceLegalEntityTable())
	if err != nil {
		t.Fatalf("query after valid_to: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("post-validity total = %d, want 0", result.Total)
	}

	_, err = service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
		OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{AsOfDate: "2026/01/01"},
	}, orgServiceLegalEntityTable())
	assertOrgServiceAdminError(t, err, response.ErrorCategoryParameter, apperrors.ErrorCodeParamInvalid)
}

func TestOrgServiceQueryLegalEntitiesNeverExposesPlatformSoftDeletedRows(t *testing.T) {
	service, db := newOrgServiceTestSubject(t)
	entity := orgServiceLegalEntity(1, "LE-DELETED", "Soft Deleted", "Deleted", "enabled", nil, nil)
	testutil.MustCreate(t, db, &entity)
	if err := db.Delete(&entity).Error; err != nil {
		t.Fatalf("soft delete legal entity fixture: %v", err)
	}

	result, err := service.QueryLegalEntities(nil, request.OrgLegalEntityQueryReq{
		Basic: request.Basic{IncludeDeleted: true},
		OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{
			OnlyEffective: orgServiceBoolPointer(false),
		},
	}, orgServiceLegalEntityTable())
	if err != nil {
		t.Fatalf("query legal entities with generic include_deleted: %v", err)
	}
	if result.Total != 0 || len(result.Data) != 0 {
		t.Fatalf("platform soft-deleted row leaked through organization query: %+v", result)
	}
}

func TestOrgServiceGetLegalEntityDetailUsesInternalIDAndSafeDTO(t *testing.T) {
	service, db := newOrgServiceTestSubject(t)
	entity := orgServiceLegalEntity(10, "LE-010", "Safe Legal", "Safe", "enabled", nil, nil)
	entity.SourceId = "secret-source-id"
	entity.SourceVersion = "secret-version"
	entity.LocalNote = "platform note"
	testutil.MustCreate(t, db, &entity)

	detail, err := service.GetLegalEntityDetail(nil, entity.Id, request.OrgLegalEntityDetailReq{})
	if err != nil {
		t.Fatalf("get legal entity detail: %v", err)
	}
	if detail.Id != entity.Id || detail.Code != entity.Code || detail.LocalNote != entity.LocalNote {
		t.Fatalf("unexpected legal entity detail: %+v", detail)
	}
	encoded, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal legal entity detail: %v", err)
	}
	for _, forbidden := range []string{"source_id", "source_version", "sync_status", "last_error"} {
		if strings.Contains(string(encoded), `"`+forbidden+`"`) {
			t.Fatalf("detail leaked %s: %s", forbidden, encoded)
		}
	}

	_, err = service.GetLegalEntityDetail(nil, 0, request.OrgLegalEntityDetailReq{})
	assertOrgServiceAdminError(t, err, response.ErrorCategoryParameter, apperrors.ErrorCodeParamInvalid)

	_, err = service.GetLegalEntityDetail(nil, 999, request.OrgLegalEntityDetailReq{})
	assertOrgServiceAdminError(t, err, response.ErrorCategoryBusiness, apperrors.ErrorCodeOrgLegalEntityNotFound)

	disabled := orgServiceLegalEntity(11, "LE-011", "Disabled", "Disabled", "disabled", nil, nil)
	testutil.MustCreate(t, db, &disabled)
	_, err = service.GetLegalEntityDetail(nil, disabled.Id, request.OrgLegalEntityDetailReq{})
	assertOrgServiceAdminError(t, err, response.ErrorCategoryBusiness, apperrors.ErrorCodeOrgLegalEntityNotFound)

	detail, err = service.GetLegalEntityDetail(nil, disabled.Id, request.OrgLegalEntityDetailReq{
		OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{IncludeDisabled: true},
	})
	if err != nil || detail.Id != disabled.Id {
		t.Fatalf("include disabled detail = %+v, err=%v", detail, err)
	}
}

func TestOrgServiceGetLegalEntityTreeSupportsRootsSubtreesOrphansAndCycles(t *testing.T) {
	t.Run("multiple roots subtree and orphan", func(t *testing.T) {
		service, db := newOrgServiceTestSubject(t)
		rootA := orgServiceLegalEntity(1, "A", "Root A", "Root A", "enabled", nil, nil)
		childA := orgServiceLegalEntity(2, "A-1", "Child A", "Child A", "enabled", nil, nil)
		childA.ParentId = &rootA.Id
		rootB := orgServiceLegalEntity(3, "B", "Root B", "Root B", "enabled", nil, nil)
		missingParent := 999
		orphan := orgServiceLegalEntity(4, "O", "Orphan", "Orphan", "enabled", nil, nil)
		orphan.ParentId = &missingParent
		testutil.MustCreate(t, db, &[]model.OrgLegalEntity{rootA, childA, rootB, orphan})

		tree, err := service.GetLegalEntityTree(nil, request.OrgLegalEntityTreeReq{})
		if err != nil {
			t.Fatalf("get legal entity tree: %v", err)
		}
		if len(tree) != 3 {
			t.Fatalf("tree root count = %d, want 3: %+v", len(tree), tree)
		}
		byValue := make(map[int]response.OrgLegalEntityTreeNodeRes, len(tree))
		for _, node := range tree {
			byValue[node.Value] = node
			if node.Value != node.LegalEntityId {
				t.Fatalf("tree value %d differs from legal_entity_id %d", node.Value, node.LegalEntityId)
			}
		}
		if len(byValue[rootA.Id].Children) != 1 || byValue[rootA.Id].Children[0].Value != childA.Id {
			t.Fatalf("root A children are invalid: %+v", byValue[rootA.Id])
		}
		if !byValue[orphan.Id].Orphan {
			t.Fatalf("orphan node was not diagnosed: %+v", byValue[orphan.Id])
		}

		subtree, err := service.GetLegalEntityTree(nil, request.OrgLegalEntityTreeReq{RootId: &rootA.Id})
		if err != nil {
			t.Fatalf("get legal entity subtree: %v", err)
		}
		if len(subtree) != 1 || subtree[0].Value != rootA.Id || len(subtree[0].Children) != 1 {
			t.Fatalf("unexpected legal entity subtree: %+v", subtree)
		}
	})

	t.Run("cycle", func(t *testing.T) {
		service, db := newOrgServiceTestSubject(t)
		first := orgServiceLegalEntity(10, "C-1", "Cycle 1", "Cycle 1", "enabled", nil, nil)
		second := orgServiceLegalEntity(11, "C-2", "Cycle 2", "Cycle 2", "enabled", nil, nil)
		first.ParentId = &second.Id
		second.ParentId = &first.Id
		testutil.MustCreate(t, db, &[]model.OrgLegalEntity{first, second})

		_, err := service.GetLegalEntityTree(nil, request.OrgLegalEntityTreeReq{})
		assertOrgServiceAdminError(t, err, response.ErrorCategoryBusiness, apperrors.ErrorCodeOrgLegalEntityCycle)
	})
}

func TestOrgServiceLegalEntityOptionsUseInternalIDsAndReplayHistoricalSelection(t *testing.T) {
	service, db := newOrgServiceTestSubject(t)
	enabled := orgServiceLegalEntity(1, "LE-001", "Alpha Legal", "Alpha", "enabled", nil, nil)
	other := orgServiceLegalEntity(2, "LE-002", "Beta Legal", "Beta", "enabled", nil, nil)
	disabled := orgServiceLegalEntity(3, "LE-003", "Legacy Legal", "Legacy", "disabled", nil, nil)
	testutil.MustCreate(t, db, &[]model.OrgLegalEntity{enabled, other, disabled})

	result, err := service.QueryLegalEntityOptions(nil, request.OrgLegalEntityOptionsReq{
		Page:        1,
		Num:         10,
		Keyword:     "Alpha",
		SelectedIds: []int{disabled.Id},
	}, orgServiceLegalEntityTable())
	if err != nil {
		t.Fatalf("query legal entity options: %v", err)
	}
	if result.Total != 1 || len(result.Data) != 2 {
		t.Fatalf("unexpected legal entity options: %+v", result)
	}
	if result.Data[0].Value != enabled.Id || result.Data[0].Disabled {
		t.Fatalf("enabled option is invalid: %+v", result.Data[0])
	}
	if result.Data[1].Value != disabled.Id || !result.Data[1].Disabled {
		t.Fatalf("historical option replay is invalid: %+v", result.Data[1])
	}
	if result.Data[1].Value == 0 || result.Data[1].Label != "LE-003 - Legacy Legal" {
		t.Fatalf("option value/label contract is invalid: %+v", result.Data[1])
	}
}

func newOrgServiceTestSubject(t *testing.T) (*OrgService, *gorm.DB) {
	return newOrgServiceTestSubjectWithAuditWriter(t, &testTransactionalAuditWriter{})
}

func newOrgServiceTestSubjectWithAuditWriter(
	t *testing.T,
	auditWriter TransactionalAuditWriter,
) (*OrgService, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLite(
		t,
		&model.OrgLegalEntity{},
		&model.OrgUnit{},
		&model.OrgStructure{},
		&model.OrgStructureNode{},
		&model.OrgPosition{},
		&model.OrgEmployee{},
		&model.OrgAssignment{},
		&model.SysUser{},
	)
	primaryDB := &database.PrimaryDB{DB: db}
	return NewOrgService(
		impl.NewOrgLegalEntityRepositoryImpl(primaryDB),
		impl.NewOrgUnitRepositoryImpl(primaryDB),
		impl.NewOrgStructureRepositoryImpl(primaryDB),
		impl.NewOrgStructureNodeRepositoryImpl(primaryDB),
		impl.NewOrgEmployeeRepositoryImpl(primaryDB),
		impl.NewOrgPositionRepositoryImpl(primaryDB),
		impl.NewOrgAssignmentRepositoryImpl(primaryDB),
		auditWriter,
	), db
}

func orgServiceLegalEntity(
	id int,
	code string,
	name string,
	shortName string,
	status string,
	validFrom *time.Time,
	validTo *time.Time,
) model.OrgLegalEntity {
	return model.OrgLegalEntity{
		Basic:            model.Basic{Id: id, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-" + code,
		SourceCode:       "source-code-" + code,
		Code:             code,
		Name:             name,
		ShortName:        shortName,
		EntityType:       "legal_company",
		Status:           status,
		ValidFrom:        validFrom,
		ValidTo:          validTo,
		SyncStatus:       "synced",
	}
}

func orgServiceLegalEntityTable() model.SysTable {
	field := func(code string, fieldType enum.SysTableFieldType, quick bool) model.SysTableField {
		return model.SysTableField{
			FieldCode:        code,
			FieldType:        fieldType,
			IsListShow:       true,
			IsQuickSearch:    quick,
			IsAdvancedSearch: true,
			IsSort:           true,
		}
	}
	return model.SysTable{
		Basic:     model.Basic{Id: 1, State: true},
		TableCode: "org_legal_entity",
		TableFields: []model.SysTableField{
			field("id", enum.BigIntFieldType, false),
			field("code", enum.VarcharFieldType, true),
			field("name", enum.VarcharFieldType, true),
			field("short_name", enum.VarcharFieldType, true),
			field("unified_social_credit_code", enum.VarcharFieldType, true),
			field("entity_type", enum.VarcharFieldType, false),
			field("parent_id", enum.BigIntFieldType, false),
			field("status", enum.VarcharFieldType, false),
			field("valid_from", enum.DatetimeFieldType, false),
			field("valid_to", enum.DatetimeFieldType, false),
		},
	}
}

func orgServiceDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation(time.DateOnly, value, model.AppLocation())
	if err != nil {
		t.Fatalf("parse fixture date %s: %v", value, err)
	}
	return parsed
}

func orgServiceBoolPointer(value bool) *bool {
	return &value
}

func assertOrgServiceAdminError(
	t *testing.T,
	err error,
	category response.ErrorCategory,
	code int,
) {
	t.Helper()
	if err == nil {
		t.Fatal("expected AdminError")
	}
	var adminErr *response.AdminError
	if !errors.As(err, &adminErr) {
		t.Fatalf("expected AdminError, got %T: %v", err, err)
	}
	if adminErr.Category != category || adminErr.ErrorCode != code {
		t.Fatalf("unexpected AdminError: %+v", adminErr)
	}
}
