package service

import (
	"backend/dto/request"
	"backend/internal/cache"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"errors"
	"strconv"
	"testing"

	"gorm.io/gorm"
)

func newSysDictTestService(t *testing.T) (*SysDictService, *jsonMemoryCacher, *database.PrimaryDB) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.SysDict{}, &model.SysDictItem{})
	primary := &database.PrimaryDB{DB: db}
	store := newJSONMemoryCacher()
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	return NewSysDictService(
		impl.NewSysDictRepositoryImpl(primary),
		impl.NewSysDictItemRepositoryImpl(primary),
		sf,
		cache.NewSysDictCache(store),
	), store, primary
}

func seedSysDictTestData(t *testing.T, primary *database.PrimaryDB) (model.SysDict, model.SysDictItem) {
	t.Helper()
	dict := model.SysDict{Basic: model.Basic{Id: 101, State: true}, DictName: "测试字典", DictCode: "test_dict"}
	item := model.SysDictItem{Basic: model.Basic{Id: 201, State: true}, DictId: dict.Id, ItemName: "项目", ItemCode: "test_item", ItemValue: "item"}
	if err := primary.DB.Create(&dict).Error; err != nil {
		t.Fatalf("seed dict: %v", err)
	}
	if err := primary.DB.Create(&item).Error; err != nil {
		t.Fatalf("seed dict item: %v", err)
	}
	return dict, item
}

func primeSysDictAliases(t *testing.T, service *SysDictService, dict model.SysDict) {
	t.Helper()
	if _, err := service.getSysDictByID(dict.Id); err != nil {
		t.Fatalf("prime id cache: %v", err)
	}
	if _, err := service.getSysDictByCode(dict.DictCode); err != nil {
		t.Fatalf("prime code cache: %v", err)
	}
}

func TestSysDictUpdateInvalidatesIDAndCodeAliases(t *testing.T) {
	service, store, primary := newSysDictTestService(t)
	dict, _ := seedSysDictTestData(t, primary)
	primeSysDictAliases(t, service, dict)

	if err := service.UpdateSysDict(context.Background(), request.DictUpdateReq{Id: dict.Id, DictName: "新名称"}); err != nil {
		t.Fatalf("update dict: %v", err)
	}
	for _, key := range []string{cache.DictCacheKey + strconv.Itoa(dict.Id), cache.DictCacheKey + dict.DictCode} {
		if _, ok := store.values[key]; ok {
			t.Fatalf("cache alias %q remains after update", key)
		}
	}
}

func TestSysDictItemDeleteInvalidatesParentAliases(t *testing.T) {
	service, store, primary := newSysDictTestService(t)
	dict, item := seedSysDictTestData(t, primary)
	primeSysDictAliases(t, service, dict)

	if err := service.DeleteSysDictItemById(context.Background(), item.Id); err != nil {
		t.Fatalf("delete dict item: %v", err)
	}
	for _, key := range []string{cache.DictCacheKey + strconv.Itoa(dict.Id), cache.DictCacheKey + dict.DictCode} {
		if _, ok := store.values[key]; ok {
			t.Fatalf("parent cache alias %q remains after item delete", key)
		}
	}
}

func TestSysDictDeletePreservesMutationError(t *testing.T) {
	service, _, primary := newSysDictTestService(t)
	dict, _ := seedSysDictTestData(t, primary)
	want := errors.New("delete failed")
	if err := primary.DB.Callback().Delete().Before("gorm:delete").Register("test:fail-dict-delete", func(tx *gorm.DB) {
		tx.AddError(want)
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}

	err := service.DeleteSysDictById(context.Background(), dict.Id)
	if !errors.Is(err, want) {
		t.Fatalf("delete error = %v, want %v", err, want)
	}
}
