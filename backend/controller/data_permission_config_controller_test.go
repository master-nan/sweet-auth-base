package controller

import (
	"bytes"
	"context"
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
	createReq      request.DataResourceCreateReq
	updateReq      request.DataResourceUpdateReq
	replaceReq     request.DataResourceOperationBatchReq
	pageReq        request.DataResourceQueryReq
	pageTable      model.SysTable
	pageResult     response.ListResult[response.DataResourceListRes]
	detail         response.DataResourceDetailRes
	operations     []response.DataResourceOperationListRes
	createCalls    int
	updateCalls    int
	replaceCalls   int
	operationCalls int
	err            error
}

func (s *dataPermissionConfigResourceStub) CreateResource(
	_ context.Context,
	req request.DataResourceCreateReq,
) (response.DataResourceDetailRes, error) {
	s.createReq = req
	s.createCalls++
	return s.detail, s.err
}

func (s *dataPermissionConfigResourceStub) UpdateResource(
	_ context.Context,
	req request.DataResourceUpdateReq,
) (response.DataResourceDetailRes, error) {
	s.updateReq = req
	s.updateCalls++
	return s.detail, s.err
}

func (s *dataPermissionConfigResourceStub) GetResource(
	context.Context,
	int,
) (response.DataResourceDetailRes, error) {
	return s.detail, s.err
}

func (s *dataPermissionConfigResourceStub) PageResources(
	_ context.Context,
	req request.DataResourceQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataResourceListRes], error) {
	s.pageReq = req
	s.pageTable = table
	return s.pageResult, s.err
}

func (s *dataPermissionConfigResourceStub) ListResourceOperations(
	context.Context,
	int,
) ([]response.DataResourceOperationListRes, error) {
	s.operationCalls++
	return s.operations, s.err
}

func (s *dataPermissionConfigResourceStub) ReplaceResourceOperations(
	_ context.Context,
	req request.DataResourceOperationBatchReq,
) ([]response.DataResourceOperationListRes, error) {
	s.replaceReq = req
	s.replaceCalls++
	return s.operations, s.err
}

type dataPermissionConfigOwnershipStub struct {
	dimensionPage      response.ListResult[response.DataDimensionDefinitionListRes]
	ownershipPage      response.ListResult[response.DataOwnershipFieldListRes]
	createReq          request.DataOwnershipFieldCreateReq
	updateReq          request.DataOwnershipFieldUpdateReq
	dimensionPageTable model.SysTable
	ownershipPageTable model.SysTable
	list               []response.DataOwnershipFieldListRes
	detail             response.DataOwnershipFieldDetailRes
	err                error
}

func (s *dataPermissionConfigOwnershipStub) PageDimensions(
	_ context.Context,
	_ request.DataDimensionDefinitionQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataDimensionDefinitionListRes], error) {
	s.dimensionPageTable = table
	return s.dimensionPage, s.err
}

func (s *dataPermissionConfigOwnershipStub) CreateOwnership(
	_ context.Context,
	req request.DataOwnershipFieldCreateReq,
) (response.DataOwnershipFieldDetailRes, error) {
	s.createReq = req
	return s.detail, s.err
}

func (s *dataPermissionConfigOwnershipStub) UpdateOwnership(
	_ context.Context,
	req request.DataOwnershipFieldUpdateReq,
) (response.DataOwnershipFieldDetailRes, error) {
	s.updateReq = req
	return s.detail, s.err
}

func (s *dataPermissionConfigOwnershipStub) GetOwnership(
	context.Context,
	int,
) (response.DataOwnershipFieldDetailRes, error) {
	return s.detail, s.err
}

func (s *dataPermissionConfigOwnershipStub) ListOwnershipsByResource(
	context.Context,
	int,
) ([]response.DataOwnershipFieldListRes, error) {
	return s.list, s.err
}

func (s *dataPermissionConfigOwnershipStub) PageOwnerships(
	_ context.Context,
	_ request.DataOwnershipFieldQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataOwnershipFieldListRes], error) {
	s.ownershipPageTable = table
	return s.ownershipPage, s.err
}

type dataPermissionConfigPolicyStub struct {
	pageResult     response.ListResult[response.DataPolicyListRes]
	rulePageResult response.ListResult[response.DataPolicyRuleListRes]
	createReq      request.DataPolicyCreateReq
	updateReq      request.DataPolicyUpdateReq
	replaceReq     request.DataPolicyRuleBatchReq
	detail         response.DataPolicyDetailRes
	err            error
}

func (s *dataPermissionConfigPolicyStub) CreatePolicy(
	_ context.Context,
	req request.DataPolicyCreateReq,
) (response.DataPolicyDetailRes, error) {
	s.createReq = req
	return s.detail, s.err
}

func (s *dataPermissionConfigPolicyStub) UpdatePolicy(
	_ context.Context,
	req request.DataPolicyUpdateReq,
) (response.DataPolicyDetailRes, error) {
	s.updateReq = req
	return s.detail, s.err
}

func (s *dataPermissionConfigPolicyStub) GetPolicy(
	context.Context,
	int,
) (response.DataPolicyDetailRes, error) {
	return s.detail, s.err
}

func (s *dataPermissionConfigPolicyStub) PagePolicies(
	context.Context,
	request.DataPolicyQueryReq,
	model.SysTable,
) (response.ListResult[response.DataPolicyListRes], error) {
	return s.pageResult, s.err
}

func (s *dataPermissionConfigPolicyStub) PagePolicyRules(
	context.Context,
	request.DataPolicyRuleQueryReq,
	model.SysTable,
) (response.ListResult[response.DataPolicyRuleListRes], error) {
	return s.rulePageResult, s.err
}

func (s *dataPermissionConfigPolicyStub) ReplacePolicyRules(
	_ context.Context,
	req request.DataPolicyRuleBatchReq,
) ([]response.DataPolicyRuleListRes, error) {
	s.replaceReq = req
	return s.rulePageResult.Data, s.err
}

type dataPermissionConfigGrantStub struct {
	pageResult response.ListResult[response.DataGrantListRes]
	createReq  request.DataGrantCreateReq
	detail     response.DataGrantDetailRes
	err        error
}

func (s *dataPermissionConfigGrantStub) CreateGrant(
	_ context.Context,
	req request.DataGrantCreateReq,
) (response.DataGrantDetailRes, error) {
	s.createReq = req
	return s.detail, s.err
}

func (s *dataPermissionConfigGrantStub) GetGrant(
	context.Context,
	int,
) (response.DataGrantDetailRes, error) {
	return s.detail, s.err
}

func (s *dataPermissionConfigGrantStub) PageGrants(
	context.Context,
	request.DataGrantQueryReq,
	model.SysTable,
) (response.ListResult[response.DataGrantListRes], error) {
	return s.pageResult, s.err
}

type dataPermissionConfigPreflightStub struct {
	resource             response.DataPermissionValidationResultRes
	policy               response.DataPermissionValidationResultRes
	grant                response.DataPermissionValidationResultRes
	enableResourceCalls  int
	disableResourceCalls int
	enablePolicyCalls    int
	disablePolicyCalls   int
	enableGrantCalls     int
	disableGrantCalls    int
	lastResourceStateID  int
	lastPolicyStateID    int
	lastGrantStateID     int
	err                  error
}

func (s *dataPermissionConfigPreflightStub) PreflightResource(
	context.Context,
	int,
) (response.DataPermissionValidationResultRes, error) {
	return s.resource, s.err
}

func (s *dataPermissionConfigPreflightStub) PreflightPolicy(
	context.Context,
	int,
) (response.DataPermissionValidationResultRes, error) {
	return s.policy, s.err
}

func (s *dataPermissionConfigPreflightStub) PreflightGrant(
	context.Context,
	int,
) (response.DataPermissionValidationResultRes, error) {
	return s.grant, s.err
}

func (s *dataPermissionConfigPreflightStub) EnableResource(
	_ context.Context,
	id int,
) (response.DataPermissionValidationResultRes, error) {
	s.enableResourceCalls++
	s.lastResourceStateID = id
	return s.resource, s.err
}

func (s *dataPermissionConfigPreflightStub) DisableResource(
	_ context.Context,
	id int,
) (response.DataPermissionValidationResultRes, error) {
	s.disableResourceCalls++
	s.lastResourceStateID = id
	return s.resource, s.err
}

func (s *dataPermissionConfigPreflightStub) EnablePolicy(
	_ context.Context,
	id int,
) (response.DataPermissionValidationResultRes, error) {
	s.enablePolicyCalls++
	s.lastPolicyStateID = id
	return s.policy, s.err
}

func (s *dataPermissionConfigPreflightStub) DisablePolicy(
	_ context.Context,
	id int,
) (response.DataPermissionValidationResultRes, error) {
	s.disablePolicyCalls++
	s.lastPolicyStateID = id
	return s.policy, s.err
}

func (s *dataPermissionConfigPreflightStub) EnableGrant(
	_ context.Context,
	id int,
) (response.DataPermissionValidationResultRes, error) {
	s.enableGrantCalls++
	s.lastGrantStateID = id
	return s.grant, s.err
}

func (s *dataPermissionConfigPreflightStub) DisableGrant(
	_ context.Context,
	id int,
) (response.DataPermissionValidationResultRes, error) {
	s.disableGrantCalls++
	s.lastGrantStateID = id
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
		operations: []response.DataResourceOperationListRes{{
			DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
				Id:    102,
				State: true,
			},
			ResourceId:        101,
			Operation:         model.DataPermissionOperationQuery,
			PermissionEnabled: false,
		}},
	}
	ownershipStub := &dataPermissionConfigOwnershipStub{
		dimensionPage: response.ListResult[response.DataDimensionDefinitionListRes]{
			Data: []response.DataDimensionDefinitionListRes{{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{
					Id:    301,
					State: true,
				},
				DimensionCode: "management_organization",
				Name:          "管理组织",
				Category:      model.DataDimensionCategoryOrganization,
				ValueType:     model.DataDimensionValueTypeBigint,
				ProviderCode:  "organization_provider",
			}},
			Total: 1,
		},
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
	ownershipStub.ownershipPage = response.ListResult[response.DataOwnershipFieldListRes]{
		Data:  append([]response.DataOwnershipFieldListRes(nil), ownershipStub.list...),
		Total: len(ownershipStub.list),
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
		{"dimension query", http.MethodPost, "/admin/data-permission/config/dimension/query", `{"page":1,"num":20}`},
		{"resource query", http.MethodPost, "/admin/data-permission/config/resource/query", `{"page":1,"num":20}`},
		{"resource detail", http.MethodGet, "/admin/data-permission/config/resource/101", ""},
		{"resource operation list", http.MethodGet, "/admin/data-permission/config/resource/101/operations", ""},
		{"ownership list", http.MethodGet, "/admin/data-permission/config/resource/101/ownerships", ""},
		{"ownership query", http.MethodPost, "/admin/data-permission/config/ownership/query", `{"page":1,"num":20}`},
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
	if ownershipStub.dimensionPageTable.TableCode != dataDimensionConfigTableCode {
		t.Fatalf("dimension table code = %q", ownershipStub.dimensionPageTable.TableCode)
	}
	if ownershipStub.ownershipPageTable.TableCode != dataOwnershipConfigTableCode {
		t.Fatalf("ownership table code = %q", ownershipStub.ownershipPageTable.TableCode)
	}
}

func TestDataPermissionConfigControllerPublishesConfigurationWrites(t *testing.T) {
	resourceStub := &dataPermissionConfigResourceStub{
		detail: response.DataResourceDetailRes{
			DataResourceListRes: response.DataResourceListRes{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{Id: 101, State: true},
				ResourceCode:                "transport_order",
				Name:                        "运输订单",
				ResourceType:                model.DataResourceTypeBusinessService,
			},
		},
		operations: []response.DataResourceOperationListRes{{
			DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{Id: 102, State: true},
			ResourceId:                  101,
			Operation:                   model.DataPermissionOperationQuery,
		}},
	}
	ownershipStub := &dataPermissionConfigOwnershipStub{
		detail: response.DataOwnershipFieldDetailRes{
			DataOwnershipFieldListRes: response.DataOwnershipFieldListRes{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{Id: 201, State: true},
				ResourceId:                  101,
				OwnershipCode:               "owner_org",
				DimensionId:                 301,
				BindingType:                 model.DataOwnershipBindingTypeRegisteredField,
				ValueType:                   model.DataDimensionValueTypeBigint,
			},
		},
	}
	policyStub := &dataPermissionConfigPolicyStub{
		detail: response.DataPolicyDetailRes{
			DataPolicyListRes: response.DataPolicyListRes{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{Id: 401, State: true},
				PolicyCode:                  "own_org",
				Name:                        "本组织",
				PolicyType:                  model.DataPolicyTypeRuleSet,
			},
		},
		rulePageResult: response.ListResult[response.DataPolicyRuleListRes]{
			Data: []response.DataPolicyRuleListRes{{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{Id: 402, State: true},
				PolicyId:                    401,
				Sequence:                    1,
				DimensionId:                 301,
				OwnershipCode:               "owner_org",
				ScopeSource:                 model.DataPolicyScopeSourceEffectiveOrgUnits,
				Relation:                    model.DataPolicyRelationExact,
				Operator:                    model.DataPolicyOperatorIn,
			}},
		},
	}
	grantStub := &dataPermissionConfigGrantStub{
		detail: response.DataGrantDetailRes{
			DataGrantListRes: response.DataGrantListRes{
				DataPermissionConfigBaseRes: response.DataPermissionConfigBaseRes{Id: 501, State: true},
				SubjectType:                 model.DataGrantSubjectTypeRole,
				SubjectId:                   1,
				ResourceId:                  101,
				Operation:                   model.DataPermissionOperationQuery,
				PolicyId:                    401,
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

	requests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{
			"create resource",
			http.MethodPost,
			"/admin/data-permission/config/resource",
			`{"resource_code":"transport_order","name":"运输订单","resource_type":"business_service","target":{"reference_code":"tms.transport_order"},"adapter_code":"tms"}`,
		},
		{
			"update resource",
			http.MethodPut,
			"/admin/data-permission/config/resource/101",
			`{"id":999,"name":"运输订单配置"}`,
		},
		{
			"replace operations",
			http.MethodPut,
			"/admin/data-permission/config/resource/101/operations",
			`{"resource_id":999,"items":[{"operation":"query"}]}`,
		},
		{
			"enable resource permission",
			http.MethodPut,
			"/admin/data-permission/config/resource/101/permission",
			`{"permission_enabled":true}`,
		},
		{
			"disable resource permission",
			http.MethodPut,
			"/admin/data-permission/config/resource/101/permission",
			`{"permission_enabled":false}`,
		},
		{
			"create ownership",
			http.MethodPost,
			"/admin/data-permission/config/ownership",
			`{"resource_id":101,"ownership_code":"owner_org","dimension_id":301,"binding_type":"registered_field","binding_target":{"reference_code":"tms.transport_order.owner_org"},"value_type":"bigint"}`,
		},
		{
			"update ownership",
			http.MethodPut,
			"/admin/data-permission/config/ownership/201",
			`{"id":999,"state":false}`,
		},
		{
			"create policy",
			http.MethodPost,
			"/admin/data-permission/config/policy",
			`{"policy_code":"own_org","name":"本组织","state":false}`,
		},
		{
			"update policy",
			http.MethodPut,
			"/admin/data-permission/config/policy/401",
			`{"id":999,"name":"本组织范围"}`,
		},
		{
			"replace policy rules",
			http.MethodPut,
			"/admin/data-permission/config/policy/401/rules",
			`{"policy_id":999,"items":[{"sequence":1,"dimension_id":301,"ownership_code":"owner_org","scope_source":"effective_org_units","relation":"exact","operator":"in"}]}`,
		},
		{
			"enable policy",
			http.MethodPut,
			"/admin/data-permission/config/policy/401/state",
			`{"state":true}`,
		},
		{
			"disable policy",
			http.MethodPut,
			"/admin/data-permission/config/policy/401/state",
			`{"state":false}`,
		},
		{
			"create grant",
			http.MethodPost,
			"/admin/data-permission/config/grant",
			`{"subject_type":"role","subject_id":1,"resource_id":101,"operation":"query","policy_id":401}`,
		},
		{
			"enable grant",
			http.MethodPut,
			"/admin/data-permission/config/grant/501/state",
			`{"state":true}`,
		},
		{
			"disable grant",
			http.MethodPut,
			"/admin/data-permission/config/grant/501/state",
			`{"state":false}`,
		},
	}
	for _, item := range requests {
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
		})
	}

	if resourceStub.createCalls != 1 || resourceStub.updateCalls != 1 ||
		resourceStub.replaceCalls != 1 {
		t.Fatalf(
			"resource write calls create=%d update=%d replace=%d",
			resourceStub.createCalls,
			resourceStub.updateCalls,
			resourceStub.replaceCalls,
		)
	}
	if resourceStub.updateReq.Id != 101 || resourceStub.replaceReq.ResourceId != 101 {
		t.Fatalf(
			"path IDs did not win: update=%d replace=%d",
			resourceStub.updateReq.Id,
			resourceStub.replaceReq.ResourceId,
		)
	}
	if ownershipStub.updateReq.Id != 201 || policyStub.updateReq.Id != 401 ||
		policyStub.replaceReq.PolicyId != 401 {
		t.Fatalf(
			"configuration path IDs did not win: ownership=%d policy=%d rules=%d",
			ownershipStub.updateReq.Id,
			policyStub.updateReq.Id,
			policyStub.replaceReq.PolicyId,
		)
	}
	if preflightStub.enableResourceCalls != 1 || preflightStub.disableResourceCalls != 1 ||
		preflightStub.enablePolicyCalls != 1 || preflightStub.disablePolicyCalls != 1 ||
		preflightStub.enableGrantCalls != 1 || preflightStub.disableGrantCalls != 1 {
		t.Fatalf("state changes bypassed preflight service: %#v", preflightStub)
	}
	if preflightStub.lastResourceStateID != 101 ||
		preflightStub.lastPolicyStateID != 401 ||
		preflightStub.lastGrantStateID != 501 {
		t.Fatalf("unexpected state IDs: %#v", preflightStub)
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
		{http.MethodPost, "/admin/data-permission/config/dimension/query"},
		{http.MethodPost, "/admin/data-permission/config/resource"},
		{http.MethodPost, "/admin/data-permission/config/resource/query"},
		{http.MethodGet, "/admin/data-permission/config/resource/:id"},
		{http.MethodPut, "/admin/data-permission/config/resource/:id"},
		{http.MethodGet, "/admin/data-permission/config/resource/:id/operations"},
		{http.MethodPut, "/admin/data-permission/config/resource/:id/operations"},
		{http.MethodPut, "/admin/data-permission/config/resource/:id/permission"},
		{http.MethodGet, "/admin/data-permission/config/resource/:id/ownerships"},
		{http.MethodPost, "/admin/data-permission/config/ownership"},
		{http.MethodPost, "/admin/data-permission/config/ownership/query"},
		{http.MethodGet, "/admin/data-permission/config/ownership/:id"},
		{http.MethodPut, "/admin/data-permission/config/ownership/:id"},
		{http.MethodPost, "/admin/data-permission/config/policy"},
		{http.MethodPost, "/admin/data-permission/config/policy/query"},
		{http.MethodGet, "/admin/data-permission/config/policy/:id"},
		{http.MethodPut, "/admin/data-permission/config/policy/:id"},
		{http.MethodPut, "/admin/data-permission/config/policy/:id/rules"},
		{http.MethodPut, "/admin/data-permission/config/policy/:id/state"},
		{http.MethodPost, "/admin/data-permission/config/policy/rule/query"},
		{http.MethodPost, "/admin/data-permission/config/grant"},
		{http.MethodPost, "/admin/data-permission/config/grant/query"},
		{http.MethodGet, "/admin/data-permission/config/grant/:id"},
		{http.MethodPut, "/admin/data-permission/config/grant/:id/state"},
		{http.MethodGet, "/admin/data-permission/config/preflight/resource/:id"},
		{http.MethodGet, "/admin/data-permission/config/preflight/policy/:id"},
		{http.MethodGet, "/admin/data-permission/config/preflight/grant/:id"},
	}
}

func registerDataPermissionConfigControllerTestRoutes(
	router *gin.Engine,
	controller *DataPermissionConfigController,
) {
	router.POST("/admin/data-permission/config/dimension/query", controller.QueryDimensions)
	router.POST("/admin/data-permission/config/resource", controller.CreateResource)
	router.POST("/admin/data-permission/config/resource/query", controller.QueryResources)
	router.GET("/admin/data-permission/config/resource/:id", controller.GetResource)
	router.PUT("/admin/data-permission/config/resource/:id", controller.UpdateResource)
	router.GET("/admin/data-permission/config/resource/:id/operations", controller.ListResourceOperations)
	router.PUT("/admin/data-permission/config/resource/:id/operations", controller.ReplaceResourceOperations)
	router.PUT("/admin/data-permission/config/resource/:id/permission", controller.SetResourcePermission)
	router.GET("/admin/data-permission/config/resource/:id/ownerships", controller.ListOwnershipsByResource)
	router.POST("/admin/data-permission/config/ownership", controller.CreateOwnership)
	router.POST("/admin/data-permission/config/ownership/query", controller.QueryOwnerships)
	router.GET("/admin/data-permission/config/ownership/:id", controller.GetOwnership)
	router.PUT("/admin/data-permission/config/ownership/:id", controller.UpdateOwnership)
	router.POST("/admin/data-permission/config/policy", controller.CreatePolicy)
	router.POST("/admin/data-permission/config/policy/query", controller.QueryPolicies)
	router.GET("/admin/data-permission/config/policy/:id", controller.GetPolicy)
	router.PUT("/admin/data-permission/config/policy/:id", controller.UpdatePolicy)
	router.PUT("/admin/data-permission/config/policy/:id/rules", controller.ReplacePolicyRules)
	router.PUT("/admin/data-permission/config/policy/:id/state", controller.SetPolicyState)
	router.POST("/admin/data-permission/config/policy/rule/query", controller.QueryPolicyRules)
	router.POST("/admin/data-permission/config/grant", controller.CreateGrant)
	router.POST("/admin/data-permission/config/grant/query", controller.QueryGrants)
	router.GET("/admin/data-permission/config/grant/:id", controller.GetGrant)
	router.PUT("/admin/data-permission/config/grant/:id/state", controller.SetGrantState)
	router.GET("/admin/data-permission/config/preflight/resource/:id", controller.PreflightResource)
	router.GET("/admin/data-permission/config/preflight/policy/:id", controller.PreflightPolicy)
	router.GET("/admin/data-permission/config/preflight/grant/:id", controller.PreflightGrant)
}
