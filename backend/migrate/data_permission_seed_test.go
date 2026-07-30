package main

import (
	"backend/model"
	"reflect"
	"testing"

	"gorm.io/gorm"
)

func TestDataPermissionFoundationSeedIsIdempotentAndPreservesMaintainedFields(t *testing.T) {
	db := dataPermissionSeedTestDB(t)
	sf := newMigrationTestSnowflake(t)

	if err := seedDataPermissionFoundation(db, sf); err != nil {
		t.Fatalf("seed data permission foundation: %v", err)
	}
	firstCounts := dataPermissionSeedCountSnapshot(t, db)
	firstKeys := dataPermissionSeedKeySnapshot(t, db)
	customizeDataPermissionSeedFields(t, db)

	if err := seedDataPermissionFoundation(db, sf); err != nil {
		t.Fatalf("seed data permission foundation twice: %v", err)
	}
	secondCounts := dataPermissionSeedCountSnapshot(t, db)
	secondKeys := dataPermissionSeedKeySnapshot(t, db)

	if !reflect.DeepEqual(firstCounts, secondCounts) {
		t.Fatalf("data permission seed counts changed: first=%#v second=%#v", firstCounts, secondCounts)
	}
	if !reflect.DeepEqual(firstKeys, secondKeys) {
		t.Fatalf("data permission seed stable IDs changed: first=%#v second=%#v", firstKeys, secondKeys)
	}

	assertDataPermissionSeedCatalog(t, db)
	assertDataPermissionSeedMaintainedFieldsPreserved(t, db)
	assertNoDuplicateGroups(t, db, "sys_dict", []string{"dict_code"})
	assertNoDuplicateGroups(t, db, "sys_dict_item", []string{"dict_id", "item_code"})
	assertNoDuplicateGroups(t, db, "sys_data_dimension_definition", []string{"code"})
}

func TestDataPermissionFoundationSeedUsesStableBusinessKeys(t *testing.T) {
	dictCodes := make(map[string]struct{})
	itemCodes := make(map[string]struct{})
	for _, seed := range dataPermissionDictionarySeeds() {
		if seed.code == "" {
			t.Fatal("data permission dictionary has empty code")
		}
		if _, exists := dictCodes[seed.code]; exists {
			t.Fatalf("duplicate data permission dictionary code %s", seed.code)
		}
		dictCodes[seed.code] = struct{}{}
		for _, item := range seed.items {
			if item.code == "" || item.value == "" {
				t.Fatalf("dictionary %s contains unstable item: %#v", seed.code, item)
			}
			if _, exists := itemCodes[item.code]; exists {
				t.Fatalf("duplicate data permission dictionary item code %s", item.code)
			}
			itemCodes[item.code] = struct{}{}
		}
	}

	dimensionCodes := make(map[string]struct{})
	for _, seed := range dataPermissionDimensionSeeds() {
		if seed.code == "" || seed.providerCode == "" || seed.category == "" || seed.valueType == "" {
			t.Fatalf("data permission dimension seed is incomplete: %#v", seed)
		}
		if _, exists := dimensionCodes[seed.code]; exists {
			t.Fatalf("duplicate data permission dimension code %s", seed.code)
		}
		dimensionCodes[seed.code] = struct{}{}
	}
}

func dataPermissionSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := migrateTestDB(t)
	if err := db.AutoMigrate(&model.SysDict{}, &model.SysDictItem{}); err != nil {
		t.Fatalf("migrate data permission dictionaries: %v", err)
	}
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate data permission schema: %v", err)
	}
	return db
}

func dataPermissionSeedCountSnapshot(t *testing.T, db *gorm.DB) map[string]int64 {
	t.Helper()
	return map[string]int64{
		"dict":       countRows(t, db, &model.SysDict{}),
		"dict_item":  countRows(t, db, &model.SysDictItem{}),
		"dimensions": countRows(t, db, &model.DataDimensionDefinition{}),
	}
}

func dataPermissionSeedKeySnapshot(t *testing.T, db *gorm.DB) map[string]map[string]int64 {
	t.Helper()
	return map[string]map[string]int64{
		"dict": keyIDSnapshot(
			t,
			db,
			"SELECT dict_code AS seed_key, id FROM sys_dict",
		),
		"dict_item": keyIDSnapshot(
			t,
			db,
			"SELECT CAST(dict_id AS TEXT) || ':' || item_code AS seed_key, id FROM sys_dict_item",
		),
		"dimensions": keyIDSnapshot(
			t,
			db,
			"SELECT code AS seed_key, id FROM sys_data_dimension_definition",
		),
	}
}

func customizeDataPermissionSeedFields(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Model(&model.DataDimensionDefinition{}).
		Where("code = ?", "legal_entity").
		Updates(map[string]any{
			"name":        "管理员维护法人维度",
			"description": "管理员维护说明",
		}).Error; err != nil {
		t.Fatalf("customize data permission dimension: %v", err)
	}

	var dict model.SysDict
	if err := db.Where("dict_code = ?", "data_permission_operation").First(&dict).Error; err != nil {
		t.Fatalf("query data permission operation dictionary: %v", err)
	}
	if err := db.Model(&model.SysDict{}).
		Where("id = ?", dict.Id).
		Update("dict_name", "管理员维护操作字典").Error; err != nil {
		t.Fatalf("customize data permission dictionary: %v", err)
	}
	if err := db.Model(&model.SysDictItem{}).
		Where("dict_id = ? AND item_code = ?", dict.Id, "data_permission_operation_query").
		Update("item_name", "管理员维护查询名称").Error; err != nil {
		t.Fatalf("customize data permission dictionary item: %v", err)
	}
}

func assertDataPermissionSeedCatalog(t *testing.T, db *gorm.DB) {
	t.Helper()
	if got := countRows(t, db, &model.SysDict{}); got != 7 {
		t.Fatalf("expected 7 data permission dictionaries, got %d", got)
	}
	if got := countRows(t, db, &model.SysDictItem{}); got != 26 {
		t.Fatalf("expected 26 data permission dictionary items, got %d", got)
	}
	if got := countRows(t, db, &model.DataDimensionDefinition{}); got != 3 {
		t.Fatalf("expected 3 data permission dimensions, got %d", got)
	}

	expectedItems := map[string][]string{
		"data_permission_dimension_category": {
			model.DataDimensionCategoryOrganization,
			model.DataDimensionCategoryEmployee,
			model.DataDimensionCategoryUser,
			model.DataDimensionCategoryBusiness,
			model.DataDimensionCategorySystem,
		},
		"data_permission_dimension_value_type": {
			model.DataDimensionValueTypeBigint,
			model.DataDimensionValueTypeString,
		},
		"data_permission_operation": {
			model.DataPermissionOperationQuery,
			model.DataPermissionOperationDetail,
			model.DataPermissionOperationCreate,
			model.DataPermissionOperationUpdate,
			model.DataPermissionOperationDelete,
			model.DataPermissionOperationExport,
			model.DataPermissionOperationRun,
		},
		"data_permission_ownership_binding_type": {
			model.DataOwnershipBindingTypeMetadataField,
			model.DataOwnershipBindingTypeRegisteredField,
		},
		"data_permission_scope_source": {
			model.DataPolicyScopeSourceCurrentUser,
			model.DataPolicyScopeSourceCurrentEmployee,
			model.DataPolicyScopeSourceEffectiveLegalEntities,
			model.DataPolicyScopeSourceEffectiveOrgUnits,
			model.DataPolicyScopeSourceSpecifiedValues,
			model.DataPolicyScopeSourceProviderSubjectScope,
		},
		"data_permission_relation": {
			model.DataPolicyRelationExact,
			model.DataPolicyRelationSelfAndDescendants,
		},
		"data_permission_operator": {
			model.DataPolicyOperatorEqual,
			model.DataPolicyOperatorIn,
		},
	}
	for dictCode, values := range expectedItems {
		var dict model.SysDict
		if err := db.Where("dict_code = ?", dictCode).First(&dict).Error; err != nil {
			t.Fatalf("query dictionary %s: %v", dictCode, err)
		}
		var items []model.SysDictItem
		if err := db.Where("dict_id = ?", dict.Id).Order("item_code ASC").Find(&items).Error; err != nil {
			t.Fatalf("query dictionary items %s: %v", dictCode, err)
		}
		actual := make(map[string]struct{}, len(items))
		for _, item := range items {
			actual[item.ItemValue] = struct{}{}
		}
		for _, value := range values {
			if _, exists := actual[value]; !exists {
				t.Errorf("dictionary %s missing value %s", dictCode, value)
			}
		}
	}

	for _, expected := range dataPermissionDimensionSeeds() {
		var dimension model.DataDimensionDefinition
		if err := db.Where("code = ?", expected.code).First(&dimension).Error; err != nil {
			t.Fatalf("query dimension %s: %v", expected.code, err)
		}
		if dimension.Category != expected.category ||
			dimension.ValueType != expected.valueType ||
			dimension.ProviderCode != expected.providerCode ||
			dimension.SelectorType == nil ||
			*dimension.SelectorType != expected.selectorType {
			t.Errorf("unexpected dimension %s: %#v", expected.code, dimension)
		}
	}

	var forbiddenCount int64
	if err := db.Model(&model.SysDictItem{}).
		Where("item_value = ?", "report_source").
		Count(&forbiddenCount).Error; err != nil {
		t.Fatalf("count forbidden ownership binding: %v", err)
	}
	if forbiddenCount != 0 {
		t.Fatalf("report_source must not be seeded, got %d entries", forbiddenCount)
	}
	if err := db.Model(&model.DataDimensionDefinition{}).
		Where("category = ?", model.DataDimensionCategoryBusiness).
		Count(&forbiddenCount).Error; err != nil {
		t.Fatalf("count concrete business dimensions: %v", err)
	}
	if forbiddenCount != 0 {
		t.Fatalf("concrete business dimensions must not be seeded, got %d", forbiddenCount)
	}
	for _, modelValue := range []any{
		&model.DataResource{},
		&model.DataResourceOperation{},
		&model.DataOwnershipField{},
		&model.DataPolicy{},
		&model.DataPolicyRule{},
		&model.DataGrant{},
	} {
		if got := countRows(t, db, modelValue); got != 0 {
			t.Errorf("foundation seed created forbidden %T records: %d", modelValue, got)
		}
	}
}

func assertDataPermissionSeedMaintainedFieldsPreserved(t *testing.T, db *gorm.DB) {
	t.Helper()
	var dimension model.DataDimensionDefinition
	if err := db.Where("code = ?", "legal_entity").First(&dimension).Error; err != nil {
		t.Fatalf("query customized data permission dimension: %v", err)
	}
	if dimension.Name != "管理员维护法人维度" || dimension.Description != "管理员维护说明" {
		t.Fatalf("seed overwrote maintained dimension fields: %#v", dimension)
	}

	var dict model.SysDict
	if err := db.Where("dict_code = ?", "data_permission_operation").First(&dict).Error; err != nil {
		t.Fatalf("query customized data permission dictionary: %v", err)
	}
	if dict.DictName != "管理员维护操作字典" {
		t.Fatalf("seed overwrote maintained dictionary name: %q", dict.DictName)
	}
	var item model.SysDictItem
	if err := db.Where(
		"dict_id = ? AND item_code = ?",
		dict.Id,
		"data_permission_operation_query",
	).First(&item).Error; err != nil {
		t.Fatalf("query customized data permission dictionary item: %v", err)
	}
	if item.ItemName != "管理员维护查询名称" {
		t.Fatalf("seed overwrote maintained dictionary item name: %q", item.ItemName)
	}
}
