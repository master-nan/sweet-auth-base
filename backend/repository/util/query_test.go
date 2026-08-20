package util

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"backend/dto/request"
	"backend/enum"
	"backend/internal/datapermission"
	"backend/internal/queryscheme"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestExecuteQueryRejectsUnknownFilterFields(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Filters: map[string]any{
			"unknown_field": "demo",
		},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))

	if strings.Contains(sql, "unknown_field") {
		t.Fatalf("query leaked unknown filter field into SQL: %s", sql)
	}
	if !strings.Contains(sql, "1 = 0") {
		t.Fatalf("query did not block unknown filter field: %s", sql)
	}
}

func TestGetSQLTypeUsesPostgresTypes(t *testing.T) {
	cases := []struct {
		name      string
		fieldType enum.SysTableFieldType
		length    int
		decimal   int
		want      string
	}{
		{name: "datetime", fieldType: enum.DatetimeFieldType, want: "timestamp"},
		{name: "json", fieldType: enum.JsonFieldType, want: "jsonb"},
		{name: "smallint", fieldType: enum.SmallIntFieldType, want: "smallint"},
		{name: "int", fieldType: enum.IntFieldType, want: "integer"},
		{name: "decimal", fieldType: enum.DecimalFieldType, length: 12, decimal: 2, want: "numeric(12,2)"},
		{name: "varchar default length", fieldType: enum.VarcharFieldType, want: "varchar(255)"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			field := model.SysTableField{FieldType: tt.fieldType, FieldLength: tt.length, NumericPrecision: tt.length, NumericScale: tt.decimal}
			if got := getSQLType(field); got != tt.want {
				t.Fatalf("getSQLType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExecuteQueryRejectsUnknownExpressionFields(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "unknown_field",
						ExpressionType: enum.Eq,
						Value:          "demo",
						Type:           enum.VarcharFieldType,
					},
				},
			},
		},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))

	if strings.Contains(sql, "unknown_field") {
		t.Fatalf("query leaked unknown expression field into SQL: %s", sql)
	}
	if !strings.Contains(sql, "1 = 0") {
		t.Fatalf("query did not block unknown expression field: %s", sql)
	}
}

func TestExecuteQueryIgnoresUnknownOrderFields(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Order: request.Order{Field: "unknown_field", IsAsc: false},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))

	if strings.Contains(sql, "unknown_field") {
		t.Fatalf("query leaked unknown order field into SQL: %s", sql)
	}
	if strings.Contains(strings.ToLower(sql), "order by") {
		t.Fatalf("query should ignore unknown order field: %s", sql)
	}
}

func TestExecuteQueryRejectsSensitiveFilterFields(t *testing.T) {
	db := dryRunDB(t)
	table := sensitiveQueryTestTable()
	basic := &request.Basic{
		Filters: map[string]any{
			"password": "secret",
		},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))

	if strings.Contains(sql, "password") {
		t.Fatalf("query leaked sensitive filter field into SQL: %s", sql)
	}
	if !strings.Contains(sql, "1 = 0") {
		t.Fatalf("query did not block sensitive filter field: %s", sql)
	}
}

func TestExecuteQuerySkipsSensitiveQuickSearchFields(t *testing.T) {
	db := dryRunDB(t)
	table := sensitiveQueryTestTable()
	basic := &request.Basic{
		QuickQuery: &request.QuickQuery{Keyword: "secret"},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))

	if strings.Contains(sql, "password") || strings.Contains(sql, "access_token") {
		t.Fatalf("query leaked sensitive quick-search fields into SQL: %s", sql)
	}
	if !strings.Contains(sql, "name") {
		t.Fatalf("query should still search ordinary quick-search field: %s", sql)
	}
}

func TestExecuteQueryIgnoresSensitiveOrderFields(t *testing.T) {
	db := dryRunDB(t)
	table := sensitiveQueryTestTable()
	basic := &request.Basic{
		Order: request.Order{Field: "password", IsAsc: false},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))

	if strings.Contains(sql, "password") {
		t.Fatalf("query leaked sensitive order field into SQL: %s", sql)
	}
	if strings.Contains(strings.ToLower(sql), "order by") {
		t.Fatalf("query should ignore sensitive order field: %s", sql)
	}
}

func TestExecuteQuerySplitsStringInExpressionValues(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "name",
						ExpressionType: enum.In,
						Value:          "alice, bob\ncarol",
						Type:           enum.VarcharFieldType,
					},
				},
			},
		},
	}

	stmt := renderQueryStatement(ExecuteQuery(db.Table(table.TableCode), basic, table))
	sql := stmt.SQL.String()

	if !strings.Contains(sql, " IN ") {
		t.Fatalf("query should use IN expression: %s", sql)
	}
	assertVars(t, stmt.Vars, []interface{}{"alice", "bob", "carol"})
}

func TestExecuteQueryBuildsMultiKeywordLikeExpression(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "name",
						ExpressionType: enum.Like,
						Value:          []interface{}{"role", "admin"},
						Type:           enum.VarcharFieldType,
					},
				},
			},
		},
	}

	stmt := renderQueryStatement(ExecuteQuery(db.Table(table.TableCode), basic, table))
	sql := stmt.SQL.String()

	if !strings.Contains(sql, " LIKE ") || !strings.Contains(sql, " OR ") {
		t.Fatalf("multi-keyword LIKE should use OR LIKE expressions: %s", sql)
	}
	assertVars(t, stmt.Vars, []interface{}{"%role%", "%admin%"})
}

func TestExecuteQueryCastsNumericLikeFieldToText(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "id",
						ExpressionType: enum.Like,
						Value:          "43",
						Type:           enum.BigIntFieldType,
					},
				},
			},
		},
	}

	stmt := renderQueryStatement(ExecuteQuery(db.Table(table.TableCode), basic, table))
	sql := stmt.SQL.String()

	if !strings.Contains(sql, "CAST(") || !strings.Contains(sql, " LIKE ") {
		t.Fatalf("numeric LIKE should cast field to text: %s", sql)
	}
	assertVars(t, stmt.Vars, []interface{}{"%43%"})
}

func TestExecuteQueryCastsQuickSearchFieldsToText(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	table.TableFields[0].IsQuickSearch = true
	basic := &request.Basic{
		QuickQuery: &request.QuickQuery{Keyword: "43"},
	}

	stmt := renderQueryStatement(ExecuteQuery(db.Table(table.TableCode), basic, table))
	sql := stmt.SQL.String()

	if !strings.Contains(sql, "CAST(") || !strings.Contains(sql, " LIKE ") {
		t.Fatalf("quick search should cast searchable fields to text: %s", sql)
	}
	assertVars(t, stmt.Vars, []interface{}{"%43%"})
}

func TestExecuteQueryBuildsMultiKeywordNotLikeExpression(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "name",
						ExpressionType: enum.NotLike,
						Value:          "role; admin",
						Type:           enum.VarcharFieldType,
					},
				},
			},
		},
	}

	stmt := renderQueryStatement(ExecuteQuery(db.Table(table.TableCode), basic, table))
	sql := stmt.SQL.String()

	if !strings.Contains(sql, " NOT LIKE ") || !strings.Contains(sql, " AND ") {
		t.Fatalf("multi-keyword NOT LIKE should use AND NOT LIKE expressions: %s", sql)
	}
	assertVars(t, stmt.Vars, []interface{}{"%role%", "%admin%"})
}

func TestExecuteQueryParsesNumericInExpressionValues(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "id",
						ExpressionType: enum.In,
						Value:          []interface{}{"1", "2", float64(3)},
						Type:           enum.BigIntFieldType,
					},
				},
			},
		},
	}

	stmt := renderQueryStatement(ExecuteQuery(db.Table(table.TableCode), basic, table))

	assertVars(t, stmt.Vars, []interface{}{1, 2, 3})
}

func TestExecuteQueryRejectsInvalidTypedExpressionValues(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	tests := []struct {
		name  string
		field string
		value interface{}
	}{
		{name: "integer fraction", field: "id", value: "1.5"},
		{name: "boolean text", field: "enabled", value: "maybe"},
		{name: "date text", field: "biz_date", value: "2026/06/06"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basic := &request.Basic{
				Expressions: []request.ExpressionGroup{
					{
						Logic: enum.And,
						Rules: []request.QueryRule{
							{
								Field:          tt.field,
								ExpressionType: enum.Eq,
								Value:          tt.value,
								Type:           enum.VarcharFieldType,
							},
						},
					},
				},
			}

			sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))
			if !strings.Contains(sql, "1 = 0") {
				t.Fatalf("invalid typed expression should match nothing: %s", sql)
			}
		})
	}
}

func TestExecuteQueryRejectsInvalidTypedInExpressionValues(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "id",
						ExpressionType: enum.In,
						Value:          []interface{}{"1", "bad"},
						Type:           enum.BigIntFieldType,
					},
				},
			},
		},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))
	if !strings.Contains(sql, "1 = 0") {
		t.Fatalf("invalid typed IN expression should match nothing: %s", sql)
	}
}

func TestExecuteQueryRejectsInvalidTypedFilterValues(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Filters: map[string]any{
			"enabled": "maybe",
		},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))
	if !strings.Contains(sql, "1 = 0") {
		t.Fatalf("invalid typed filter should match nothing: %s", sql)
	}
}

func TestExecuteQueryEmptyInExpressionMatchesNothing(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "name",
						ExpressionType: enum.In,
						Value:          " , ， ; ",
						Type:           enum.VarcharFieldType,
					},
				},
			},
		},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))

	if !strings.Contains(sql, "1 = 0") {
		t.Fatalf("empty IN expression should match nothing: %s", sql)
	}
}

func TestExecuteQueryBuildsTypedBetweenExpression(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "biz_date",
						ExpressionType: enum.Between,
						Value:          []interface{}{"2026-06-01", "2026-06-30"},
						Type:           enum.DateFieldType,
					},
				},
			},
		},
	}

	stmt := renderQueryStatement(ExecuteQuery(db.Table(table.TableCode), basic, table))
	sql := stmt.SQL.String()

	if !strings.Contains(sql, " BETWEEN ") {
		t.Fatalf("query should use BETWEEN expression: %s", sql)
	}
	if len(stmt.Vars) != 2 {
		t.Fatalf("expected 2 range vars, got %#v", stmt.Vars)
	}
}

func TestExecuteQueryRejectsInvalidBetweenExpressionValues(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "biz_date",
						ExpressionType: enum.Between,
						Value:          []interface{}{"2026-06-01"},
						Type:           enum.DateFieldType,
					},
				},
			},
		},
	}

	sql := renderQuerySQL(ExecuteQuery(db.Table(table.TableCode), basic, table))
	if !strings.Contains(sql, "1 = 0") {
		t.Fatalf("invalid BETWEEN expression should match nothing: %s", sql)
	}
}

func TestExecuteQueryNormalizesTimeExpressionValues(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.And,
				Rules: []request.QueryRule{
					{
						Field:          "start_time",
						ExpressionType: enum.Gte,
						Value:          "09:30",
						Type:           enum.TimeFieldType,
					},
				},
			},
		},
	}

	stmt := renderQueryStatement(ExecuteQuery(db.Table(table.TableCode), basic, table))
	sql := stmt.SQL.String()

	if !strings.Contains(sql, "start_time") || !strings.Contains(sql, ">= ?") {
		t.Fatalf("query should use time comparison: %s", sql)
	}
	assertVars(t, stmt.Vars, []interface{}{"09:30:00"})
}

func TestExecuteQueryBuildsNestedOrExpression(t *testing.T) {
	db := dryRunDB(t)
	table := queryTestTable()
	basic := &request.Basic{
		Expressions: []request.ExpressionGroup{
			{
				Logic: enum.Or,
				Rules: []request.QueryRule{
					{
						Field:          "status",
						ExpressionType: enum.Eq,
						Value:          2,
						Type:           enum.SmallIntFieldType,
					},
				},
				Nested: []request.ExpressionGroup{
					{
						Logic: enum.And,
						Rules: []request.QueryRule{
							{
								Field:          "enabled",
								ExpressionType: enum.Eq,
								Value:          true,
								Type:           enum.BooleanFieldType,
							},
							{
								Field:          "biz_date",
								ExpressionType: enum.Gte,
								Value:          "2026-07-01",
								Type:           enum.DateFieldType,
							},
						},
					},
				},
			},
		},
	}

	stmt := renderQueryStatement(ExecuteQuery(db.Table(table.TableCode), basic, table))
	sql := stmt.SQL.String()

	if !strings.Contains(sql, " OR ") {
		t.Fatalf("nested query should preserve OR logic: %s", sql)
	}
	if !strings.Contains(sql, "status") || !strings.Contains(sql, "enabled") || !strings.Contains(sql, "biz_date") {
		t.Fatalf("nested query lost conditions: %s", sql)
	}
	if len(stmt.Vars) != 3 {
		t.Fatalf("expected status/enabled/date vars, got %#v in %s", stmt.Vars, sql)
	}
}

func TestListResultFieldsSkipsSensitiveFieldsEvenWhenVisible(t *testing.T) {
	fields := listResultFields(sensitiveQueryTestTable().TableFields)

	for _, field := range fields {
		if field.FieldCode == "password" || field.FieldCode == "access_token" {
			t.Fatalf("sensitive field should not be selected for list response: %+v", fields)
		}
	}
	if len(fields) != 2 || fields[0].FieldCode != "id" || fields[1].FieldCode != "name" {
		t.Fatalf("unexpected list result fields: %+v", fields)
	}
}

func TestListResultFieldsKeepsHiddenPrimaryKeyForRowActions(t *testing.T) {
	table := model.SysTable{
		TableCode: "demo_table",
		TableFields: []model.SysTableField{
			{FieldCode: "id", FieldType: enum.BigIntFieldType, IsListShow: false, IsPrimaryKey: true},
			{FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true},
		},
	}

	fields := listResultFields(table.TableFields)

	if len(fields) != 2 || fields[0].FieldCode != "id" || fields[1].FieldCode != "name" {
		t.Fatalf("hidden primary key should be selected before visible fields, got %+v", fields)
	}
}

func TestListResultFieldsRejectsArbitraryCalculatedExpressions(t *testing.T) {
	unsafe := "(select password from sys_user limit 1)"
	safe := "rel:org_unit.name"
	table := model.SysTable{
		TableCode: "demo_table",
		TableFields: []model.SysTableField{
			{FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true},
			{FieldCode: "unsafe", FieldType: enum.VarcharFieldType, IsListShow: true, FieldCategory: enum.CalculatedField, Expression: &unsafe},
			{FieldCode: "org_name", FieldType: enum.VarcharFieldType, IsListShow: true, FieldCategory: enum.VirtualField, Expression: &safe},
		},
	}

	fields := listResultFields(table.TableFields)
	if len(fields) != 2 || fields[0].FieldCode != "id" || fields[1].FieldCode != "org_name" {
		t.Fatalf("unsafe calculated field reached runtime selection: %+v", fields)
	}
}

func TestDynamicQueryDoesNotDuplicateRowsForOneToManyRelations(t *testing.T) {
	db := testutil.OpenSQLite(t)
	if err := db.Exec("CREATE TABLE sys_table (id INTEGER PRIMARY KEY, table_code TEXT)").Error; err != nil {
		t.Fatalf("create sys_table: %v", err)
	}
	if err := db.Exec("CREATE TABLE demo_parent (id INTEGER PRIMARY KEY, name TEXT)").Error; err != nil {
		t.Fatalf("create parent table: %v", err)
	}
	if err := db.Exec("CREATE TABLE demo_child (id INTEGER PRIMARY KEY, parent_id INTEGER, name TEXT)").Error; err != nil {
		t.Fatalf("create child table: %v", err)
	}
	if err := db.Exec("INSERT INTO sys_table (id, table_code) VALUES (1, 'demo_parent'), (2, 'demo_child')").Error; err != nil {
		t.Fatalf("seed table metadata: %v", err)
	}
	if err := db.Exec("INSERT INTO demo_parent (id, name) VALUES (1, 'A'), (2, 'B')").Error; err != nil {
		t.Fatalf("seed parent rows: %v", err)
	}
	if err := db.Exec("INSERT INTO demo_child (id, parent_id, name) VALUES (10, 1, 'A1'), (11, 1, 'A2'), (12, 2, 'B1')").Error; err != nil {
		t.Fatalf("seed child rows: %v", err)
	}

	table := model.SysTable{
		TableCode: "demo_parent",
		TableFields: []model.SysTableField{
			{FieldCode: "id", FieldType: enum.BigIntFieldType, IsListShow: true, IsPrimaryKey: true},
			{FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true},
		},
		TableRelations: []model.SysTableRelation{
			{
				TableId:        1,
				RelatedTableId: 2,
				ReferenceKey:   "id",
				ForeignKey:     "parent_id",
				RelationType:   enum.OneToMany,
			},
		},
	}
	result, err := DynamicQuery(db, &request.Basic{Page: 1, Num: 15}, table)
	if err != nil {
		t.Fatalf("dynamic query failed: %v", err)
	}
	if result.Total != 2 || len(result.Data) != 2 {
		t.Fatalf("one-to-many relation should not duplicate parent rows, total=%d data=%+v", result.Total, result.Data)
	}
}

func TestDynamicQueryWithPermissionKeepsRowsAndTotalConsistent(t *testing.T) {
	db := generalizationPermissionTestDB(t)
	table := generalizationPermissionTestTable()

	tests := []struct {
		name      string
		basic     *request.Basic
		execution datapermission.AdapterExecution
		wantIds   []int64
		wantTotal int
	}{
		{
			name:      "allow all",
			basic:     &request.Basic{Page: 1, Num: 2, Order: request.Order{Field: "id", IsAsc: true}},
			execution: mustQueryAdapterExecution(t, datapermission.DataScopeDecisionAll, nil),
			wantIds:   []int64{1, 2},
			wantTotal: 4,
		},
		{
			name:      "deny all",
			basic:     &request.Basic{Page: 1, Num: 10},
			execution: mustQueryAdapterExecution(t, datapermission.DataScopeDecisionNone, nil),
			wantIds:   nil,
			wantTotal: 0,
		},
		{
			name:  "single filtered condition",
			basic: &request.Basic{Page: 1, Num: 10, Order: request.Order{Field: "id", IsAsc: true}},
			execution: mustQueryAdapterExecution(t, datapermission.DataScopeDecisionFiltered, [][]datapermission.DataScopeCondition{
				{mustQueryScopeCondition(t, "owner_org", 101, datapermission.DataScopeOperatorIn, []any{int64(11)})},
			}),
			wantIds:   []int64{1, 2},
			wantTotal: 2,
		},
		{
			name:  "conditions inside a group use AND",
			basic: &request.Basic{Page: 1, Num: 10, Order: request.Order{Field: "id", IsAsc: true}},
			execution: mustQueryAdapterExecution(t, datapermission.DataScopeDecisionFiltered, [][]datapermission.DataScopeCondition{
				{
					mustQueryScopeCondition(t, "owner_org", 101, datapermission.DataScopeOperatorIn, []any{int64(11)}),
					mustQueryScopeCondition(t, "legal_entity", 102, datapermission.DataScopeOperatorEqual, []any{int64(100)}),
				},
			}),
			wantIds:   []int64{1},
			wantTotal: 1,
		},
		{
			name:  "condition groups use OR",
			basic: &request.Basic{Page: 1, Num: 10, Order: request.Order{Field: "id", IsAsc: true}},
			execution: mustQueryAdapterExecution(t, datapermission.DataScopeDecisionFiltered, [][]datapermission.DataScopeCondition{
				{mustQueryScopeCondition(t, "owner_org", 101, datapermission.DataScopeOperatorEqual, []any{int64(12)})},
				{mustQueryScopeCondition(t, "legal_entity", 102, datapermission.DataScopeOperatorEqual, []any{int64(200)})},
			}),
			wantIds:   []int64{2, 3},
			wantTotal: 2,
		},
		{
			name: "user query is AND with the whole permission tree",
			basic: &request.Basic{
				Page:    1,
				Num:     10,
				Order:   request.Order{Field: "id", IsAsc: true},
				Filters: map[string]any{"name": "alpha"},
			},
			execution: mustQueryAdapterExecution(t, datapermission.DataScopeDecisionFiltered, [][]datapermission.DataScopeCondition{
				{mustQueryScopeCondition(t, "owner_org", 101, datapermission.DataScopeOperatorEqual, []any{int64(11)})},
				{mustQueryScopeCondition(t, "legal_entity", 102, datapermission.DataScopeOperatorEqual, []any{int64(300)})},
			}),
			wantIds:   []int64{1},
			wantTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permission := repository.GeneralizationPermission{
				AdapterExecution: &tt.execution,
			}
			result, err := DynamicQueryWithPermission(db, tt.basic, table, permission)
			if err != nil {
				t.Fatalf("query with permission: %v", err)
			}
			if result.Total != tt.wantTotal {
				t.Fatalf("total = %d, want %d; rows=%+v", result.Total, tt.wantTotal, result.Data)
			}
			if got := generalizationPermissionResultIds(t, result.Data); !reflect.DeepEqual(got, tt.wantIds) {
				t.Fatalf("ids = %v, want %v", got, tt.wantIds)
			}
		})
	}
}

func TestDynamicQueryWithPermissionKeepsNotApplicableUnfiltered(t *testing.T) {
	db := generalizationPermissionTestDB(t)
	table := generalizationPermissionTestTable()
	execution := mustQueryAdapterExecution(t, datapermission.DataScopeDecisionNotApplicable, nil)

	result, err := DynamicQueryWithPermission(
		db,
		&request.Basic{Page: 1, Num: 10},
		table,
		repository.GeneralizationPermission{
			AdapterExecution: &execution,
		},
	)
	if err != nil {
		t.Fatalf("not_applicable query: %v", err)
	}
	if result.Total != 4 {
		t.Fatalf("not_applicable total = %d, want 4", result.Total)
	}
}

func TestResolvedQuerySchemeIsANDedWithDataPermission(t *testing.T) {
	db := generalizationPermissionTestDB(t)
	table := generalizationPermissionTestTable()
	resolved := queryscheme.ResolvedQuery{Expressions: []request.ExpressionGroup{{
		Logic: enum.And,
		Rules: []request.QueryRule{{Field: "name", ExpressionType: enum.Eq, Value: "alpha", Type: enum.VarcharFieldType}},
	}}}
	execution := mustQueryAdapterExecution(t, datapermission.DataScopeDecisionFiltered, [][]datapermission.DataScopeCondition{{
		mustQueryScopeCondition(t, "owner_org", 101, datapermission.DataScopeOperatorEqual, []any{int64(11)}),
	}})
	result, err := DynamicQueryWithPermission(db, &request.Basic{
		Page: 1, Num: 10, Expressions: resolved.Expressions, QuickQuery: &resolved.QuickQuery, Order: resolved.Order,
	}, table, repository.GeneralizationPermission{AdapterExecution: &execution})
	if err != nil {
		t.Fatal(err)
	}
	if ids := generalizationPermissionResultIds(t, result.Data); !reflect.DeepEqual(ids, []int64{1}) {
		t.Fatalf("scheme query must narrow, never replace, data permission: ids=%v", ids)
	}
}

func TestGetFieldTypeUsesStringForTimeFields(t *testing.T) {
	if got := GetFieldType(enum.TimeFieldType); got != reflect.TypeOf("") {
		t.Fatalf("TIME fields should scan into string, got %v", got)
	}
	if got := GetFieldType(enum.DatetimeFieldType); got != reflect.TypeOf(time.Time{}) {
		t.Fatalf("DATETIME fields should scan into time.Time, got %v", got)
	}
}

func dryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}
	return db
}

func sensitiveQueryTestTable() model.SysTable {
	return model.SysTable{
		TableCode: "demo_table",
		TableFields: []model.SysTableField{
			{FieldCode: "id", FieldType: enum.BigIntFieldType, IsListShow: true, IsPrimaryKey: true},
			{FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true, IsQuickSearch: true},
			{FieldCode: "password", FieldType: enum.VarcharFieldType, IsListShow: true, IsQuickSearch: true, IsSort: true},
			{FieldCode: "access_token", FieldType: enum.TextFieldType, IsListShow: true, IsQuickSearch: true},
		},
	}
}

func queryTestTable() model.SysTable {
	return model.SysTable{
		TableCode: "demo_table",
		TableFields: []model.SysTableField{
			{FieldCode: "id", FieldType: enum.BigIntFieldType, IsListShow: true, IsPrimaryKey: true},
			{FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true},
			{FieldCode: "enabled", FieldType: enum.BooleanFieldType, IsListShow: true},
			{FieldCode: "biz_date", FieldType: enum.DateFieldType, IsListShow: true},
			{FieldCode: "start_time", FieldType: enum.TimeFieldType, IsListShow: true},
			{FieldCode: "status", FieldType: enum.SmallIntFieldType, IsListShow: true},
		},
	}
}

func generalizationPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.OpenSQLiteWithConfig(t, &gorm.Config{})
	if err := db.Exec(`CREATE TABLE permission_demo (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		owner_org_id INTEGER NOT NULL,
		legal_entity_id INTEGER NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create permission demo table: %v", err)
	}
	if err := db.Exec(`INSERT INTO permission_demo (id, name, owner_org_id, legal_entity_id) VALUES
		(1, 'alpha', 11, 100),
		(2, 'beta', 11, 200),
		(3, 'alpha', 12, 100),
		(4, 'gamma', 13, 300)`).Error; err != nil {
		t.Fatalf("seed permission demo table: %v", err)
	}
	return db
}

func generalizationPermissionTestTable() model.SysTable {
	return model.SysTable{
		Basic:     model.Basic{Id: 9001, State: true},
		TableCode: "permission_demo",
		TableFields: []model.SysTableField{
			{Basic: model.Basic{Id: 1, State: true}, TableId: 9001, FieldCode: "id", FieldType: enum.BigIntFieldType, IsListShow: true, IsPrimaryKey: true, IsSort: true},
			{Basic: model.Basic{Id: 2, State: true}, TableId: 9001, FieldCode: "name", FieldType: enum.VarcharFieldType, IsListShow: true},
			{Basic: model.Basic{Id: 501, State: true}, TableId: 9001, FieldCode: "owner_org_id", FieldType: enum.BigIntFieldType, IsListShow: true, IsAdvancedSearch: true},
			{Basic: model.Basic{Id: 502, State: true}, TableId: 9001, FieldCode: "legal_entity_id", FieldType: enum.BigIntFieldType, IsListShow: true, IsAdvancedSearch: true},
		},
	}
}

func mustQueryScopeCondition(
	t *testing.T,
	ownershipCode string,
	dimensionId int,
	operator datapermission.DataScopeOperator,
	values []any,
) datapermission.DataScopeCondition {
	t.Helper()
	condition, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: ownershipCode,
		DimensionId:   dimensionId,
		Operator:      operator,
		ValueType:     datapermission.DataScopeValueTypeBigint,
		Values:        values,
	})
	if err != nil {
		t.Fatalf("create data scope condition: %v", err)
	}
	return condition
}

func mustQueryAdapterExecution(
	t *testing.T,
	decision datapermission.DataScopeDecision,
	conditionGroups [][]datapermission.DataScopeCondition,
) datapermission.AdapterExecution {
	t.Helper()
	resource, err := datapermission.NewAdapterResourceContext(datapermission.AdapterResourceContextInput{
		ResourceCode: "permission_demo",
		Operation:    model.DataPermissionOperationQuery,
		AdapterCode:  "metadata_filter",
		TableId:      9001,
	})
	if err != nil {
		t.Fatalf("create adapter resource: %v", err)
	}
	groups := make([]datapermission.DataScopeConditionGroup, 0, len(conditionGroups))
	for _, conditions := range conditionGroups {
		group, groupErr := datapermission.NewDataScopeConditionGroup(conditions)
		if groupErr != nil {
			t.Fatalf("create data scope condition group: %v", groupErr)
		}
		groups = append(groups, group)
	}
	result, err := datapermission.NewDataScopeResult(datapermission.DataScopeResultInput{
		ResourceCode:    resource.ResourceCode(),
		Operation:       resource.Operation(),
		Decision:        decision,
		ConditionGroups: groups,
	})
	if err != nil {
		t.Fatalf("create data scope result: %v", err)
	}
	definitions := make([]datapermission.AdapterOwnershipDefinition, 0, 2)
	for _, input := range []datapermission.AdapterOwnershipDefinitionInput{
		{OwnershipCode: "owner_org", DimensionId: 101, BindingType: datapermission.AdapterBindingTypeMetadataField, TableFieldId: 501, ValueType: datapermission.DataScopeValueTypeBigint},
		{OwnershipCode: "legal_entity", DimensionId: 102, BindingType: datapermission.AdapterBindingTypeMetadataField, TableFieldId: 502, ValueType: datapermission.DataScopeValueTypeBigint},
	} {
		definition, definitionErr := datapermission.NewAdapterOwnershipDefinition(input)
		if definitionErr != nil {
			t.Fatalf("create adapter ownership: %v", definitionErr)
		}
		definitions = append(definitions, definition)
	}
	input, err := datapermission.NewAdapterInput(resource, result, definitions)
	if err != nil {
		t.Fatalf("create adapter input: %v", err)
	}
	execution, err := datapermission.BuildAdapterExecution(input)
	if err != nil {
		t.Fatalf("build adapter execution: %v", err)
	}
	return execution
}

func generalizationPermissionResultIds(t *testing.T, rows []map[string]interface{}) []int64 {
	t.Helper()
	if len(rows) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		switch value := row["id"].(type) {
		case int64:
			ids = append(ids, value)
		case int:
			ids = append(ids, int64(value))
		default:
			t.Fatalf("unexpected id type %T in row %+v", row["id"], row)
		}
	}
	return ids
}

func renderQuerySQL(query *gorm.DB) string {
	return renderQueryStatement(query).SQL.String()
}

func renderQueryStatement(query *gorm.DB) *gorm.Statement {
	var rows []map[string]any
	stmt := query.Find(&rows).Statement
	return stmt
}

func assertVars(t *testing.T, actual []interface{}, expected []interface{}) {
	t.Helper()
	if len(actual) == 1 {
		if values, ok := actual[0].([]interface{}); ok {
			actual = values
		}
	}
	if len(actual) != len(expected) {
		t.Fatalf("expected vars %v, got %v", expected, actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("expected vars %v, got %v", expected, actual)
		}
	}
}
