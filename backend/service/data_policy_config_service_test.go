package service

import (
	"backend/dto/request"
	"backend/internal/database"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"backend/enum"
	"gorm.io/gorm"
)

func TestDataPolicyConfigServiceCreateQueryAndUpdate(t *testing.T) {
	auditWriter := &testTransactionalAuditWriter{}
	service, db := newDataPolicyConfigTestSubject(t, auditWriter)
	fixtures := createDataPolicyConfigFixtures(t, db)

	result, err := service.CreatePolicy(
		dataResourceConfigContext(),
		request.DataPolicyCreateReq{
			PolicyCode:  "management_org_scope",
			Name:        "管理组织范围",
			Description: "配置端内部说明",
			Rules: []request.DataPolicyRuleCreateItemReq{
				managementOrgRuleRequest(fixtures.managementOrgDimension.Id),
			},
		},
	)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if result.PolicyCode != "management_org_scope" ||
		result.PolicyType != model.DataPolicyTypeRuleSet ||
		len(result.Rules) != 1 {
		t.Fatalf("unexpected policy response: %+v", result)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal policy response: %v", err)
	}
	for _, forbidden := range []string{
		"description",
		"resource_id",
		"menu_id",
		"table_name",
		"配置端内部说明",
	} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("policy response leaked %q: %s", forbidden, payload)
		}
	}

	var stored model.DataPolicy
	if err = db.Where("code = ?", "management_org_scope").First(&stored).Error; err != nil {
		t.Fatalf("reload policy: %v", err)
	}
	if stored.Description != "配置端内部说明" {
		t.Fatalf("stored description = %q", stored.Description)
	}

	detail, err := service.GetPolicy(dataResourceConfigContext(), result.Id)
	if err != nil || len(detail.Rules) != 1 {
		t.Fatalf("get policy = %+v, err=%v", detail, err)
	}
	page, err := service.PagePolicies(
		dataResourceConfigContext(),
		request.DataPolicyQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{
				Page:       1,
				Num:        10,
				QuickQuery: &request.QuickQuery{Keyword: "management"},
			},
			PolicyType: model.DataPolicyTypeRuleSet,
		},
		dataPolicyConfigTable(),
	)
	if err != nil || page.Total != 1 || len(page.Data) != 1 {
		t.Fatalf("page policies = %+v, err=%v", page, err)
	}

	newName := "管理组织数据范围"
	newDescription := "更新后的内部说明"
	updated, err := service.UpdatePolicy(
		dataResourceConfigContext(),
		request.DataPolicyUpdateReq{
			Id:          result.Id,
			Name:        &newName,
			Description: &newDescription,
		},
	)
	if err != nil || updated.Name != newName {
		t.Fatalf("update policy = %+v, err=%v", updated, err)
	}
	otherCode := "changed_code"
	_, err = service.UpdatePolicy(
		dataResourceConfigContext(),
		request.DataPolicyUpdateReq{Id: result.Id, PolicyCode: &otherCode},
	)
	assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataPolicyFieldImmutable)

	disabled := false
	updated, err = service.UpdatePolicy(
		dataResourceConfigContext(),
		request.DataPolicyUpdateReq{Id: result.Id, State: &disabled},
	)
	if err != nil || updated.State {
		t.Fatalf("disable policy = %+v, err=%v", updated, err)
	}
	if len(auditWriter.snapshot()) != 3 {
		t.Fatalf("audit count = %d, want 3", len(auditWriter.snapshot()))
	}
}

func TestDataPolicyConfigServicePolicyValidation(t *testing.T) {
	service, db := newDataPolicyConfigTestSubject(t, &testTransactionalAuditWriter{})

	_, err := service.CreatePolicy(
		dataResourceConfigContext(),
		request.DataPolicyCreateReq{PolicyCode: "Invalid Code", Name: "无效策略"},
	)
	assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataPolicyCodeInvalid)

	_, err = service.CreatePolicy(
		dataResourceConfigContext(),
		request.DataPolicyCreateReq{PolicyCode: "empty_name", Name: "  "},
	)
	assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataPolicyNameRequired)

	existing := dataPolicyFixture(401, "duplicate_policy")
	testutil.MustCreate(t, db, &existing)
	_, err = service.CreatePolicy(
		dataResourceConfigContext(),
		request.DataPolicyCreateReq{PolicyCode: existing.Code, Name: "重复策略"},
	)
	assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataPolicyCodeDuplicate)
}

func TestDataPolicyConfigServiceRuleValidation(t *testing.T) {
	tests := []struct {
		name      string
		edit      func(*request.DataPolicyRuleCreateItemReq, dataPolicyConfigFixtures)
		errorCode int
	}{
		{
			name: "dimension not found",
			edit: func(req *request.DataPolicyRuleCreateItemReq, _ dataPolicyConfigFixtures) {
				req.DimensionId = 999
			},
			errorCode: apperrors.ErrorCodeDataDimensionNotFound,
		},
		{
			name: "ownership not found",
			edit: func(req *request.DataPolicyRuleCreateItemReq, _ dataPolicyConfigFixtures) {
				req.OwnershipCode = "missing_org"
			},
			errorCode: apperrors.ErrorCodeDataPolicyRuleOwnershipNotFound,
		},
		{
			name: "ownership dimension mismatch",
			edit: func(req *request.DataPolicyRuleCreateItemReq, fixtures dataPolicyConfigFixtures) {
				req.DimensionId = fixtures.legalEntityDimension.Id
			},
			errorCode: apperrors.ErrorCodeDataPolicyRuleDimensionMismatch,
		},
		{
			name: "scope source invalid",
			edit: func(req *request.DataPolicyRuleCreateItemReq, _ dataPolicyConfigFixtures) {
				req.ScopeSource = model.DataPolicyScopeSourceCurrentEmployee
				req.Operator = model.DataPolicyOperatorEqual
			},
			errorCode: apperrors.ErrorCodeDataPolicyRuleScopeSourceInvalid,
		},
		{
			name: "relation invalid",
			edit: func(req *request.DataPolicyRuleCreateItemReq, fixtures dataPolicyConfigFixtures) {
				req.DimensionId = fixtures.legalEntityDimension.Id
				req.OwnershipCode = "legal_entity"
				req.ScopeSource = model.DataPolicyScopeSourceEffectiveLegalEntities
				req.Relation = model.DataPolicyRelationSelfAndDescendants
			},
			errorCode: apperrors.ErrorCodeDataPolicyRuleRelationInvalid,
		},
		{
			name: "operator invalid for provider collection",
			edit: func(req *request.DataPolicyRuleCreateItemReq, _ dataPolicyConfigFixtures) {
				req.Operator = model.DataPolicyOperatorEqual
			},
			errorCode: apperrors.ErrorCodeDataPolicyRuleOperatorInvalid,
		},
		{
			name: "specified values type mismatch",
			edit: func(req *request.DataPolicyRuleCreateItemReq, _ dataPolicyConfigFixtures) {
				req.ScopeSource = model.DataPolicyScopeSourceSpecifiedValues
				req.SpecifiedValues = json.RawMessage(`["not-an-id"]`)
			},
			errorCode: apperrors.ErrorCodeDataPolicyRuleSpecifiedValuesInvalid,
		},
		{
			name: "specified values cannot contain SQL object",
			edit: func(req *request.DataPolicyRuleCreateItemReq, _ dataPolicyConfigFixtures) {
				req.ScopeSource = model.DataPolicyScopeSourceSpecifiedValues
				req.SpecifiedValues = json.RawMessage(`{"sql":"select *"}`)
			},
			errorCode: apperrors.ErrorCodeDataPolicyRuleSpecifiedValuesInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, db := newDataPolicyConfigTestSubject(t, &testTransactionalAuditWriter{})
			fixtures := createDataPolicyConfigFixtures(t, db)
			policy := dataPolicyFixture(501, "rule_validation")
			testutil.MustCreate(t, db, &policy)
			item := managementOrgRuleRequest(fixtures.managementOrgDimension.Id)
			test.edit(&item, fixtures)

			_, err := service.AddPolicyRule(
				dataResourceConfigContext(),
				request.DataPolicyRuleCreateReq{
					PolicyId:                    policy.Id,
					DataPolicyRuleCreateItemReq: item,
				},
			)
			assertDataResourceConfigError(t, err, test.errorCode)
			assertPolicyRuleCount(t, db, policy.Id, 0)
		})
	}
}

func TestDataPolicyConfigServiceSupportedScopeSources(t *testing.T) {
	service, db := newDataPolicyConfigTestSubject(t, &testTransactionalAuditWriter{})
	fixtures := createDataPolicyConfigFixtures(t, db)
	policy := dataPolicyFixture(550, "supported_sources")
	testutil.MustCreate(t, db, &policy)
	structureCode := "administrative"
	rules := []request.DataPolicyRuleCreateItemReq{
		{
			Sequence:      1,
			DimensionId:   fixtures.legalEntityDimension.Id,
			OwnershipCode: "legal_entity",
			ScopeSource:   model.DataPolicyScopeSourceEffectiveLegalEntities,
			Relation:      model.DataPolicyRelationExact,
			Operator:      model.DataPolicyOperatorIn,
		},
		{
			Sequence:      2,
			DimensionId:   fixtures.managementOrgDimension.Id,
			OwnershipCode: "owner_org",
			ScopeSource:   model.DataPolicyScopeSourceEffectiveOrgUnits,
			Relation:      model.DataPolicyRelationSelfAndDescendants,
			Operator:      model.DataPolicyOperatorIn,
			StructureCode: &structureCode,
		},
		{
			Sequence:      3,
			DimensionId:   fixtures.employeeDimension.Id,
			OwnershipCode: "owner_employee",
			ScopeSource:   model.DataPolicyScopeSourceCurrentEmployee,
			Relation:      model.DataPolicyRelationExact,
			Operator:      model.DataPolicyOperatorEqual,
		},
		{
			Sequence:        4,
			DimensionId:     fixtures.managementOrgDimension.Id,
			OwnershipCode:   "owner_org",
			ScopeSource:     model.DataPolicyScopeSourceSpecifiedValues,
			Relation:        model.DataPolicyRelationExact,
			Operator:        model.DataPolicyOperatorIn,
			SpecifiedValues: json.RawMessage(`[101,202]`),
		},
	}

	for _, item := range rules {
		_, err := service.AddPolicyRule(
			dataResourceConfigContext(),
			request.DataPolicyRuleCreateReq{
				PolicyId:                    policy.Id,
				DataPolicyRuleCreateItemReq: item,
			},
		)
		if err != nil {
			t.Fatalf("add supported rule %d: %v", item.Sequence, err)
		}
	}
	assertPolicyRuleCount(t, db, policy.Id, int64(len(rules)))
}

func TestDataPolicyConfigServiceSpecifiedValuesNormalizationAndLimits(t *testing.T) {
	service, db := newDataPolicyConfigTestSubject(t, &testTransactionalAuditWriter{})
	fixtures := createDataPolicyConfigFixtures(t, db)
	policy := dataPolicyFixture(560, "specified_values")
	testutil.MustCreate(t, db, &policy)

	item := managementOrgRuleRequest(fixtures.managementOrgDimension.Id)
	item.ScopeSource = model.DataPolicyScopeSourceSpecifiedValues
	item.SpecifiedValues = json.RawMessage(`[202,101,202]`)
	created, err := service.AddPolicyRule(
		dataResourceConfigContext(),
		request.DataPolicyRuleCreateReq{
			PolicyId:                    policy.Id,
			DataPolicyRuleCreateItemReq: item,
		},
	)
	if err != nil {
		t.Fatalf("add normalized specified values rule: %v", err)
	}
	if string(created.SpecifiedValues) != `[101,202]` {
		t.Fatalf("normalized specified values = %s, want [101,202]", created.SpecifiedValues)
	}

	empty := item
	empty.Sequence = 2
	empty.SpecifiedValues = json.RawMessage(`[]`)
	created, err = service.AddPolicyRule(
		dataResourceConfigContext(),
		request.DataPolicyRuleCreateReq{
			PolicyId:                    policy.Id,
			DataPolicyRuleCreateItemReq: empty,
		},
	)
	if err != nil {
		t.Fatalf("add empty specified values rule: %v", err)
	}
	if string(created.SpecifiedValues) != `[]` {
		t.Fatalf("empty specified values = %s, want []", created.SpecifiedValues)
	}

	detail, err := service.GetPolicy(dataResourceConfigContext(), policy.Id)
	if err != nil {
		t.Fatalf("get policy detail: %v", err)
	}
	if len(detail.Rules) != 2 ||
		string(detail.Rules[0].SpecifiedValues) != `[101,202]` ||
		string(detail.Rules[1].SpecifiedValues) != `[]` {
		t.Fatalf("policy detail rules lost configuration: %+v", detail.Rules)
	}

	values := make([]int, maxDataPolicySpecifiedValues+1)
	for index := range values {
		values[index] = index + 1
	}
	raw, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal oversized values: %v", err)
	}
	oversized := item
	oversized.Sequence = 3
	oversized.SpecifiedValues = raw
	_, err = service.AddPolicyRule(
		dataResourceConfigContext(),
		request.DataPolicyRuleCreateReq{
			PolicyId:                    policy.Id,
			DataPolicyRuleCreateItemReq: oversized,
		},
	)
	assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataPolicyRuleSpecifiedValuesInvalid)
	assertPolicyRuleCount(t, db, policy.Id, 2)
}

func TestDataPolicyConfigServiceRuleLifecycle(t *testing.T) {
	service, db := newDataPolicyConfigTestSubject(t, &testTransactionalAuditWriter{})
	fixtures := createDataPolicyConfigFixtures(t, db)
	policy := dataPolicyFixture(601, "rule_lifecycle")
	testutil.MustCreate(t, db, &policy)

	created, err := service.AddPolicyRule(
		dataResourceConfigContext(),
		request.DataPolicyRuleCreateReq{
			PolicyId:                    policy.Id,
			DataPolicyRuleCreateItemReq: managementOrgRuleRequest(fixtures.managementOrgDimension.Id),
		},
	)
	if err != nil {
		t.Fatalf("add rule: %v", err)
	}
	if created.Dimension == nil ||
		created.Dimension.Code != fixtures.managementOrgDimension.Code ||
		created.Policy == nil ||
		created.Policy.Id != policy.Id {
		t.Fatalf("missing rule summaries: %+v", created)
	}
	page, err := service.PagePolicyRules(
		dataResourceConfigContext(),
		request.DataPolicyRuleQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{
				Page: 1,
				Num:  10,
			},
			PolicyId:      &policy.Id,
			OwnershipCode: "owner_org",
		},
		dataPolicyRuleConfigTable(),
	)
	if err != nil || page.Total != 1 || len(page.Data) != 1 {
		t.Fatalf("page policy rules = %+v, err=%v", page, err)
	}

	_, err = service.AddPolicyRule(
		dataResourceConfigContext(),
		request.DataPolicyRuleCreateReq{
			PolicyId:                    policy.Id,
			DataPolicyRuleCreateItemReq: managementOrgRuleRequest(fixtures.managementOrgDimension.Id),
		},
	)
	assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataPolicyRuleDuplicate)

	if err = service.DisablePolicyRule(dataResourceConfigContext(), created.Id); err != nil {
		t.Fatalf("disable rule: %v", err)
	}
	var stored model.DataPolicyRule
	if err = db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("reload rule: %v", err)
	}
	if stored.State {
		t.Fatal("rule state = true, want false")
	}
}

func TestDataPolicyConfigServiceBatchAndTransactionRollback(t *testing.T) {
	t.Run("replace rules succeeds atomically", func(t *testing.T) {
		service, db := newDataPolicyConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataPolicyConfigFixtures(t, db)
		policy := dataPolicyFixture(700, "replace_success")
		existing := dataPolicyRuleFixture(800, policy.Id, fixtures.managementOrgDimension.Id)
		testutil.MustCreate(t, db, &policy)
		testutil.MustCreate(t, db, &existing)

		first := managementOrgRuleRequest(fixtures.managementOrgDimension.Id)
		first.Sequence = 10
		second := first
		second.Sequence = 20
		second.ScopeSource = model.DataPolicyScopeSourceSpecifiedValues
		second.SpecifiedValues = json.RawMessage(`[101,202]`)
		replaced, err := service.ReplacePolicyRules(
			dataResourceConfigContext(),
			request.DataPolicyRuleBatchReq{
				PolicyId: policy.Id,
				Items:    []request.DataPolicyRuleCreateItemReq{second, first},
			},
		)
		if err != nil {
			t.Fatalf("replace rules: %v", err)
		}
		if len(replaced) != 2 || replaced[0].Sequence != 10 || replaced[1].Sequence != 20 {
			t.Fatalf("replacement order = %+v", replaced)
		}
		assertPolicyRuleCount(t, db, policy.Id, 2)
		var deletedCount int64
		if err = db.Unscoped().
			Model(&model.DataPolicyRule{}).
			Where("id = ? AND gmt_delete IS NOT NULL", existing.Id).
			Count(&deletedCount).Error; err != nil {
			t.Fatalf("count replaced rules: %v", err)
		}
		if deletedCount != 1 {
			t.Fatalf("replaced rule soft-delete count = %d, want 1", deletedCount)
		}
	})

	t.Run("invalid replacement keeps existing rules", func(t *testing.T) {
		service, db := newDataPolicyConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataPolicyConfigFixtures(t, db)
		policy := dataPolicyFixture(701, "replace_rollback")
		existing := dataPolicyRuleFixture(801, policy.Id, fixtures.managementOrgDimension.Id)
		testutil.MustCreate(t, db, &policy)
		testutil.MustCreate(t, db, &existing)

		valid := managementOrgRuleRequest(fixtures.managementOrgDimension.Id)
		invalid := valid
		invalid.Sequence = 2
		invalid.ScopeSource = model.DataPolicyScopeSourceCurrentEmployee
		invalid.Operator = model.DataPolicyOperatorEqual
		_, err := service.ReplacePolicyRules(
			dataResourceConfigContext(),
			request.DataPolicyRuleBatchReq{
				PolicyId: policy.Id,
				Items:    []request.DataPolicyRuleCreateItemReq{valid, invalid},
			},
		)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataPolicyRuleScopeSourceInvalid)
		assertPolicyRuleCount(t, db, policy.Id, 1)
		var unchanged model.DataPolicyRule
		if err = db.First(&unchanged, existing.Id).Error; err != nil {
			t.Fatalf("existing rule was removed: %v", err)
		}
	})

	t.Run("create policy and rules rolls back together", func(t *testing.T) {
		service, db := newDataPolicyConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataPolicyConfigFixtures(t, db)
		valid := managementOrgRuleRequest(fixtures.managementOrgDimension.Id)
		invalid := valid
		invalid.Sequence = 2
		invalid.SpecifiedValues = json.RawMessage(`[1]`)

		_, err := service.CreatePolicy(
			dataResourceConfigContext(),
			request.DataPolicyCreateReq{
				PolicyCode: "atomic_policy",
				Name:       "事务策略",
				Rules:      []request.DataPolicyRuleCreateItemReq{valid, invalid},
			},
		)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataPolicyRuleSpecifiedValuesInvalid)
		var count int64
		if err = db.Model(&model.DataPolicy{}).Where("code = ?", "atomic_policy").Count(&count).Error; err != nil {
			t.Fatalf("count policies: %v", err)
		}
		if count != 0 {
			t.Fatalf("policy count = %d, want 0 after rollback", count)
		}
	})

	t.Run("rule count limit rejects policy before persistence", func(t *testing.T) {
		service, db := newDataPolicyConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataPolicyConfigFixtures(t, db)
		rules := make([]request.DataPolicyRuleCreateItemReq, maxDataPolicyRules+1)
		for index := range rules {
			rules[index] = managementOrgRuleRequest(fixtures.managementOrgDimension.Id)
			rules[index].Sequence = index + 1
		}

		_, err := service.CreatePolicy(
			dataResourceConfigContext(),
			request.DataPolicyCreateReq{
				PolicyCode: "too_many_rules",
				Name:       "规则过多",
				Rules:      rules,
			},
		)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataPolicyRuleCountInvalid)
		var count int64
		if err = db.Model(&model.DataPolicy{}).
			Where("code = ?", "too_many_rules").
			Count(&count).Error; err != nil {
			t.Fatalf("count rejected policy: %v", err)
		}
		if count != 0 {
			t.Fatalf("rejected policy count = %d, want 0", count)
		}
	})

	t.Run("audit failure rolls back policy and rules", func(t *testing.T) {
		service, db := newDataPolicyConfigTestSubject(
			t,
			&testTransactionalAuditWriter{err: errors.New("audit failed")},
		)
		fixtures := createDataPolicyConfigFixtures(t, db)
		_, err := service.CreatePolicy(
			dataResourceConfigContext(),
			request.DataPolicyCreateReq{
				PolicyCode: "audit_rollback",
				Name:       "审计回滚",
				Rules: []request.DataPolicyRuleCreateItemReq{
					managementOrgRuleRequest(fixtures.managementOrgDimension.Id),
				},
			},
		)
		if err == nil {
			t.Fatal("expected audit failure")
		}
		var policyCount int64
		if err = db.Model(&model.DataPolicy{}).Where("code = ?", "audit_rollback").Count(&policyCount).Error; err != nil {
			t.Fatalf("count policies: %v", err)
		}
		if policyCount != 0 {
			t.Fatalf("policy count = %d, want 0 after audit rollback", policyCount)
		}
	})

	t.Run("audit failure rolls back state change", func(t *testing.T) {
		service, db := newDataPolicyConfigTestSubject(
			t,
			&testTransactionalAuditWriter{err: errors.New("audit failed")},
		)
		policy := dataPolicyFixture(702, "state_rollback")
		testutil.MustCreate(t, db, &policy)
		disabled := false
		_, err := service.UpdatePolicy(
			dataResourceConfigContext(),
			request.DataPolicyUpdateReq{Id: policy.Id, State: &disabled},
		)
		if err == nil {
			t.Fatal("expected audit failure")
		}
		var stored model.DataPolicy
		if err = db.First(&stored, policy.Id).Error; err != nil {
			t.Fatalf("reload policy: %v", err)
		}
		if !stored.State {
			t.Fatal("policy state changed despite transaction rollback")
		}
	})
}

type dataPolicyConfigFixtures struct {
	legalEntityDimension   model.DataDimensionDefinition
	managementOrgDimension model.DataDimensionDefinition
	employeeDimension      model.DataDimensionDefinition
}

func newDataPolicyConfigTestSubject(
	t *testing.T,
	auditWriter TransactionalAuditWriter,
) (*DataPolicyConfigService, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLite(
		t,
		&model.DataDimensionDefinition{},
		&model.DataResource{},
		&model.DataOwnershipField{},
		&model.DataPolicy{},
		&model.DataPolicyRule{},
	)
	primaryDB := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	return NewDataPolicyConfigService(
		impl.NewDataPolicyRepositoryImpl(primaryDB),
		impl.NewDataPolicyRuleRepositoryImpl(primaryDB),
		impl.NewDataDimensionDefinitionRepositoryImpl(primaryDB),
		impl.NewDataOwnershipFieldRepositoryImpl(primaryDB),
		sf,
		auditWriter,
	), db
}

func createDataPolicyConfigFixtures(t *testing.T, db *gorm.DB) dataPolicyConfigFixtures {
	t.Helper()
	resource := model.DataResource{
		Basic:             model.Basic{Id: 101, State: true},
		ResourceCode:      "service:tms.transport_order",
		Name:              "运输订单",
		ResourceType:      model.DataResourceTypeBusinessService,
		ServiceCode:       dataPolicyStringPointer("tms.transport_order"),
		AdapterCode:       "registered_filter",
		PermissionEnabled: false,
	}
	legalEntity := dataPolicyDimensionFixture(201, "legal_entity")
	managementOrg := dataPolicyDimensionFixture(202, "management_org")
	employee := dataPolicyDimensionFixture(203, "employee")
	testutil.MustCreate(t, db, &resource)
	testutil.MustCreate(t, db, &legalEntity)
	testutil.MustCreate(t, db, &managementOrg)
	testutil.MustCreate(t, db, &employee)
	for _, ownership := range []model.DataOwnershipField{
		dataPolicyOwnershipFixture(301, resource.Id, legalEntity.Id, "legal_entity"),
		dataPolicyOwnershipFixture(302, resource.Id, managementOrg.Id, "owner_org"),
		dataPolicyOwnershipFixture(303, resource.Id, employee.Id, "owner_employee"),
	} {
		testutil.MustCreate(t, db, &ownership)
	}
	return dataPolicyConfigFixtures{
		legalEntityDimension:   legalEntity,
		managementOrgDimension: managementOrg,
		employeeDimension:      employee,
	}
}

func dataPolicyDimensionFixture(id int, code string) model.DataDimensionDefinition {
	category := model.DataDimensionCategoryOrganization
	selector := "org_unit"
	if code == "employee" {
		category = model.DataDimensionCategoryEmployee
		selector = "employee"
	} else if code == "legal_entity" {
		selector = "legal_entity"
	}
	return model.DataDimensionDefinition{
		Basic:        model.Basic{Id: id, State: true},
		Code:         code,
		Name:         code,
		Category:     category,
		ValueType:    model.DataDimensionValueTypeBigint,
		ProviderCode: "organization",
		SelectorType: &selector,
	}
}

func dataPolicyOwnershipFixture(
	id int,
	resourceId int,
	dimensionId int,
	code string,
) model.DataOwnershipField {
	adapterFieldCode := code + "_id"
	return model.DataOwnershipField{
		Basic:            model.Basic{Id: id, State: true},
		ResourceId:       resourceId,
		OwnershipCode:    code,
		DimensionId:      dimensionId,
		BindingType:      model.DataOwnershipBindingTypeRegisteredField,
		AdapterFieldCode: &adapterFieldCode,
		ValueType:        model.DataDimensionValueTypeBigint,
	}
}

func dataPolicyFixture(id int, code string) model.DataPolicy {
	return model.DataPolicy{
		Basic:      model.Basic{Id: id, State: true},
		Code:       code,
		Name:       code,
		PolicyType: model.DataPolicyTypeRuleSet,
	}
}

func dataPolicyRuleFixture(
	id int,
	policyId int,
	dimensionId int,
) model.DataPolicyRule {
	return model.DataPolicyRule{
		Basic:         model.Basic{Id: id, State: true},
		PolicyId:      policyId,
		Sequence:      1,
		DimensionId:   dimensionId,
		OwnershipCode: "owner_org",
		ScopeSource:   model.DataPolicyScopeSourceEffectiveOrgUnits,
		Relation:      model.DataPolicyRelationExact,
		Operator:      model.DataPolicyOperatorIn,
	}
}

func managementOrgRuleRequest(dimensionId int) request.DataPolicyRuleCreateItemReq {
	return request.DataPolicyRuleCreateItemReq{
		Sequence:      1,
		DimensionId:   dimensionId,
		OwnershipCode: "owner_org",
		ScopeSource:   model.DataPolicyScopeSourceEffectiveOrgUnits,
		Relation:      model.DataPolicyRelationExact,
		Operator:      model.DataPolicyOperatorIn,
	}
}

func dataPolicyConfigTable() model.SysTable {
	return model.SysTable{
		TableCode: "sys_data_policy",
		TableFields: []model.SysTableField{
			{
				FieldCode:        "code",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsQuickSearch:    true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
			{
				FieldCode:        "name",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsQuickSearch:    true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
			{
				FieldCode:        "policy_type",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
		},
	}
}

func dataPolicyRuleConfigTable() model.SysTable {
	return model.SysTable{
		TableCode: "sys_data_policy_rule",
		TableFields: []model.SysTableField{
			{
				FieldCode:        "policy_id",
				FieldType:        enum.BigIntFieldType,
				IsListShow:       true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
			{
				FieldCode:        "ownership_code",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsQuickSearch:    true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
			{
				FieldCode:        "sequence",
				FieldType:        enum.IntFieldType,
				IsListShow:       true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
		},
	}
}

func assertPolicyRuleCount(t *testing.T, db *gorm.DB, policyId int, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.DataPolicyRule{}).
		Where("policy_id = ?", policyId).
		Count(&count).Error; err != nil {
		t.Fatalf("count rules: %v", err)
	}
	if count != expected {
		t.Fatalf("rule count = %d, want %d", count, expected)
	}
}

func dataPolicyStringPointer(value string) *string {
	return &value
}
