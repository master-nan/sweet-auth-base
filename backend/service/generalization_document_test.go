package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	"backend/internal/datapermission"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
)

// SQLite runs by default. PostgreSQL also exercises the production row-locking SQL.
func TestGeneralizationSaveDocument(t *testing.T) {
	for _, engine := range []struct {
		name string
		open func(*testing.T) *gorm.DB
	}{
		{"SQLite", func(t *testing.T) *gorm.DB { return testutil.OpenSQLite(t) }},
		{"PostgreSQL", openSysTableRelationPostgreSQL},
	} {
		t.Run(engine.name, func(t *testing.T) {
			t.Run("reads_only_own_lines", func(t *testing.T) {
				f := newDocumentFixture(t, engine.open(t))
				f.seedTwoDocuments(t)
				master := map[string]any{"id": 10}
				rows, err := f.svc.ReadDocumentRows(f.ctx, f.lines, master, false)
				if err != nil || len(rows) != 2 || rows[0]["label"] != "one" || rows[1]["label"] != "two" {
					t.Fatalf("read goods: rows=%v err=%v", rows, err)
				}
				fees, err := f.svc.ReadDocumentRows(f.ctx, f.fees, master, false)
				if err != nil || len(fees) != 0 {
					t.Fatalf("read fees: rows=%v err=%v", fees, err)
				}
			})
			t.Run("commits_multiple_sections", func(t *testing.T) {
				f := newDocumentFixture(t, engine.open(t))
				id, err := f.save(0, "original",
					request.DocumentChanges{Added: []map[string]any{{"label": "keep", "master_id": -1}, {"label": "remove"}}},
					request.DocumentChanges{Added: []map[string]any{{"label": "shipping"}}})
				if err != nil || id <= 0 {
					t.Fatalf("create document: id=%d, err=%v", id, err)
				}
				created := f.snapshot(t)
				assertDocumentMasters(t, created.Masters, documentTestRow{ID: id, Label: "original"})
				assertDocumentChildren(t, created.Lines, id, "keep", "remove")
				assertDocumentChildren(t, created.Fees, id, "shipping")

				editedID, err := f.save(id, "edited", request.DocumentChanges{
					Updated: []request.DocumentUpdate{{ID: created.Lines[0].ID, Data: map[string]any{"label": "kept", "master_id": -1}}},
					Deleted: []int{created.Lines[1].ID},
				}, request.DocumentChanges{Added: []map[string]any{{"label": "tax"}}})
				if err != nil || editedID != id {
					t.Fatalf("edit document %d: id=%d, err=%v", id, editedID, err)
				}
				edited := f.snapshot(t)
				assertDocumentMasters(t, edited.Masters, documentTestRow{ID: id, Label: "edited"})
				assertDocumentChildren(t, edited.Lines, id, "kept")
				assertDocumentChildren(t, edited.Fees, id, "shipping", "tax")
				if edited.Lines[0].ID != created.Lines[0].ID || edited.Fees[0] != created.Fees[0] {
					t.Fatal("edit replaced the updated line or changed the untouched fee")
				}
			})

			for _, tc := range []struct {
				name string
				id   int
			}{{"create", 0}, {"edit", 10}} {
				t.Run("last_row_rolls_back_"+tc.name, func(t *testing.T) {
					f := newDocumentFixture(t, engine.open(t))
					lines := request.DocumentChanges{Added: []map[string]any{{"label": "new line"}}}
					if tc.id != 0 {
						f.seedTwoDocuments(t)
						lines.Updated = []request.DocumentUpdate{{ID: 11, Data: map[string]any{"label": "changed"}}}
						lines.Deleted = []int{12}
					}
					before := f.snapshot(t)
					// Only the second fee can violate this constraint: the first fee has to reach the DB.
					id, err := f.save(tc.id, "new title", lines, request.DocumentChanges{
						Added: []map[string]any{{"label": "tax"}, {"label": "tax"}},
					})
					if id != 0 || err == nil || !(strings.Contains(err.Error(), "doc_fees.master_id, doc_fees.label") ||
						strings.Contains(err.Error(), "doc_fees_unique_label")) {
						t.Fatalf("expected the last fee's unique constraint, got id=%d, err=%v", id, err)
					}
					f.assertUnchanged(t, before)
				})
			}

			for _, op := range []string{"update", "delete"} {
				t.Run("rejects_foreign_"+op, func(t *testing.T) {
					f := newDocumentFixture(t, engine.open(t))
					f.seedTwoDocuments(t)
					before := f.snapshot(t)
					foreign := request.DocumentChanges{}
					if op == "update" {
						foreign.Updated = []request.DocumentUpdate{{ID: 21, Data: map[string]any{"label": "changed"}}}
					} else {
						foreign.Deleted = []int{21}
					}
					id, err := f.save(10, "edited", request.DocumentChanges{Deleted: []int{11}}, foreign)
					if id != 0 || !errors.Is(err, apperrors.ErrDataNotFound) {
						t.Fatalf("foreign fee %s: id=%d, err=%v; want not found", op, id, err)
					}
					f.assertUnchanged(t, before)

					id, err = f.save(20, "other", request.DocumentChanges{}, foreign)
					if err != nil || id != 20 {
						t.Fatalf("same fee %s on its owning document: id=%d, err=%v", op, id, err)
					}
					after := f.snapshot(t)
					if op == "update" {
						assertDocumentChildren(t, after.Fees, 20, "changed")
					} else {
						assertDocumentChildren(t, after.Fees, 20)
					}
				})
			}
		})
	}
}

func TestDocumentTabConfiguration(t *testing.T) {
	db := openSysTableRelationPostgreSQL(t)
	svc := newSysTableRelationPostgreSQLService(t, db)
	ctx := context.Background()
	for id, code := range []string{"orders", "goods"} {
		testutil.MustCreate(t, db, &model.SysTable{Basic: model.Basic{Id: id + 1, State: true}, TableCode: code, TableName: code, TableType: enum.System})
	}
	for i, code := range []string{"id", "order_id", "quantity", "price"} {
		tableID, typ := 2, enum.DecimalFieldType
		if i < 2 {
			typ = enum.BigIntFieldType
		}
		if i == 0 {
			tableID = 1
		}
		testutil.MustCreate(t, db, &model.SysTableField{Basic: model.Basic{Id: 10 + i, State: true}, TableId: tableID, FieldName: code, FieldCode: code, FieldType: typ, IsPrimaryKey: i == 0})
	}
	req := request.TableRelationCreateReq{TableId: 1, RelatedTableId: 2, ReferenceKey: "id", ForeignKey: "order_id", RelationType: enum.OneToMany, DocumentTab: "商品明细", QuantityField: "quantity", PriceField: "price"}
	bad := req
	bad.AmountField = "price"
	if err := svc.CreateTableRelation(ctx, bad); err == nil {
		t.Fatal("accepted conflicting amount rules")
	}
	if err := svc.CreateTableRelation(ctx, req); err != nil {
		t.Fatal(err)
	}
	schema, err := svc.metadataRuntime.GetTableResponse(ctx, "orders")
	if err != nil || len(schema.TableRelations) != 1 {
		t.Fatalf("read configuration: %+v, %v", schema, err)
	}
	tab := schema.TableRelations[0]
	if tab.DocumentTab != req.DocumentTab || tab.RelatedTableCode != "goods" || tab.PriceField != "price" || tab.QuantityField != "quantity" {
		t.Fatalf("unexpected tab: %+v", tab)
	}
	if err := svc.UpdateTableRelation(ctx, request.TableRelationUpdateReq{Id: tab.Id, TableId: 1, RelatedTableId: 2, ReferenceKey: "id", ForeignKey: "order_id", RelationType: enum.OneToMany}); err != nil {
		t.Fatal(err)
	}
	schema, err = svc.metadataRuntime.GetTableResponse(ctx, "orders")
	if err != nil || len(schema.TableRelations) != 1 || schema.TableRelations[0].DocumentTab != "" || schema.TableRelations[0].PriceField != "" {
		t.Fatalf("tab configuration was not cleared: %+v, %v", schema, err)
	}
}

type documentFixture struct {
	db          *gorm.DB
	svc         *GeneralizationService
	ctx         context.Context
	master      model.SysTable
	lines, fees DocumentSection
}

func newDocumentFixture(t *testing.T, db *gorm.DB) documentFixture {
	t.Helper()
	for _, ddl := range []string{
		"CREATE TABLE doc_masters (id BIGINT PRIMARY KEY, label TEXT NOT NULL)",
		"CREATE TABLE doc_lines (id BIGINT PRIMARY KEY, master_id BIGINT NOT NULL REFERENCES doc_masters(id), label TEXT NOT NULL)",
		"CREATE TABLE doc_fees (id BIGINT PRIMARY KEY, master_id BIGINT NOT NULL REFERENCES doc_masters(id), label TEXT NOT NULL, CONSTRAINT doc_fees_unique_label UNIQUE(master_id, label))",
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create document fixture: %v", err)
		}
	}
	runtime := newLowCodeDataPermissionRuntime(
		func(context.Context, int) ([]model.DataResource, error) { return nil, nil },
		func(context.Context, int) ([]model.DataOwnershipField, error) { return nil, nil },
		func(context.Context, int) (datapermission.SubjectContext, error) {
			return datapermission.SubjectContext{}, nil
		},
		func(context.Context, datapermission.ResolverInput) (datapermission.DataScopeResult, error) {
			return datapermission.DataScopeResult{}, nil
		},
		func(_ context.Context, in datapermission.AdapterInput) (datapermission.AdapterExecution, error) {
			return datapermission.BuildAdapterExecution(in)
		},
	)
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatal(err)
	}
	table := func(id int, code string) model.SysTable {
		return model.SysTable{Basic: model.Basic{Id: id, State: true}, TableCode: code, TableFields: []model.SysTableField{
			{FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true},
			{FieldCode: "label", FieldType: enum.VarcharFieldType, IsInsertShow: true, IsUpdateShow: true},
		}}
	}
	master := table(1, "doc_masters")
	section := func(id int, code string) DocumentSection {
		child := table(id, code)
		child.TableFields = append(child.TableFields, model.SysTableField{FieldCode: "master_id", FieldType: enum.BigIntFieldType, IsInsertShow: true, IsUpdateShow: true})
		relation := model.SysTableRelation{Basic: model.Basic{Id: id, State: true}, TableId: master.Id, RelatedTableId: id,
			ReferenceKey: "id", ForeignKey: "master_id", RelationType: enum.OneToMany, DocumentTab: code}
		master.TableRelations = append(master.TableRelations, relation)
		return DocumentSection{Relation: relation, Table: child}
	}
	lines, fees := section(2, "doc_lines"), section(3, "doc_fees")
	ctx, cancel := context.WithTimeout(testUserContext(), 10*time.Second)
	t.Cleanup(cancel)
	return documentFixture{db: db, ctx: ctx, master: master, lines: lines, fees: fees,
		svc: newGeneralizationService(impl.NewGeneralizationRepositoryImpl(&database.PrimaryDB{DB: db}), sf, runtime)}
}

func (f documentFixture) save(id int, label string, lines, fees request.DocumentChanges) (int, error) {
	f.lines.Changes, f.fees.Changes = lines, fees
	return f.svc.SaveDocument(f.ctx, f.master, id, map[string]any{"label": label}, []DocumentSection{f.lines, f.fees})
}

func (f documentFixture) seedTwoDocuments(t *testing.T) {
	t.Helper()
	for _, sql := range []string{
		"INSERT INTO doc_masters VALUES (10, 'first'), (20, 'other')",
		"INSERT INTO doc_lines VALUES (11, 10, 'one'), (12, 10, 'two')",
		"INSERT INTO doc_fees VALUES (21, 20, 'other fee')",
	} {
		if err := f.db.Exec(sql).Error; err != nil {
			t.Fatalf("seed documents: %v", err)
		}
	}
}

type documentTestRow struct {
	ID, MasterID int
	Label        string
}

type documentSnapshot struct {
	Masters, Lines, Fees []documentTestRow
}

func (f documentFixture) snapshot(t *testing.T) documentSnapshot {
	t.Helper()
	read := func(table string) []documentTestRow {
		t.Helper()
		var rows []documentTestRow
		if err := f.db.Table(table).Order("label, id").Find(&rows).Error; err != nil {
			t.Fatalf("read persisted %s: %v", table, err)
		}
		return rows
	}
	return documentSnapshot{Masters: read("doc_masters"), Lines: read("doc_lines"), Fees: read("doc_fees")}
}

func (f documentFixture) assertUnchanged(t *testing.T, before documentSnapshot) {
	t.Helper()
	if diff := cmp.Diff(before, f.snapshot(t)); diff != "" {
		t.Fatalf("persisted document state after rejection (-before +after):\n%s", diff)
	}
}

func assertDocumentMasters(t *testing.T, rows []documentTestRow, want documentTestRow) {
	t.Helper()
	if diff := cmp.Diff([]documentTestRow{want}, rows); diff != "" {
		t.Fatalf("persisted master (-want +got):\n%s", diff)
	}
}

func assertDocumentChildren(t *testing.T, rows []documentTestRow, masterID int, labels ...string) {
	t.Helper()
	if len(rows) != len(labels) {
		t.Fatalf("persisted children = %+v; want %v for master %d", rows, labels, masterID)
	}
	for i, row := range rows {
		if row.ID <= 0 || row.MasterID != masterID || row.Label != labels[i] {
			t.Fatalf("persisted child = %+v; want %q for master %d", row, labels[i], masterID)
		}
	}
}
