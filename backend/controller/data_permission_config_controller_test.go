package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"backend/dto/request"
	"backend/dto/response"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/middleware"
	"backend/model"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	dataPermissionConfigReaderRole = "data_permission_config_reader"
	dataPermissionConfigDeniedRole = "data_permission_config_denied"
)

type dataPermissionConfigResourceStub struct {
	pageReq    request.DataResourceQueryReq
	pageTable  model.SysTable
	pageResult response.ListResult[response.DataResourceListRes]
	detail     response.DataResourceDetailRes
	err        error
}

func (s *dataPermissionConfigResourceStub) GetResource(
	*gin.Context,
	int,
) (response.DataResourceDetailRes, error) {
	return s.detail, s.err
}

func (s *dataPermissionConfigResourceStub) PageResources(
	_ *gin.Context,
	req request.DataResourceQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataResourceListRes], error) {
	s.pageReq = req
	s.pageTable = table
	return s.pageResult, s.err
}

type dataPermissionConfigOwnershipStub struct {
	list   []response.DataOwnershipFieldListRes
	detail response.DataOwnershipFieldDetailRes
	err    error
}

func (s *dataPermissionConfigOwnershipStub) GetOwnership(
	*gin.Context,
	int,
) (response.DataOwnershipFieldDetailRes, error) {
	return s.detail, s.err
}

func (s *dataPermissionConfigOwnershipStub) ListOwnershipsByResource(
	*gin.Context,
	int,
) ([]response.DataOwnershipFieldListRes, error) {
	return s.list, s.err
}

type dataPermissionConfigPolicyStub struct {
	pageResult     response.ListResult[response.DataPolicyListRes]
	rulePageResult response.ListResult[response.DataPolicyRuleListRes]
	detail         response.DataPolicyDetailRes
	err            error
}

func (s *dataPermissionConfigPolicyStub) GetPolicy(
	*gin.Context,
	int,
) (response.DataPolicyDetailRes, error) {
	return s.detail, s.err
}

func (s *dataPermissionConfigPolicyStub) PagePolicies(
	*gin.Context,
	request.DataPolicyQueryReq,
	model.SysTable,
) (response.ListResult[response.DataPolicyListRes], error) {
	return s.pageResult, s.err
}

func (s *dataPermissionConfigPolicyStub) PagePolicyRules(
	*gin.Context,
	request.DataPolicyRuleQueryReq,
	model.SysTable,
) (response.ListResult[response.DataPolicyRuleListRes], error) {
	return s.rulePageResult, s.err
}

type dataPermissionConfigGrantStub struct {
	pageResult response.ListResult[response.DataGrantListRes]
	detail     response.DataGrantDetailRes
	err        error
}

func (s *dataPermissionConfigGrantStub) GetGrant(
	*gin.Context,
	int,
) (response.DataGrantDetailRes, error) {
	return s.detail, s.err
}

func (s *dataPermissionConfigGrantStub) PageGrants(
	*gin.Context,
	request.DataGrantQueryReq,
	model.SysTable,
) (response.ListResult[response.DataGrantListRes], error) {
	return s.pageResult, s.err
}

type dataPermissionConfigPreflightStub struct {
	resource response.DataPermissionValidationResultRes
	policy   response.DataPermissionValidationResultRes
	grant    response.DataPermissionValidationResultRes
	err      error
}

func (s *dataPermissionConfigPreflightStub) PreflightResource(
	*gin.Context,
	int,
) (response.DataPermissionValidationResultRes, error) {
	return s.resource, s.err
}

func (s *dataPermissionConfigPreflightStub) PreflightPolicy(
	*gin.Context,
	int,
) (response.DataPermissionValidationResultRes, error) {
	return s.policy, s.err
}

func (s *dataPermissionConfigPreflightStub) PreflightGrant(
	*gin.Context,
	int,
) (response.DataPermissionValidationResultRes, error) {
	return s.grant, s.err
}

func TestDataPermissionConfigControllerQueriesUseDTOsAndUnifiedResponse(t *testing.T) {
	resourceStub := &dataPermissionConfigResourceStub{
		pageResult: response.ListResult[response.DataResourceListRes]{
			Data: []response.DataResourceListRes{{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
					Id:    101,
					State: true,
				},
				ResourceCode:      "transport_order",
				Name:              "运输订单",
				ResourceType:      model.DataResourceTypeBusinessService,
				PermissionEnabled: false,
			}},
			Total: 1,
		},
		detail: response.DataResourceDetailRes{
			DataResourceListRes: response.DataResourceListRes{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
					Id:    101,
					State: true,
				},
				ResourceCode: "transport_order",
				Name:         "运输订单",
				ResourceType: model.DataResourceTypeBusinessService,
			},
			AdapterCode: "tms.transport_order",
		},
	}
	ownershipStub := &dataPermissionConfigOwnershipStub{
		list: []response.DataOwnershipFieldListRes{{
			DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
				Id:    201,
				State: true,
			},
			ResourceId:    101,
			OwnershipCode: "owner_org",
			DimensionId:   301,
			BindingType:   model.DataOwnershipBindingTypeRegisteredField,
			ValueType:     model.DataDimensionValueTypeBigint,
		}},
		detail: response.DataOwnershipFieldDetailRes{
			DataOwnershipFieldListRes: response.DataOwnershipFieldListRes{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
					Id:    201,
					State: true,
				},
				ResourceId:    101,
				OwnershipCode: "owner_org",
				DimensionId:   301,
				BindingType:   model.DataOwnershipBindingTypeRegisteredField,
				ValueType:     model.DataDimensionValueTypeBigint,
			},
		},
	}
	policyStub := &dataPermissionConfigPolicyStub{
		pageResult: response.ListResult[response.DataPolicyListRes]{
			Data: []response.DataPolicyListRes{{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
					Id:    401,
					State: true,
				},
				PolicyCode: "own_org",
				Name:       "本组织",
				PolicyType: model.DataPolicyTypeRuleSet,
			}},
			Total: 1,
		},
		detail: response.DataPolicyDetailRes{
			DataPolicyListRes: response.DataPolicyListRes{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
					Id:    401,
					State: true,
				},
				PolicyCode: "own_org",
				Name:       "本组织",
				PolicyType: model.DataPolicyTypeRuleSet,
			},
			Rules: []response.DataPolicyRuleDetailRes{},
		},
		rulePageResult: response.ListResult[response.DataPolicyRuleListRes]{
			Data: []response.DataPolicyRuleListRes{{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
					Id:    402,
					State: true,
				},
				PolicyId:      401,
				Sequence:      1,
				DimensionId:   301,
				OwnershipCode: "owner_org",
				ScopeSource:   model.DataPolicyScopeSourceEffectiveOrgUnits,
				Relation:      model.DataPolicyRelationExact,
				Operator:      model.DataPolicyOperatorIn,
			}},
			Total: 1,
		},
	}
	grantStub := &dataPermissionConfigGrantStub{
		pageResult: response.ListResult[response.DataGrantListRes]{
			Data: []response.DataGrantListRes{{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
					Id:    501,
					State: true,
				},
				SubjectType: model.DataGrantSubjectTypeRole,
				SubjectId:   1,
				ResourceId:  101,
				Operation:   model.DataPermissionOperationQuery,
				PolicyId:    401,
			}},
			Total: 1,
		},
		detail: response.DataGrantDetailRes{
			DataGrantListRes: response.DataGrantListRes{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
					Id:    501,
					State: true,
				},
				SubjectType: model.DataGrantSubjectTypeRole,
				SubjectId:   1,
				ResourceId:  101,
				Operation:   model.DataPermissionOperationQuery,
				PolicyId:    401,
			},
		},
	}
	preflightStub := &dataPermissionConfigPreflightStub{
		resource: response.DataPermissionValidationResultRes{Valid: true, Errors: []response.DataPermissionValidationErrorRes{}},
		policy:   response.DataPermissionValidationResultRes{Valid: true, Errors: []response.DataPermissionValidationErrorRes{}},
		grant:    response.DataPermissionValidationResultRes{Valid: true, Errors: []response.DataPermissionValidationErrorRes{}},
	}
	controller := newDataPermissionConfigController(
		resourceStub,
		ownershipStub,
		policyStub,
		grantStub,
		preflightStub,
		nil,
	)
	router, _ := newDataPermissionConfigControllerTestRouter(
		t,
		controller,
		dataPermissionConfigReaderRole,
	)

	for _, item := range []struct {
		name   string
		method string
		target string
		body   string
	}{
		{"resource query", http.MethodPost, "/admin/data-permission/config/resource/query", `{"page":1,"num":20}`},
		{"resource detail", http.MethodGet, "/admin/data-permission/config/resource/101", ""},
		{"ownership list", http.MethodGet, "/admin/data-permission/config/resource/101/ownerships", ""},
		{"ownership detail", http.MethodGet, "/admin/data-permission/config/ownership/201", ""},
		{"policy query", http.MethodPost, "/admin/data-permission/config/policy/query", `{"page":1,"num":20}`},
		{"policy detail", http.MethodGet, "/admin/data-permission/config/policy/401", ""},
		{"policy rule query", http.MethodPost, "/admin/data-permission/config/policy/rule/query", `{"page":1,"num":20,"policy_id":401}`},
		{"grant query", http.MethodPost, "/admin/data-permission/config/grant/query", `{"page":1,"num":20}`},
		{"grant detail", http.MethodGet, "/admin/data-permission/config/grant/501", ""},
		{"resource preflight", http.MethodGet, "/admin/data-permission/config/preflight/resource/101", ""},
		{"policy preflight", http.MethodGet, "/admin/data-permission/config/preflight/policy/401", ""},
		{"grant preflight", http.MethodGet, "/admin/data-permission/config/preflight/grant/501", ""},
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
				`"gmt_delete"`,
				`"delete_user"`,
				`"description"`,
				`"table_id"`,
				`"service_code"`,
			} {
				if strings.Contains(recorder.Body.String(), forbidden) {
					t.Fatalf("response leaked internal field %s: %s", forbidden, recorder.Body.String())
				}
			}
		})
	}

	if resourceStub.pageReq.Page != 1 || resourceStub.pageReq.Num != 20 {
		t.Fatalf("resource query request = %#v", resourceStub.pageReq)
	}
	if resourceStub.pageTable.TableCode != dataResourceConfigTableCode {
		t.Fatalf("resource table code = %q", resourceStub.pageTable.TableCode)
	}
}

func TestDataPermissionConfigControllerValidationPermissionAndPreflightErrors(t *testing.T) {
	preflight := response.DataPermissionValidationResultRes{
		Valid: false,
		Errors: []response.DataPermissionValidationErrorRes{{
			Code:       "ownership_missing",
			Message:    "数据资源缺少有效归属定义",
			ObjectType: request.DataPermissionConfigObjectResource,
			ObjectId:   101,
		}},
	}
	controller := newDataPermissionConfigController(
		&dataPermissionConfigResourceStub{},
		&dataPermissionConfigOwnershipStub{},
		&dataPermissionConfigPolicyStub{},
		&dataPermissionConfigGrantStub{},
		&dataPermissionConfigPreflightStub{resource: preflight},
		nil,
	)

	allowedRouter, enforcer := newDataPermissionConfigControllerTestRouter(
		t,
		controller,
		dataPermissionConfigReaderRole,
	)
	testutil.AssertPermissions(t, enforcer,
		testutil.PermissionCase{
			Name:    "resource query",
			Subject: dataPermissionConfigReaderRole,
			Path:    "/admin/data-permission/config/resource/query",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "resource preflight",
			Subject: dataPermissionConfigReaderRole,
			Path:    "/admin/data-permission/config/preflight/resource/:id",
			Method:  http.MethodGet,
			Allowed: true,
		},
	)

	invalidID := testutil.PerformRequest(t, allowedRouter, testutil.HTTPRequest{
		Method: http.MethodGet,
		Target: "/admin/data-permission/config/resource/not-a-number",
	})
	if invalidID.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status = %d: %s", invalidID.Code, invalidID.Body.String())
	}
	var parameterError response.AdminError
	if err := json.Unmarshal(invalidID.Body.Bytes(), &parameterError); err != nil {
		t.Fatalf("decode parameter error: %v", err)
	}
	if parameterError.ErrorCode != apperrors.ErrorCodeParamInvalid {
		t.Fatalf("parameter error = %#v", parameterError)
	}

	invalidQuery := testutil.PerformRequest(t, allowedRouter, testutil.HTTPRequest{
		Method: http.MethodPost,
		Target: "/admin/data-permission/config/resource/query",
		Body:   bytes.NewBufferString(`{"page":-1,"num":20}`),
		Header: http.Header{"Content-Type": []string{"application/json"}},
	})
	if invalidQuery.Code != http.StatusBadRequest {
		t.Fatalf("invalid query status = %d: %s", invalidQuery.Code, invalidQuery.Body.String())
	}

	preflightRecorder := testutil.PerformRequest(t, allowedRouter, testutil.HTTPRequest{
		Method: http.MethodGet,
		Target: "/admin/data-permission/config/preflight/resource/101",
	})
	if preflightRecorder.Code != http.StatusOK {
		t.Fatalf("preflight status = %d: %s", preflightRecorder.Code, preflightRecorder.Body.String())
	}
	var preflightPayload struct {
		Success bool                                       `json:"success"`
		Data    response.DataPermissionValidationResultRes `json:"data"`
	}
	if err := json.Unmarshal(preflightRecorder.Body.Bytes(), &preflightPayload); err != nil {
		t.Fatalf("decode preflight response: %v", err)
	}
	if !preflightPayload.Success || preflightPayload.Data.Valid ||
		len(preflightPayload.Data.Errors) != 1 ||
		preflightPayload.Data.Errors[0].Code != "ownership_missing" {
		t.Fatalf("preflight response = %#v", preflightPayload)
	}

	deniedRouter, _ := newDataPermissionConfigControllerTestRouter(
		t,
		controller,
		dataPermissionConfigDeniedRole,
	)
	denied := testutil.PerformRequest(t, deniedRouter, testutil.HTTPRequest{
		Method: http.MethodGet,
		Target: "/admin/data-permission/config/resource/101",
	})
	if denied.Code != http.StatusForbidden {
		t.Fatalf("denied status = %d: %s", denied.Code, denied.Body.String())
	}
	var permissionError response.AdminError
	if err := json.Unmarshal(denied.Body.Bytes(), &permissionError); err != nil {
		t.Fatalf("decode permission error: %v", err)
	}
	if permissionError.ErrorCode != 30006 {
		t.Fatalf("permission error = %#v", permissionError)
	}
}

func newDataPermissionConfigControllerTestRouter(
	t *testing.T,
	controller *DataPermissionConfigController,
	roleName string,
) (*gin.Engine, *casbin.Enforcer) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	restoreLogger := zap.ReplaceGlobals(zap.NewNop())
	t.Cleanup(restoreLogger)

	enforcer, err := casbin.NewEnforcer("../casbin_model.conf")
	if err != nil {
		t.Fatalf("new Casbin enforcer: %v", err)
	}
	for _, route := range dataPermissionConfigControllerRoutes() {
		if _, err = enforcer.AddPolicy(
			dataPermissionConfigReaderRole,
			route.path,
			route.method,
		); err != nil {
			t.Fatalf("add Casbin policy %s %s: %v", route.method, route.path, err)
		}
	}

	router := gin.New()
	router.Use(middleware.ResponseHandler())
	router.Use(func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{
			UserName: "data_permission_reader",
			Roles:    []model.SysRole{{Name: roleName}},
		})
		ctx.Next()
	})
	router.Use(middleware.CasbinHandler(enforcer, middleware.CasbinOptions{
		EnforcePolicyCoverage: true,
	}))
	registerDataPermissionConfigControllerTestRoutes(router, controller)
	return router, enforcer
}

type dataPermissionConfigControllerRoute struct {
	method string
	path   string
}

func dataPermissionConfigControllerRoutes() []dataPermissionConfigControllerRoute {
	return []dataPermissionConfigControllerRoute{
		{http.MethodPost, "/admin/data-permission/config/resource/query"},
		{http.MethodGet, "/admin/data-permission/config/resource/:id"},
		{http.MethodGet, "/admin/data-permission/config/resource/:id/ownerships"},
		{http.MethodGet, "/admin/data-permission/config/ownership/:id"},
		{http.MethodPost, "/admin/data-permission/config/policy/query"},
		{http.MethodGet, "/admin/data-permission/config/policy/:id"},
		{http.MethodPost, "/admin/data-permission/config/policy/rule/query"},
		{http.MethodPost, "/admin/data-permission/config/grant/query"},
		{http.MethodGet, "/admin/data-permission/config/grant/:id"},
		{http.MethodGet, "/admin/data-permission/config/preflight/resource/:id"},
		{http.MethodGet, "/admin/data-permission/config/preflight/policy/:id"},
		{http.MethodGet, "/admin/data-permission/config/preflight/grant/:id"},
	}
}

func registerDataPermissionConfigControllerTestRoutes(
	router *gin.Engine,
	controller *DataPermissionConfigController,
) {
	router.POST("/admin/data-permission/config/resource/query", controller.QueryResources)
	router.GET("/admin/data-permission/config/resource/:id", controller.GetResource)
	router.GET("/admin/data-permission/config/resource/:id/ownerships", controller.ListOwnershipsByResource)
	router.GET("/admin/data-permission/config/ownership/:id", controller.GetOwnership)
	router.POST("/admin/data-permission/config/policy/query", controller.QueryPolicies)
	router.GET("/admin/data-permission/config/policy/:id", controller.GetPolicy)
	router.POST("/admin/data-permission/config/policy/rule/query", controller.QueryPolicyRules)
	router.POST("/admin/data-permission/config/grant/query", controller.QueryGrants)
	router.GET("/admin/data-permission/config/grant/:id", controller.GetGrant)
	router.GET("/admin/data-permission/config/preflight/resource/:id", controller.PreflightResource)
	router.GET("/admin/data-permission/config/preflight/policy/:id", controller.PreflightPolicy)
	router.GET("/admin/data-permission/config/preflight/grant/:id", controller.PreflightGrant)
}
