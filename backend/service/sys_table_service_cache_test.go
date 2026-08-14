package service

import (
	"backend/config"
	"backend/dto/request"
	"backend/enum"
	"backend/internal/cache"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository/impl"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestUpdateTableRejectsStableCodeMutationAndRefreshesCache(t *testing.T) {
	db := sysTableCacheTestDB(t)
	table := model.SysTable{
		Basic:     model.Basic{Id: 10, State: true},
		TableName: "Orders",
		TableCode: "orders",
		TableType: enum.System,
		TableFields: []model.SysTableField{
			{
				Basic:     model.Basic{Id: 101, State: true},
				TableId:   10,
				FieldName: "Name",
				FieldCode: "name",
				FieldType: enum.VarcharFieldType,
				InputType: enum.InputType,
				Sequence:  1,
			},
		},
	}
	if err := db.Create(&table).Error; err != nil {
		t.Fatalf("seed sys table: %v", err)
	}
	svc, tableCache, _ := newSysTableCacheTestService(db)
	if err := tableCache.Set(strconvKey(table.Id), table); err != nil {
		t.Fatalf("seed table id cache: %v", err)
	}
	if err := tableCache.Set(table.TableCode, table); err != nil {
		t.Fatalf("seed table code cache: %v", err)
	}

	if err := svc.UpdateTable(ginTestContext(), request.TableUpdateReq{
		Id:        table.Id,
		TableName: "Orders V2",
		TableCode: "orders_v2",
	}); err == nil {
		t.Fatal("expected stable table_code mutation to be rejected")
	}

	if !tableCache.Exists("orders") {
		t.Fatal("expected stable table_code cache to remain available")
	}
	if tableCache.Exists("orders_v2") {
		t.Fatal("unexpected cache entry for rejected table_code")
	}

	if err := svc.UpdateTable(ginTestContext(), request.TableUpdateReq{
		Id:        table.Id,
		TableName: "Orders V2",
		TableCode: "orders",
	}); err != nil {
		t.Fatalf("update table name: %v", err)
	}
	updated, err := tableCache.Get("orders")
	if err != nil {
		t.Fatalf("expected stable table_code cache to be refreshed: %v", err)
	}
	if updated.TableCode != "orders" || updated.TableName != "Orders V2" {
		t.Fatalf("unexpected refreshed table cache: %+v", updated)
	}
}

func TestRefreshCacheInvalidatesRemovedFieldCache(t *testing.T) {
	db := sysTableCacheTestDB(t)
	currentField := model.SysTableField{
		Basic:     model.Basic{Id: 201, State: true},
		TableId:   20,
		FieldName: "Name",
		FieldCode: "name",
		FieldType: enum.VarcharFieldType,
		InputType: enum.InputType,
		Sequence:  1,
	}
	removedField := model.SysTableField{
		Basic:     model.Basic{Id: 202, State: true},
		TableId:   20,
		FieldName: "Removed",
		FieldCode: "removed",
		FieldType: enum.VarcharFieldType,
		InputType: enum.InputType,
		Sequence:  2,
	}
	table := model.SysTable{
		Basic:       model.Basic{Id: 20, State: true},
		TableName:   "Customers",
		TableCode:   "customers",
		TableType:   enum.System,
		TableFields: []model.SysTableField{currentField},
	}
	if err := db.Create(&table).Error; err != nil {
		t.Fatalf("seed sys table: %v", err)
	}

	svc, tableCache, fieldCache := newSysTableCacheTestService(db)
	stale := table
	stale.TableFields = []model.SysTableField{currentField, removedField}
	if err := tableCache.Set(strconvKey(stale.Id), stale); err != nil {
		t.Fatalf("seed stale table cache: %v", err)
	}
	if err := fieldCache.Set(strconvKey(currentField.Id), currentField); err != nil {
		t.Fatalf("seed current field cache: %v", err)
	}
	if err := fieldCache.Set(strconvKey(removedField.Id), removedField); err != nil {
		t.Fatalf("seed removed field cache: %v", err)
	}

	svc.RefreshCache(table.Id)

	if fieldCache.Exists(strconvKey(removedField.Id)) {
		t.Fatal("expected removed field cache to be invalidated")
	}
	if !fieldCache.Exists(strconvKey(currentField.Id)) {
		t.Fatal("expected current field cache to be refreshed")
	}
}

func sysTableCacheTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.OpenSQLite(t,
		&model.SysTable{},
		&model.SysTableField{},
		&model.SysTableRelation{},
		&model.SysTableIndex{},
		&model.SysTableIndexField{},
	)
	return db
}

func newSysTableCacheTestService(db *gorm.DB) (*SysTableService, *cache.SysTableCache, *cache.SysTableFieldCache) {
	store := newJSONMemoryCacher()
	tableCache := cache.NewSysTableCache(store)
	fieldCache := cache.NewSysTableFieldCache(store)
	primaryDB := &database.PrimaryDB{DB: db}
	tableRepo := impl.NewSysTableRepositoryImpl(primaryDB)
	metadataRuntime := NewMetadataRuntimeService(
		tableRepo,
		impl.NewSysTableFieldRepositoryImpl(primaryDB),
		tableCache,
		fieldCache,
	)
	return NewSysTableService(
		tableRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		metadataRuntime,
		&config.Server{},
	), tableCache, fieldCache
}

func ginTestContext() *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("user", model.SysUser{Basic: model.Basic{Id: 1}, UserName: "admin"})
	return ctx
}

func strconvKey(value int) string {
	return strconv.Itoa(value)
}

type jsonMemoryCacher struct {
	values map[string][]byte
}

func newJSONMemoryCacher() *jsonMemoryCacher {
	return &jsonMemoryCacher{values: map[string][]byte{}}
}

func (c *jsonMemoryCacher) Get(key string, value interface{}) error {
	data, ok := c.values[key]
	if !ok {
		return cache.ErrCacheMiss
	}
	return json.Unmarshal(data, value)
}

func (c *jsonMemoryCacher) Set(key string, value interface{}, _ time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.values[key] = data
	return nil
}

func (c *jsonMemoryCacher) Del(key string) error {
	delete(c.values, key)
	return nil
}

func (c *jsonMemoryCacher) Exists(keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		if _, ok := c.values[key]; ok {
			count++
		}
	}
	return count, nil
}

func (c *jsonMemoryCacher) Expire(key string, _ time.Duration) (bool, error) {
	_, ok := c.values[key]
	return ok, nil
}
