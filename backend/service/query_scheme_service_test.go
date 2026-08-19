package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/audit"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	"backend/internal/metadata"
	"backend/internal/queryscheme"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gorm.io/gorm"
)

type querySchemeMetadataStub struct{ table metadata.TableMetadata }

func (stub querySchemeMetadataStub) GetTable(context.Context, string) (metadata.TableMetadata, error) {
	return stub.table, nil
}
func (stub querySchemeMetadataStub) GetTableByID(context.Context, int) (metadata.TableMetadata, error) {
	return stub.table, nil
}
func (stub querySchemeMetadataStub) GetField(context.Context, int) (metadata.FieldMetadata, error) {
	return stub.table.Fields[0], nil
}
func (stub querySchemeMetadataStub) GetFields(context.Context, int) ([]metadata.FieldMetadata, error) {
	return stub.table.Fields, nil
}
func (stub querySchemeMetadataStub) ListTables(context.Context) ([]metadata.TableMetadata, error) {
	return []metadata.TableMetadata{stub.table}, nil
}

func TestQuerySchemePersonalOwnershipRevisionAndScopeIDOR(t *testing.T) {
	service, db := newQuerySchemeTestService(t, false)
	ownerCtx := querySchemeContext(101)
	created, err := service.CreatePersonal(ownerCtx, request.QuerySchemePersonalCreateReq{
		Name: "我的用户", ScopeCode: "system.user.list", Payload: querySchemePayloadJSON(t), IsDefault: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var stored model.QueryScheme
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.OwnerUserID == nil || *stored.OwnerUserID != 101 || stored.SchemeType != model.QuerySchemeTypePersonal {
		t.Fatalf("ownership was not derived from subject: %+v", stored)
	}

	if _, err := service.Detail(querySchemeContext(202), created.ID); !errors.Is(err, myerrors.ErrQuerySchemeNotFound) {
		t.Fatalf("read another personal scheme: %v", err)
	}
	if _, err := service.Resolve(ownerCtx, created.ID, request.QuerySchemeResolveReq{ScopeCode: "system.role.list"}); !errors.Is(err, myerrors.ErrQuerySchemeNotFound) {
		t.Fatalf("cross-scope resolve: %v", err)
	}
	_, err = service.UpdatePersonal(ownerCtx, created.ID, request.QuerySchemePersonalUpdateReq{
		Name: "过期更新", Payload: querySchemePayloadJSON(t), Revision: created.Revision + 1,
	})
	if !errors.Is(err, myerrors.ErrQuerySchemeRevisionConflict) {
		t.Fatalf("revision conflict: %v", err)
	}
	cleared, err := service.SetPersonalDefault(ownerCtx, created.ID, request.QuerySchemeDefaultReq{
		IsDefault: false,
		Revision:  created.Revision,
	})
	if err != nil || cleared.IsDefault {
		t.Fatalf("clear personal default: result=%+v err=%v", cleared, err)
	}
}

func TestQuerySchemeSharedCapabilityRoleVisibilityAndCopy(t *testing.T) {
	service, db := newQuerySchemeTestService(t, true)
	managerCtx := querySchemeContext(101)
	ordinaryCtx := querySchemeContext(202)
	for _, schemeType := range []model.QuerySchemeType{model.QuerySchemeTypePublic, model.QuerySchemeTypeRole, model.QuerySchemeTypePageDefault} {
		if _, err := service.CreateShared(ordinaryCtx, request.QuerySchemeSharedCreateReq{
			Name: "越权共享", ScopeCode: "system.user.list", SchemeType: schemeType,
			Payload: querySchemePayloadJSON(t), Enabled: true,
		}); !errors.Is(err, myerrors.ErrQuerySchemeSharedForbidden) {
			t.Fatalf("ordinary %s create: %v", schemeType, err)
		}
	}
	if _, err := service.CreateShared(managerCtx, request.QuerySchemeSharedCreateReq{
		Name: "空角色", ScopeCode: "system.user.list", SchemeType: model.QuerySchemeTypeRole,
		Payload: querySchemePayloadJSON(t), Enabled: true,
	}); !errors.Is(err, myerrors.ErrQuerySchemeRoleInvalid) {
		t.Fatalf("empty role range: %v", err)
	}
	roleScheme, err := service.CreateShared(managerCtx, request.QuerySchemeSharedCreateReq{
		Name: "角色用户", ScopeCode: "system.user.list", SchemeType: model.QuerySchemeTypeRole,
		Payload: querySchemePayloadJSON(t), Enabled: true, RoleIDs: []int{11},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CopyToPersonal(managerCtx, roleScheme.ID, request.QuerySchemeCopyReq{
		ScopeCode: "system.role.list", Name: "跨范围副本",
	}); !errors.Is(err, myerrors.ErrQuerySchemeNotFound) {
		t.Fatalf("cross-scope copy: %v", err)
	}
	available, err := service.Available(ordinaryCtx, "system.user.list")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range available {
		if item.ID == roleScheme.ID {
			t.Fatal("role scheme leaked to a user without the target role")
		}
	}
	copyResult, err := service.CopyToPersonal(managerCtx, roleScheme.ID, request.QuerySchemeCopyReq{
		ScopeCode: "system.user.list", Name: "独立副本",
	})
	if err != nil {
		t.Fatal(err)
	}
	var copied model.QueryScheme
	if err := db.First(&copied, copyResult.ID).Error; err != nil {
		t.Fatal(err)
	}
	if copied.SchemeType != model.QuerySchemeTypePersonal || copied.OwnerUserID == nil || *copied.OwnerUserID != 101 {
		t.Fatalf("invalid copy: %+v", copied)
	}
}

func TestQuerySchemeResolveCurrentIdentityAndDegradedMetadata(t *testing.T) {
	service, db := newQuerySchemeTestService(t, false)
	ctx := querySchemeContext(101)
	payload := queryscheme.QuerySchemePayloadV1{
		Expressions: []request.ExpressionGroup{{Logic: enum.And, Rules: []request.QueryRule{{
			Field: "owner_id", ExpressionType: enum.Eq, Value: 999999, Type: enum.BigIntFieldType,
		}}}},
		Bindings: []queryscheme.Binding{{Pointer: "/expressions/0/rules/0/value", Kind: queryscheme.BindingCurrentUser}},
	}
	raw, _ := json.Marshal(payload)
	created, err := service.CreatePersonal(ctx, request.QuerySchemePersonalCreateReq{
		Name: "当前用户", ScopeCode: "system.user.list", Payload: raw,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := service.Resolve(ctx, created.ID, request.QuerySchemeResolveReq{ScopeCode: "system.user.list"})
	if err != nil || resolved.ResolvedQuery == nil || resolved.ResolvedQuery.Expressions[0].Rules[0].Value != float64(101) {
		t.Fatalf("resolve current user: result=%+v err=%v", resolved, err)
	}
	payload.Expressions[0].Rules[0].Field = "removed"
	payload.Bindings = nil
	degradedRaw, _ := json.Marshal(payload)
	if err := db.Model(&model.QueryScheme{}).Where("id = ?", created.ID).
		Update("query_payload", json.RawMessage(degradedRaw)).Error; err != nil {
		t.Fatal(err)
	}
	degraded, err := service.Resolve(ctx, created.ID, request.QuerySchemeResolveReq{ScopeCode: "system.user.list"})
	if err != nil || degraded.ValidationStatus != queryscheme.ValidationDegraded || degraded.ResolvedQuery != nil {
		t.Fatalf("degraded resolve: result=%+v err=%v", degraded, err)
	}
}

func TestQuerySchemeRejectsInvalidBindingWithStableError(t *testing.T) {
	service, _ := newQuerySchemeTestService(t, false)
	payload := queryscheme.QuerySchemePayloadV1{
		Expressions: []request.ExpressionGroup{{Logic: enum.And, Rules: []request.QueryRule{{
			Field: "owner_id", ExpressionType: enum.Eq, Value: 1, Type: enum.BigIntFieldType,
		}}}},
		Bindings: []queryscheme.Binding{{
			Pointer: "/expressions/0/rules/0/value",
			Kind:    queryscheme.BindingKind("CLIENT_FUNCTION"),
		}},
	}
	raw, _ := json.Marshal(payload)
	_, err := service.CreatePersonal(querySchemeContext(101), request.QuerySchemePersonalCreateReq{
		Name: "非法动态条件", ScopeCode: "system.user.list", Payload: raw,
	})
	if !errors.Is(err, myerrors.ErrQuerySchemeBindingInvalid) {
		t.Fatalf("binding error = %v", err)
	}
}

func newQuerySchemeTestService(t *testing.T, manager bool) (*QuerySchemeService, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLite(t,
		&model.SysUser{}, &model.SysRole{}, &model.SysMenu{}, &model.SysUserRole{}, &model.SysRoleMenu{},
		&model.SysMenuButton{}, &model.SysRoleMenuButton{}, &model.OrgEmployee{},
		&model.QueryScheme{}, &model.QuerySchemeRole{},
	)
	users := []model.SysUser{{Basic: model.Basic{Id: 101, State: true}, UserName: "owner"}, {Basic: model.Basic{Id: 202, State: true}, UserName: "ordinary"}}
	roles := []model.SysRole{{Basic: model.Basic{Id: 11, State: true}, Name: "manager"}, {Basic: model.Basic{Id: 22, State: true}, Name: "ordinary"}}
	scopes := []string{"system.user.list", "system.role.list"}
	menus := []model.SysMenu{
		{Basic: model.Basic{Id: 31, State: true}, Name: "system_user", TableCode: "sys_user", QueryScopeCode: &scopes[0]},
		{Basic: model.Basic{Id: 32, State: true}, Name: "system_role", TableCode: "sys_role", QueryScopeCode: &scopes[1]},
	}
	for index := range users {
		testutil.MustCreate(t, db, &users[index])
	}
	for index := range roles {
		testutil.MustCreate(t, db, &roles[index])
	}
	for index := range menus {
		testutil.MustCreate(t, db, &menus[index])
	}
	testutil.MustCreate(t, db, &model.SysUserRole{UserId: 101, RoleId: 11})
	testutil.MustCreate(t, db, &model.SysUserRole{UserId: 202, RoleId: 22})
	for _, roleID := range []int{11, 22} {
		for _, menuID := range []int{31, 32} {
			testutil.MustCreate(t, db, &model.SysRoleMenu{RoleId: roleID, MenuId: menuID})
		}
	}
	if manager {
		button := model.SysMenuButton{Basic: model.Basic{Id: 41, State: true}, MenuId: 31, Code: "scheme_manage", EventAction: queryscheme.SharedManageCapability}
		testutil.MustCreate(t, db, &button)
		testutil.MustCreate(t, db, &model.SysRoleMenuButton{RoleId: 11, MenuId: 31, ButtonId: button.Id})
	}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatal(err)
	}
	reader := querySchemeMetadataStub{table: metadata.TableMetadata{Code: "sys_user", Fields: []metadata.FieldMetadata{
		{Code: "user_name", StorageType: enum.VarcharFieldType, AdvancedQuery: true, Sortable: true},
		{Code: "owner_id", StorageType: enum.BigIntFieldType, AdvancedQuery: true, Sortable: true},
	}}}
	repository := impl.NewQuerySchemeRepositoryImpl(&database.PrimaryDB{DB: db})
	return NewQuerySchemeService(repository, queryscheme.NewRegistry(), reader, sf, nil), db
}

func querySchemeContext(userID int) context.Context {
	return audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(userID, "test"))
}

func querySchemePayloadJSON(t *testing.T) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(queryscheme.QuerySchemePayloadV1{
		Expressions: []request.ExpressionGroup{{Logic: enum.And, Rules: []request.QueryRule{{
			Field: "user_name", ExpressionType: enum.Eq, Value: "nan", Type: enum.VarcharFieldType,
		}}}},
		Order: request.Order{Field: "user_name", IsAsc: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
