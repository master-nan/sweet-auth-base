package service

import (
	"context"
	"encoding/json"
	"testing"

	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository/impl"
	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
)

func TestDocumentFieldScenes(t *testing.T) {
	for _, engine := range []struct {
		name string
		open func(*testing.T) *gorm.DB
	}{
		{"SQLite", func(t *testing.T) *gorm.DB { return testutil.OpenSQLite(t) }},
		{"PostgreSQL", openSysTableRelationPostgreSQL},
	} {
		t.Run(engine.name, func(t *testing.T) {
			t.Run("mixed_added_and_updated_rows", func(t *testing.T) {
				f := newDocumentFieldScenesFixture(t, engine.open(t))
				input := func(label string) map[string]any {
					return map[string]any{
						"label": label, "master_id": 20,
						"insert_locked": label + "-il", "update_locked": label + "-ul",
						"insert_only": label + "-i", "update_only": label + "-u",
						"list_only": label + "-list", "detail_only": label + "-detail",
					}
				}
				id, err := f.save(10, "changed", request.DocumentChanges{
					Added:   []map[string]any{input("added")},
					Updated: []request.DocumentUpdate{{ID: 11, Data: input("edited")}},
				}, request.DocumentChanges{})
				if err != nil || id != 10 {
					t.Fatalf("save mixed changes: id=%d err=%v", id, err)
				}

				type storedLine struct {
					MasterID                   int
					Label                      string
					InsertLocked, UpdateLocked string
					InsertOnly, UpdateOnly     string
					ListOnly, DetailOnly       string
				}
				var got []storedLine
				if err := f.db.Table("doc_lines").Where("id = ? OR label = ?", 11, "added").Order("label").Find(&got).Error; err != nil {
					t.Fatalf("read persisted lines: %v", err)
				}
				want := []storedLine{
					{MasterID: 10, Label: "added", InsertLocked: "from-server", UpdateLocked: "added-ul",
						InsertOnly: "added-i", UpdateOnly: "stored", ListOnly: "stored", DetailOnly: "stored"},
					{MasterID: 10, Label: "edited", InsertLocked: "edited-il", UpdateLocked: "stored",
						InsertOnly: "stored", UpdateOnly: "edited-u", ListOnly: "stored", DetailOnly: "stored"},
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatalf("persisted line fields (-want +got):\n%s", diff)
				}
			})

			t.Run("read_fields_by_scene", func(t *testing.T) {
				f := newDocumentFieldScenesFixture(t, engine.open(t))
				for _, scene := range []struct {
					name      string
					forUpdate bool
					fields    []string
				}{
					{"detail", false, []string{"insert_locked", "update_locked", "insert_only", "detail_only"}},
					{"update", true, []string{"insert_locked", "update_locked", "update_only"}},
				} {
					t.Run(scene.name, func(t *testing.T) {
						got, err := f.svc.ReadDocumentRows(f.ctx, f.lines, map[string]any{"id": 10}, scene.forUpdate)
						if err != nil {
							t.Fatalf("read lines: %v", err)
						}
						want := []map[string]any{{"id": int64(11), "label": "one"}, {"id": int64(12), "label": "two"}}
						for _, row := range want {
							for _, code := range scene.fields {
								row[code] = "stored"
							}
						}
						if diff := cmp.Diff(want, got); diff != "" {
							t.Fatalf("returned line fields (-want +got):\n%s", diff)
						}
					})
				}
			})
		})
	}
}

func newDocumentFieldScenesFixture(t *testing.T, db *gorm.DB) documentFixture {
	t.Helper()
	f := newDocumentFixture(t, db)
	f.seedTwoDocuments(t)
	yes, no, insertDefault := true, false, "from-server"
	for _, field := range []model.SysTableField{
		{FieldCode: "insert_locked", IsInsertShow: true, IsUpdateShow: true, IsInsertReadonly: true, DefaultValue: &insertDefault},
		{FieldCode: "update_locked", IsInsertShow: true, IsUpdateShow: true, IsUpdateReadonly: true},
		{FieldCode: "insert_only", IsInsertShow: true},
		{FieldCode: "update_only", IsUpdateShow: true, DetailVisible: &no},
		{FieldCode: "list_only", IsListShow: true},
		{FieldCode: "detail_only", DetailVisible: &yes},
	} {
		if err := db.Exec("ALTER TABLE doc_lines ADD COLUMN " + field.FieldCode + " TEXT NOT NULL DEFAULT 'stored'").Error; err != nil {
			t.Fatalf("add fixture column %s: %v", field.FieldCode, err)
		}
		field.FieldType = enum.VarcharFieldType
		f.lines.Table.TableFields = append(f.lines.Table.TableFields, field)
	}
	return f
}

func TestDocumentFieldScenesRuntimeResponse(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.SysTable{}, &model.SysTableField{}, &model.SysTableRelation{}, &model.SysTableIndex{}, &model.SysTableIndexField{})
	table := model.SysTable{Basic: model.Basic{Id: 10, State: true}, TableCode: "doc_lines"}
	testutil.MustCreate(t, db, &table)
	primary := &database.PrimaryDB{DB: db}
	fieldRepo := impl.NewSysTableFieldRepositoryImpl(primary)
	yes, no := true, false
	fields := []model.SysTableField{
		metadataRuntimeTestField(101, table.Id, "insert_locked", "Issued", 1),
		metadataRuntimeTestField(102, table.Id, "update_locked", "Review", 2),
	}
	fields[0].IsInsertShow, fields[0].IsInsertReadonly, fields[0].DetailVisible = true, true, &no
	fields[1].IsUpdateShow, fields[1].IsUpdateReadonly = true, true
	if err := fieldRepo.Create(db, &fields); err != nil {
		t.Fatalf("create field metadata: %v", err)
	}
	runtime := NewMetadataRuntimeService(impl.NewSysTableRepositoryImpl(primary), fieldRepo, nil, nil)
	result, err := runtime.GetTableResponse(context.Background(), table.TableCode)
	if err != nil {
		t.Fatalf("get runtime response: %v", err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	type fieldFlags struct {
		Code           string `json:"field_code"`
		Detail         *bool  `json:"detail_visible"`
		InsertReadonly *bool  `json:"is_insert_readonly"`
		UpdateReadonly *bool  `json:"is_update_readonly"`
	}
	var got struct {
		Fields []fieldFlags `json:"table_fields"`
	}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	want := []fieldFlags{{"insert_locked", &no, &yes, &no}, {"update_locked", &yes, &no, &yes}}
	if diff := cmp.Diff(want, got.Fields); diff != "" {
		t.Fatalf("runtime JSON field scenes (-want +got):\n%s", diff)
	}
}

func TestDocumentFieldScenesFieldManagementRoundTripPostgreSQL(t *testing.T) {
	db := openSysTableRelationPostgreSQL(t)
	svc := newSysTableRelationPostgreSQLService(t, db)
	ctx := context.Background()
	table := model.SysTable{Basic: model.Basic{Id: 701, State: true}, TableCode: "scene_field_round_trip", TableType: enum.System}
	testutil.MustCreate(t, db, &table)
	if err := db.Exec("CREATE TABLE scene_field_round_trip (id bigint PRIMARY KEY)").Error; err != nil {
		t.Fatalf("create physical fixture: %v", err)
	}
	before, err := svc.metadataRuntime.GetTableResponse(ctx, table.TableCode)
	if err != nil || len(before.TableFields) != 0 {
		t.Fatalf("warm empty table cache: fields=%v err=%v", before.TableFields, err)
	}

	yes, no := true, false
	create := request.TableFieldCreateReq{
		TableId: table.Id, FieldName: "Line note", FieldCode: "line_note", FieldType: enum.VarcharFieldType,
		FieldLength: 64, InputType: enum.InputType, FieldCategory: enum.NormalField, Sequence: 1,
		IsNull: true, IsInsertShow: true, IsUpdateShow: true,
		DetailVisible: &yes, IsInsertReadonly: true, IsUpdateReadonly: true,
	}
	if err := svc.CreateTableField(ctx, create); err != nil {
		t.Fatalf("create field through management service: %v", err)
	}
	var fieldID int
	if err := db.Model(&model.SysTableField{}).Where("table_id = ? AND field_code = ?", table.Id, create.FieldCode).Select("id").Take(&fieldID).Error; err != nil {
		t.Fatalf("find created field: %v", err)
	}

	type scenes struct{ detail, insertReadonly, updateReadonly bool }
	assertState := func(stage string, sequence int, want scenes) {
		t.Helper()
		check := func(source string, gotSequence uint8, got scenes) {
			t.Helper()
			if int(gotSequence) != sequence || got != want {
				t.Fatalf("%s / %s: sequence=%d scenes=%+v, want sequence=%d scenes=%+v", stage, source, gotSequence, got, sequence, want)
			}
		}
		var stored model.SysTableField
		if err := db.First(&stored, fieldID).Error; err != nil {
			t.Fatalf("%s: reload persisted field: %v", stage, err)
		}
		if stored.DetailVisible == nil {
			t.Fatalf("%s: explicit detail_visible became NULL", stage)
		}
		check("database", stored.Sequence, scenes{*stored.DetailVisible, stored.IsInsertReadonly, stored.IsUpdateReadonly})

		managed, err := svc.GetTableFieldByIdResponse(fieldID)
		if err != nil || managed.DetailVisible == nil {
			t.Fatalf("%s: read field management response: detail=%v err=%v", stage, managed.DetailVisible, err)
		}
		check("field management", managed.Sequence, scenes{*managed.DetailVisible, managed.IsInsertReadonly, managed.IsUpdateReadonly})

		schema, err := svc.metadataRuntime.GetTableResponse(ctx, table.TableCode)
		if err != nil || len(schema.TableFields) != 1 {
			t.Fatalf("%s: read runtime table: fields=%v err=%v", stage, schema.TableFields, err)
		}
		f := schema.TableFields[0]
		if f.Id != fieldID || !f.IsInsertShow || !f.IsUpdateShow {
			t.Fatalf("%s: expected the same field visible in both forms, got %+v", stage, f)
		}
		check("runtime table by code", f.Sequence, scenes{f.DetailVisible, f.IsInsertReadonly, f.IsUpdateReadonly})
		field, err := svc.metadataRuntime.GetField(ctx, fieldID)
		if err != nil {
			t.Fatalf("%s: read runtime field by ID: %v", stage, err)
		}
		check("runtime field by ID", field.Sequence, scenes{field.DetailVisible, field.InsertReadonly, field.UpdateReadonly})
	}
	assertState("created", create.Sequence, scenes{true, true, true})

	update := request.TableFieldUpdateReq{
		Id: fieldID, TableId: table.Id, FieldName: create.FieldName, FieldCode: create.FieldCode,
		FieldType: create.FieldType, FieldLength: create.FieldLength, InputType: create.InputType,
		FieldCategory: create.FieldCategory, IsNull: true, IsInsertShow: true, IsUpdateShow: true,
	}
	for i, step := range []struct {
		name                                   string
		detail, insertReadonly, updateReadonly *bool
		want                                   scenes
	}{
		{"legacy omits all three", nil, nil, nil, scenes{true, true, true}},
		{"clear insert readonly", nil, &no, nil, scenes{true, false, true}},
		{"legacy keeps mixed values", nil, nil, nil, scenes{true, false, true}},
		{"clear detail", &no, nil, nil, scenes{false, false, true}},
		{"clear update readonly", nil, nil, &no, scenes{}},
		{"legacy keeps explicit false", nil, nil, nil, scenes{}},
	} {
		// The sequence change proves an old-style update actually wrote and refreshed the field.
		update.Sequence = create.Sequence + i + 1
		update.DetailVisible, update.IsInsertReadonly, update.IsUpdateReadonly = step.detail, step.insertReadonly, step.updateReadonly
		if err := svc.UpdateTableField(ctx, update); err != nil {
			t.Fatalf("%s: update field through management service: %v", step.name, err)
		}
		assertState(step.name, update.Sequence, step.want)
	}
}
