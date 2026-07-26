package controller

import (
	"backend/dto/response"
	"backend/enum"
	"backend/internal/database"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/middleware"
	"backend/model"
	"backend/repository/impl"
	"backend/service"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	orgLegalEntityReaderRole = "organization_legal_entity_reader"
	orgLegalEntityDeniedRole = "organization_legal_entity_denied"
	orgManagementQueryRole   = "organization_management_query"
)

type orgControllerTableProviderStub struct {
	table  model.SysTable
	tables map[string]model.SysTable
	err    error
}

func (s orgControllerTableProviderStub) GetTableByTableCode(tableCode string) (model.SysTable, error) {
	if s.tables != nil {
		return s.tables[tableCode], s.err
	}
	return s.table, s.err
}

func TestOrgControllerLegalEntityRoutesUseSeededPermissionsAndUnifiedResponse(t *testing.T) {
	router, db, enforcer := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)
	entity := orgControllerLegalEntityFixture(1, "LE-001", "法人一")
	testutil.MustCreate(t, db, &entity)

	testutil.AssertPermissions(
		t,
		enforcer,
		testutil.PermissionCase{
			Name:    "query allowed",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/legal-entity/query",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "tree allowed",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/legal-entity/tree",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "options allowed",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/legal-entity/options",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "detail allowed",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/legal-entity/:id",
			Method:  http.MethodGet,
			Allowed: true,
		},
	)

	for _, item := range []struct {
		name   string
		method string
		target string
		body   string
	}{
		{
			name:   "query",
			method: http.MethodPost,
			target: "/admin/org/legal-entity/query",
			body:   `{"page":1,"num":10}`,
		},
		{
			name:   "tree",
			method: http.MethodPost,
			target: "/admin/org/legal-entity/tree",
			body:   `{}`,
		},
		{
			name:   "options",
			method: http.MethodPost,
			target: "/admin/org/legal-entity/options",
			body:   `{"page":1,"num":10,"keyword":"法人"}`,
		},
		{
			name:   "detail",
			method: http.MethodGet,
			target: "/admin/org/legal-entity/1",
		},
	} {
		t.Run(item.name, func(t *testing.T) {
			recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
				Method: item.method,
				Target: item.target,
				Body:   bytes.NewBufferString(item.body),
				Header: http.Header{"Content-Type": []string{"application/json"}},
			})
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
			}
			var payload response.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !payload.Success || payload.Code != http.StatusOK {
				t.Fatalf("unexpected response: %#v", payload)
			}
			for _, forbidden := range []string{
				`"source_id"`,
				`"source_version"`,
				`"sync_status"`,
				`"last_error"`,
			} {
				if strings.Contains(recorder.Body.String(), forbidden) {
					t.Fatalf("response leaked %s: %s", forbidden, recorder.Body.String())
				}
			}
		})
	}
}

func TestOrgControllerLegalEntityQueryRejectsRoleWithoutButtonPermission(t *testing.T) {
	router, _, enforcer := newOrgControllerTestRouter(t, orgLegalEntityDeniedRole)
	testutil.AssertPermissions(
		t,
		enforcer,
		testutil.PermissionCase{
			Name:    "query denied",
			Subject: orgLegalEntityDeniedRole,
			Path:    "/admin/org/legal-entity/query",
			Method:  http.MethodPost,
			Allowed: false,
		},
	)

	recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
		Method: http.MethodPost,
		Target: "/admin/org/legal-entity/query",
		Body:   bytes.NewBufferString(`{"page":1,"num":10}`),
		Header: http.Header{"Content-Type": []string{"application/json"}},
	})
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", recorder.Code, recorder.Body.String())
	}
	var payload response.AdminError
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode permission error: %v", err)
	}
	if payload.Success || payload.ErrorCode != 30006 {
		t.Fatalf("unexpected permission error: %#v", payload)
	}
}

func TestOrgControllerLegalEntityDetailUsesStableErrorResponse(t *testing.T) {
	router, _, _ := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)

	for _, item := range []struct {
		name      string
		target    string
		status    int
		errorCode int
	}{
		{
			name:      "invalid internal id",
			target:    "/admin/org/legal-entity/not-an-id",
			status:    http.StatusBadRequest,
			errorCode: apperrors.ErrorCodeParamInvalid,
		},
		{
			name:      "missing internal id",
			target:    "/admin/org/legal-entity/999",
			status:    http.StatusNotFound,
			errorCode: apperrors.ErrorCodeOrgLegalEntityNotFound,
		},
	} {
		t.Run(item.name, func(t *testing.T) {
			recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
				Method: http.MethodGet,
				Target: item.target,
			})
			if recorder.Code != item.status {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, item.status, recorder.Body.String())
			}
			var payload response.AdminError
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if payload.Success || payload.ErrorCode != item.errorCode {
				t.Fatalf("unexpected error response: %#v", payload)
			}
		})
	}
}

func newOrgControllerTestRouter(
	t *testing.T,
	roleName string,
) (*gin.Engine, *gorm.DB, *casbin.Enforcer) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	restoreLogger := zap.ReplaceGlobals(zap.NewNop())
	t.Cleanup(restoreLogger)

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
	orgService := service.NewOrgService(
		impl.NewOrgLegalEntityRepositoryImpl(primaryDB),
		impl.NewOrgUnitRepositoryImpl(primaryDB),
		impl.NewOrgStructureRepositoryImpl(primaryDB),
		impl.NewOrgStructureNodeRepositoryImpl(primaryDB),
		impl.NewOrgEmployeeRepositoryImpl(primaryDB),
		impl.NewOrgPositionRepositoryImpl(primaryDB),
	)
	controller := &OrgController{
		orgService: orgService,
		sysTableProvider: orgControllerTableProviderStub{
			tables: map[string]model.SysTable{
				orgLegalEntityTableCode: orgControllerLegalEntityTable(),
				orgStructureTableCode:   orgControllerManagementTable(orgStructureTableCode),
				orgUnitTableCode:        orgControllerManagementTable(orgUnitTableCode),
				orgEmployeeTableCode:    orgControllerManagementTable(orgEmployeeTableCode),
				orgPositionTableCode:    orgControllerManagementTable(orgPositionTableCode),
			},
		},
	}

	enforcer, err := casbin.NewEnforcer("../casbin_model.conf")
	if err != nil {
		t.Fatalf("new Casbin enforcer: %v", err)
	}
	for _, policy := range [][]string{
		{orgLegalEntityReaderRole, "/admin/org/legal-entity/query", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/legal-entity/tree", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/legal-entity/options", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/legal-entity/:id", http.MethodGet},
		{orgLegalEntityReaderRole, "/admin/org/structure/query", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/structure/options", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/structure/:id", http.MethodGet},
		{orgLegalEntityReaderRole, "/admin/org/unit/query", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/unit/options", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/unit/tree", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/unit/:id", http.MethodGet},
		{orgManagementQueryRole, "/admin/org/structure/query", http.MethodPost},
		{orgManagementQueryRole, "/admin/org/structure/options", http.MethodPost},
		{orgManagementQueryRole, "/admin/org/unit/query", http.MethodPost},
		{orgManagementQueryRole, "/admin/org/unit/options", http.MethodPost},
		{orgManagementQueryRole, "/admin/org/unit/tree", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/employee/query", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/employee/options", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/employee/:id", http.MethodGet},
		{orgLegalEntityReaderRole, "/admin/org/position/query", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/position/options", http.MethodPost},
		{orgLegalEntityReaderRole, "/admin/org/position/:id", http.MethodGet},
	} {
		if _, err = enforcer.AddPolicy(policy[0], policy[1], policy[2]); err != nil {
			t.Fatalf("add Casbin policy %v: %v", policy, err)
		}
	}

	router := gin.New()
	router.Use(middleware.ResponseHandler())
	router.Use(func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{
			UserName: "organization_reader",
			Roles:    []model.SysRole{{Name: roleName}},
		})
		ctx.Next()
	})
	router.Use(middleware.CasbinHandler(enforcer, middleware.CasbinOptions{
		EnforcePolicyCoverage: true,
	}))
	router.POST("/admin/org/legal-entity/query", controller.QueryLegalEntities)
	router.GET("/admin/org/legal-entity/:id", controller.GetLegalEntityDetail)
	router.POST("/admin/org/legal-entity/tree", controller.GetLegalEntityTree)
	router.POST("/admin/org/legal-entity/options", controller.QueryLegalEntityOptions)
	router.POST("/admin/org/structure/query", controller.QueryStructures)
	router.POST("/admin/org/structure/options", controller.QueryStructureOptions)
	router.GET("/admin/org/structure/:id", controller.GetStructureDetail)
	router.POST("/admin/org/unit/query", controller.QueryOrgUnits)
	router.POST("/admin/org/unit/options", controller.QueryOrgUnitOptions)
	router.POST("/admin/org/unit/tree", controller.GetStructureOrgTree)
	router.GET("/admin/org/unit/:id", controller.GetOrgUnitDetail)
	router.POST("/admin/org/employee/query", controller.QueryEmployees)
	router.POST("/admin/org/employee/options", controller.QueryEmployeeOptions)
	router.GET("/admin/org/employee/:id", controller.GetEmployeeDetail)
	router.POST("/admin/org/position/query", controller.QueryPositions)
	router.POST("/admin/org/position/options", controller.QueryPositionOptions)
	router.GET("/admin/org/position/:id", controller.GetPositionDetail)
	return router, db, enforcer
}

func orgControllerLegalEntityFixture(id int, code, name string) model.OrgLegalEntity {
	return model.OrgLegalEntity{
		Basic:            model.Basic{Id: id, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-" + code,
		SourceCode:       "source-code-" + code,
		Code:             code,
		Name:             name,
		ShortName:        name,
		EntityType:       "legal_company",
		Status:           "enabled",
		SourceVersion:    "secret-version",
		SyncStatus:       "synced",
	}
}

func orgControllerLegalEntityTable() model.SysTable {
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
		TableCode: orgLegalEntityTableCode,
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

func orgControllerManagementTable(tableCode string) model.SysTable {
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
	fields := []model.SysTableField{
		field("id", enum.BigIntFieldType, false),
		field("code", enum.VarcharFieldType, true),
		field("name", enum.VarcharFieldType, true),
		field("status", enum.VarcharFieldType, false),
		field("valid_from", enum.DatetimeFieldType, false),
		field("valid_to", enum.DatetimeFieldType, false),
	}
	switch tableCode {
	case orgStructureTableCode:
		fields = append(fields,
			field("structure_type", enum.VarcharFieldType, false),
			field("is_default", enum.BooleanFieldType, false),
		)
	case orgUnitTableCode:
		fields = append(fields,
			field("unit_type", enum.VarcharFieldType, false),
			field("primary_legal_entity_id", enum.BigIntFieldType, false),
		)
	case orgEmployeeTableCode:
		fields = []model.SysTableField{
			field("id", enum.BigIntFieldType, false),
			field("employee_no", enum.VarcharFieldType, true),
			field("name", enum.VarcharFieldType, true),
			field("employment_status", enum.VarcharFieldType, false),
			field("primary_legal_entity_id", enum.BigIntFieldType, false),
			field("user_id", enum.BigIntFieldType, false),
			field("valid_from", enum.DatetimeFieldType, false),
			field("valid_to", enum.DatetimeFieldType, false),
		}
	case orgPositionTableCode:
		fields = append(fields,
			field("org_unit_id", enum.BigIntFieldType, false),
			field("position_type", enum.VarcharFieldType, false),
			field("is_manager_position", enum.BooleanFieldType, false),
		)
	}
	return model.SysTable{
		Basic:       model.Basic{Id: 1, State: true},
		TableCode:   tableCode,
		TableFields: fields,
	}
}
