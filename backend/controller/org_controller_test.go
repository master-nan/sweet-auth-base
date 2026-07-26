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
)

type orgControllerTableProviderStub struct {
	table model.SysTable
	err   error
}

func (s orgControllerTableProviderStub) GetTableByTableCode(string) (model.SysTable, error) {
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

	db := testutil.OpenSQLite(t, &model.OrgLegalEntity{})
	repository := impl.NewOrgLegalEntityRepositoryImpl(&database.PrimaryDB{DB: db})
	orgService := service.NewOrgService(repository)
	controller := &OrgController{
		orgService: orgService,
		sysTableProvider: orgControllerTableProviderStub{
			table: orgControllerLegalEntityTable(),
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
