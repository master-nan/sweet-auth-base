package service

import (
	"backend/enum"
	"backend/internal/datapermission"
	apperrors "backend/internal/errors"
	"backend/model"
	"testing"

	"gorm.io/gorm"
)

func TestDataOwnershipMetadataFieldValidationAdapter(t *testing.T) {
	t.Run("deleted field is unavailable", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(
			t,
			&testTransactionalAuditWriter{},
			nil,
		)
		resource, dimension, field := createMetadataOwnershipFixtures(t, db)
		if err := db.Delete(&field).Error; err != nil {
			t.Fatalf("delete metadata field: %v", err)
		}

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			metadataOwnershipCreateRequest(resource.Id, dimension.Id, field.Id),
		)
		assertDataOwnershipConfigError(
			t,
			err,
			apperrors.ErrorCodeDataOwnershipMetadataFieldNotFound,
		)
	})

	t.Run("forbidden technical and display fields are rejected", func(t *testing.T) {
		tests := []struct {
			name       string
			fieldCode  string
			primaryKey bool
			category   enum.SysTableFieldCategory
		}{
			{name: "primary key", fieldCode: "business_id", primaryKey: true},
			{name: "source field", fieldCode: "source_id"},
			{name: "technical path", fieldCode: "path"},
			{name: "display field", fieldCode: "owner_org_name"},
			{name: "calculated field", fieldCode: "owner_org_id", category: enum.CalculatedField},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				service, db := newDataOwnershipConfigTestSubject(
					t,
					&testTransactionalAuditWriter{},
					nil,
				)
				resource, dimension, field := createMetadataOwnershipFixtures(t, db)
				updates := map[string]any{
					"field_code":     tt.fieldCode,
					"is_primary_key": tt.primaryKey,
				}
				if tt.category != "" {
					updates["field_category"] = tt.category
				}
				if err := db.Model(&model.SysTableField{}).
					Where("id = ?", field.Id).
					Updates(updates).Error; err != nil {
					t.Fatalf("change metadata field: %v", err)
				}

				_, err := service.CreateOwnership(
					dataResourceConfigContext(),
					metadataOwnershipCreateRequest(resource.Id, dimension.Id, field.Id),
				)
				assertDataOwnershipConfigError(
					t,
					err,
					apperrors.ErrorCodeDataOwnershipMetadataFieldForbidden,
				)
			})
		}
	})

	t.Run("known identifier must match dimension semantics", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(
			t,
			&testTransactionalAuditWriter{},
			nil,
		)
		resource, dimension, field := createMetadataOwnershipFixtures(t, db)
		if err := db.Model(&model.DataDimensionDefinition{}).
			Where("id = ?", dimension.Id).
			Updates(map[string]any{
				"code":     "employee",
				"category": model.DataDimensionCategoryEmployee,
			}).Error; err != nil {
			t.Fatalf("change dimension semantics: %v", err)
		}

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			metadataOwnershipCreateRequest(resource.Id, dimension.Id, field.Id),
		)
		assertDataOwnershipConfigError(
			t,
			err,
			apperrors.ErrorCodeDataOwnershipMetadataDimension,
		)
	})
}

func TestDataOwnershipRegisteredFieldValidationAdapter(t *testing.T) {
	t.Run("registered binding succeeds", func(t *testing.T) {
		registry := newOwnershipFieldRegistryForServiceTest(t, registeredOwnerOrgRegistration())
		service, db := newDataOwnershipConfigTestSubject(
			t,
			&testTransactionalAuditWriter{},
			registry,
		)
		resource := dataOwnershipResourceFixture(151, model.DataResourceTypeBusinessService)
		dimension := dataOwnershipDimensionFixture(251, model.DataDimensionValueTypeBigint)
		mustCreateOwnershipFixtures(t, db, resource, dimension)

		result, err := service.CreateOwnership(
			dataResourceConfigContext(),
			registeredOwnershipCreateRequest(resource.Id, dimension.Id),
		)
		if err != nil {
			t.Fatalf("create registered ownership: %v", err)
		}
		if result.OwnershipCode != "owner_org" ||
			result.BindingTarget.ReferenceCode == nil ||
			*result.BindingTarget.ReferenceCode != "owner_org_id" {
			t.Fatalf("unexpected registered ownership: %+v", result)
		}
	})

	t.Run("unregistered field is rejected", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(
			t,
			&testTransactionalAuditWriter{},
			newOwnershipFieldRegistryForServiceTest(t),
		)
		resource := dataOwnershipResourceFixture(152, model.DataResourceTypeBusinessService)
		dimension := dataOwnershipDimensionFixture(252, model.DataDimensionValueTypeBigint)
		mustCreateOwnershipFixtures(t, db, resource, dimension)

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			registeredOwnershipCreateRequest(resource.Id, dimension.Id),
		)
		assertDataOwnershipConfigError(
			t,
			err,
			apperrors.ErrorCodeDataOwnershipRegisteredFieldMissing,
		)
	})

	t.Run("resource mismatch is rejected", func(t *testing.T) {
		registry := newOwnershipFieldRegistryForServiceTest(t, registeredOwnerOrgRegistration())
		service, db := newDataOwnershipConfigTestSubject(
			t,
			&testTransactionalAuditWriter{},
			registry,
		)
		resource := dataOwnershipResourceFixture(153, model.DataResourceTypeBusinessService)
		resource.ResourceCode = "wms.inventory"
		dimension := dataOwnershipDimensionFixture(253, model.DataDimensionValueTypeBigint)
		mustCreateOwnershipFixtures(t, db, resource, dimension)

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			registeredOwnershipCreateRequest(resource.Id, dimension.Id),
		)
		assertDataOwnershipConfigError(
			t,
			err,
			apperrors.ErrorCodeDataOwnershipRegisteredResource,
		)
	})

	t.Run("registered value type mismatch is rejected", func(t *testing.T) {
		registration := registeredOwnerOrgRegistration()
		registration.ValueType = model.DataDimensionValueTypeString
		registry := newOwnershipFieldRegistryForServiceTest(t, registration)
		service, db := newDataOwnershipConfigTestSubject(
			t,
			&testTransactionalAuditWriter{},
			registry,
		)
		resource := dataOwnershipResourceFixture(154, model.DataResourceTypeBusinessService)
		dimension := dataOwnershipDimensionFixture(254, model.DataDimensionValueTypeBigint)
		mustCreateOwnershipFixtures(t, db, resource, dimension)

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			registeredOwnershipCreateRequest(resource.Id, dimension.Id),
		)
		assertDataOwnershipConfigError(
			t,
			err,
			apperrors.ErrorCodeDataOwnershipValueTypeMismatch,
		)
	})

	t.Run("unsupported dimension is rejected", func(t *testing.T) {
		registration := registeredOwnerOrgRegistration()
		registration.SupportedDimensions = []string{"legal_entity"}
		registry := newOwnershipFieldRegistryForServiceTest(t, registration)
		service, db := newDataOwnershipConfigTestSubject(
			t,
			&testTransactionalAuditWriter{},
			registry,
		)
		resource := dataOwnershipResourceFixture(155, model.DataResourceTypeBusinessService)
		dimension := dataOwnershipDimensionFixture(255, model.DataDimensionValueTypeBigint)
		mustCreateOwnershipFixtures(t, db, resource, dimension)

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			registeredOwnershipCreateRequest(resource.Id, dimension.Id),
		)
		assertDataOwnershipConfigError(
			t,
			err,
			apperrors.ErrorCodeDataOwnershipRegisteredDimension,
		)
	})

	t.Run("non service resource cannot use registered binding", func(t *testing.T) {
		registry := newOwnershipFieldRegistryForServiceTest(t, registeredOwnerOrgRegistration())
		service, db := newDataOwnershipConfigTestSubject(
			t,
			&testTransactionalAuditWriter{},
			registry,
		)
		resource := dataOwnershipResourceFixture(156, model.DataResourceTypeReport)
		resource.ServiceCode = nil
		reportDefinitionID := 901
		resource.ReportDefinitionId = &reportDefinitionID
		dimension := dataOwnershipDimensionFixture(256, model.DataDimensionValueTypeBigint)
		mustCreateOwnershipFixtures(t, db, resource, dimension)

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			registeredOwnershipCreateRequest(resource.Id, dimension.Id),
		)
		assertDataOwnershipConfigError(
			t,
			err,
			apperrors.ErrorCodeDataOwnershipBindingInvalid,
		)
	})
}

func newOwnershipFieldRegistryForServiceTest(
	t *testing.T,
	registrations ...datapermission.OwnershipFieldRegistration,
) *datapermission.OwnershipFieldRegistry {
	t.Helper()
	registry, err := datapermission.NewOwnershipFieldRegistry(registrations...)
	if err != nil {
		t.Fatalf("initialize ownership field registry: %v", err)
	}
	return registry
}

func registeredOwnerOrgRegistration() datapermission.OwnershipFieldRegistration {
	return datapermission.OwnershipFieldRegistration{
		ResourceCode:        "tms.transport_order",
		OwnershipCode:       "owner_org",
		AdapterFieldCode:    "owner_org_id",
		ValueType:           model.DataDimensionValueTypeBigint,
		SupportedDimensions: []string{"management_org"},
		SupportedOperations: []string{
			model.DataPermissionOperationQuery,
			model.DataPermissionOperationDetail,
		},
	}
}

func mustCreateOwnershipFixtures(
	t *testing.T,
	db *gorm.DB,
	resource model.DataResource,
	dimension model.DataDimensionDefinition,
) {
	t.Helper()
	if err := db.Create(&resource).Error; err != nil {
		t.Fatalf("create resource fixture: %v", err)
	}
	if err := db.Create(&dimension).Error; err != nil {
		t.Fatalf("create dimension fixture: %v", err)
	}
}
