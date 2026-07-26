package service

import (
	"backend/dto/request"
	"backend/dto/response"
	testutil "backend/internal/test"
	"backend/model"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestOrgServiceFourSelectorOptionsShareProtocolAndBoundedQueries(t *testing.T) {
	t.Run("legal entity", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		currentA := orgServiceLegalEntity(1, "LE-A", "搜索法人甲", "法人甲", "enabled", nil, nil)
		currentB := orgServiceLegalEntity(2, "LE-B", "搜索法人乙", "法人乙", "enabled", nil, nil)
		disabled := orgServiceLegalEntity(3, "LE-D", "停用法人", "停用", "disabled", nil, nil)
		expired := orgServiceLegalEntity(4, "LE-H", "历史法人", "历史", "enabled", nil, nil)
		expiredAt := model.Now().Add(-24 * time.Hour)
		expired.ValidTo = &expiredAt
		testutil.MustCreate(
			t,
			db,
			&[]model.OrgLegalEntity{currentA, currentB, disabled, expired},
		)
		queryCount := registerSelectorQueryCounter(t, db)

		result, err := orgService.QueryLegalEntityOptions(
			nil,
			request.OrgLegalEntityOptionsReq{
				OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
					Page:        1,
					Num:         1,
					Keyword:     "搜索法人",
					SelectedIds: []int{disabled.Id, expired.Id},
				},
			},
			orgServiceLegalEntityTable(),
		)
		if err != nil {
			t.Fatalf("query legal entity selector options: %v", err)
		}
		if result.Total != 2 || len(result.Items) != 3 {
			t.Fatalf("legal entity selector result=%+v", result)
		}
		assertSelectorSelectableItem(t, result.Items[0], map[int]string{
			currentA.Id: "LE-A - 搜索法人甲",
			currentB.Id: "LE-B - 搜索法人乙",
		})
		assertSelectorReplayItems(t, result.Items[1:], disabled.Id, expired.Id)
		assertSelectorQueryCount(t, queryCount.Load())
	})

	t.Run("organization unit", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		legal := orgServiceLegalEntity(10, "LE-U", "组织所属法人", "法人", "enabled", nil, nil)
		current := managementUnitFixture(11, "OU-A", "搜索组织", "department", "enabled", &legal.Id)
		disabled := managementUnitFixture(12, "OU-D", "停用组织", "department", "disabled", &legal.Id)
		testutil.MustCreate(t, db, &legal)
		testutil.MustCreate(t, db, &[]model.OrgUnit{current, disabled})
		queryCount := registerSelectorQueryCounter(t, db)

		result, err := orgService.QueryOrgUnitOptions(
			nil,
			request.OrgUnitOptionsReq{
				OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
					Page:        1,
					Num:         10,
					Keyword:     "搜索组织",
					SelectedIds: []int{disabled.Id},
				},
				LegalEntityId: &legal.Id,
			},
			managementUnitTable(),
		)
		if err != nil {
			t.Fatalf("query org unit selector options: %v", err)
		}
		assertSelectorResult(t, result, current.Id, "OU-A - 搜索组织", disabled.Id)
		assertSelectorQueryCount(t, queryCount.Load())
	})

	t.Run("employee", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		current := orgServiceEmployeeFixture(21, "EMP-A", "搜索人员", "active")
		disabled := orgServiceEmployeeFixture(22, "EMP-D", "离职人员", "resigned")
		testutil.MustCreate(t, db, &[]model.OrgEmployee{current, disabled})
		queryCount := registerSelectorQueryCounter(t, db)

		result, err := orgService.QueryEmployeeOptions(
			nil,
			request.OrgEmployeeOptionsReq{
				OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
					Page:        1,
					Num:         10,
					Keyword:     "搜索人员",
					SelectedIds: []int{disabled.Id},
				},
			},
			orgEmployeePositionServiceTable("org_employee"),
		)
		if err != nil {
			t.Fatalf("query employee selector options: %v", err)
		}
		assertSelectorResult(t, result, current.Id, "EMP-A - 搜索人员", disabled.Id)
		assertSelectorQueryCount(t, queryCount.Load())
	})

	t.Run("position", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		unit := managementUnitFixture(31, "OU-P", "岗位所属组织", "department", "enabled", nil)
		current := orgServicePositionFixture(32, "POS-A", "搜索岗位", unit.Id, "enabled")
		disabled := orgServicePositionFixture(33, "POS-D", "停用岗位", unit.Id, "disabled")
		testutil.MustCreate(t, db, &unit)
		testutil.MustCreate(t, db, &[]model.OrgPosition{current, disabled})
		queryCount := registerSelectorQueryCounter(t, db)

		result, err := orgService.QueryPositionOptions(
			nil,
			request.OrgPositionOptionsReq{
				OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
					Page:        1,
					Num:         10,
					Keyword:     "搜索岗位",
					SelectedIds: []int{disabled.Id},
				},
				OrgUnitId: &unit.Id,
			},
			orgEmployeePositionServiceTable("org_position"),
		)
		if err != nil {
			t.Fatalf("query position selector options: %v", err)
		}
		assertSelectorResult(t, result, current.Id, "POS-A - 搜索岗位", disabled.Id)
		assertSelectorQueryCount(t, queryCount.Load())
	})
}

func registerSelectorQueryCounter(t *testing.T, db *gorm.DB) *atomic.Int32 {
	t.Helper()
	var queryCount atomic.Int32
	callbackName := "test:count-selector-queries"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(*gorm.DB) {
		queryCount.Add(1)
	}); err != nil {
		t.Fatalf("register selector query counter: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})
	return &queryCount
}

func assertSelectorResult(
	t *testing.T,
	result response.OrgSelectorOptionsRes,
	activeId int,
	activeLabel string,
	replayId int,
) {
	t.Helper()
	if result.Total != 1 || len(result.Items) != 2 {
		t.Fatalf("selector result=%+v, want one selectable and one replay item", result)
	}
	assertSelectorSelectableItem(t, result.Items[0], map[int]string{activeId: activeLabel})
	assertSelectorReplayItems(t, result.Items[1:], replayId)
}

func assertSelectorSelectableItem(
	t *testing.T,
	item response.OrgSelectorOptionRes,
	allowed map[int]string,
) {
	t.Helper()
	label, exists := allowed[item.Value]
	if !exists || item.Label != label || item.Disabled {
		t.Fatalf("invalid selectable selector item: %+v", item)
	}
	if item.Value <= 0 {
		t.Fatalf("selector value is not an internal numeric ID: %+v", item)
	}
}

func assertSelectorReplayItems(
	t *testing.T,
	items []response.OrgSelectorOptionRes,
	expectedIds ...int,
) {
	t.Helper()
	if len(items) != len(expectedIds) {
		t.Fatalf("replay item count=%d, want %d: %+v", len(items), len(expectedIds), items)
	}
	for index, expectedId := range expectedIds {
		if items[index].Value != expectedId || !items[index].Disabled {
			t.Fatalf("invalid selector replay item %d: %+v", index, items[index])
		}
	}
}

func assertSelectorQueryCount(t *testing.T, count int32) {
	t.Helper()
	if count != 3 {
		t.Fatalf("selector query count=%d, want count+page+one selected_ids batch", count)
	}
}
