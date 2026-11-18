package metadata

import (
	"testing"

	"backend/enum"
	"backend/model"
)

func TestDocumentFieldScenesRuntimeAndQueryModel(t *testing.T) {
	type scenes struct {
		list, detail                   bool
		insert, update                 bool
		insertReadonly, updateReadonly bool
	}
	yes, no := true, false
	tests := []struct {
		field model.SysTableField
		want  scenes
	}{
		{model.SysTableField{FieldCode: "list_only", IsListShow: true}, scenes{list: true}},
		{model.SysTableField{FieldCode: "detail_only", DetailVisible: &yes}, scenes{detail: true}},
		{model.SysTableField{FieldCode: "insert_locked", IsInsertShow: true, IsUpdateShow: true, IsInsertReadonly: true, DetailVisible: &no},
			scenes{insert: true, update: true, insertReadonly: true}},
		{model.SysTableField{FieldCode: "update_locked", IsUpdateShow: true, IsInsertReadonly: true, IsUpdateReadonly: true},
			scenes{detail: true, update: true, updateReadonly: true}},
		{model.SysTableField{FieldCode: "insert_only", IsInsertShow: true, IsUpdateReadonly: true},
			scenes{detail: true, insert: true}},
		{model.SysTableField{FieldCode: "gmt_create", IsListShow: true, IsInsertShow: true, IsUpdateShow: true, IsInsertReadonly: true, IsUpdateReadonly: true},
			scenes{detail: true}},
	}
	source := model.SysTable{Basic: model.Basic{Id: 10, State: true}, TableCode: "doc_lines"}
	for i, tc := range tests {
		f := tc.field
		f.Basic = model.Basic{Id: 100 + i, State: true}
		f.TableId, f.FieldType, f.Sequence = source.Id, enum.VarcharFieldType, uint8(i+1)
		source.TableFields = append(source.TableFields, f)
	}
	runtime := ProjectTable(source)
	query := runtime.QueryModel()
	if len(runtime.Fields) != len(tests) || len(query.TableFields) != len(tests) {
		t.Fatalf("field counts: runtime=%d query=%d want=%d", len(runtime.Fields), len(query.TableFields), len(tests))
	}
	for i, tc := range tests {
		t.Run(tc.field.FieldCode, func(t *testing.T) {
			f, q := runtime.Fields[i], query.TableFields[i]
			if f.Code != tc.field.FieldCode || q.FieldCode != tc.field.FieldCode || q.DetailVisible == nil {
				t.Fatalf("missing field or unresolved query visibility: runtime=%s query=%s detail=%v", f.Code, q.FieldCode, q.DetailVisible)
			}
			if got := (scenes{f.ListVisible, f.DetailVisible, f.InsertVisible, f.UpdateVisible, f.InsertReadonly, f.UpdateReadonly}); got != tc.want {
				t.Errorf("runtime scenes: got %+v, want %+v", got, tc.want)
			}
			if got := (scenes{q.IsListShow, *q.DetailVisible, q.IsInsertShow, q.IsUpdateShow, q.IsInsertReadonly, q.IsUpdateReadonly}); got != tc.want {
				t.Errorf("query scenes: got %+v, want %+v", got, tc.want)
			}
		})
	}
}
