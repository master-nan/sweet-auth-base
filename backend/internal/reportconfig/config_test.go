package reportconfig

import (
	"strings"
	"testing"

	"gorm.io/datatypes"
)

func TestParseReportConfigWithDatasetJoins(t *testing.T) {
	config, err := Parse(
		datatypes.JSON([]byte(`{
			"datasets":[
				{"id":"waybill","name":"运单","type":"table","source_code":"tms_waybill","primary":true},
				{"id":"company","name":"公司","type":"table","source_code":"tms_company"}
			],
			"dataset_joins":[
				{"id":"j1","left_dataset_id":"waybill","right_dataset_id":"company","join_type":"left","conditions":[{"left_field":"company_id","right_field":"id"},{"left_field":"tenant_id","right_field":"tenant_id"}]}
			]
		}`)),
		datatypes.JSON([]byte(`{"view":"sheet","sheet":{"rows":8,"cols":6,"column_widths":{"2":160},"row_heights":{"3":56},"cells":[]}}`)),
	)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	dataset, ok := config.PrimaryTableDataset()
	if !ok || dataset.Id != "waybill" || dataset.SourceCode != "tms_waybill" {
		t.Fatalf("unexpected primary dataset: %#v ok=%v", dataset, ok)
	}
	joins := config.DatasetJoins()
	if len(joins) != 1 || len(joins[0].Conditions) != 2 || joins[0].Conditions[0].LeftField != "company_id" || joins[0].Conditions[1].RightField != "tenant_id" {
		t.Fatalf("unexpected joins: %#v", joins)
	}
	if config.Layout.Sheet.ColumnWidths["2"] != 160 || config.Layout.Sheet.RowHeights["3"] != 56 {
		t.Fatalf("unexpected sheet sizes: %#v %#v", config.Layout.Sheet.ColumnWidths, config.Layout.Sheet.RowHeights)
	}
}

func TestParseReportConfigRejectsSingleFieldJoin(t *testing.T) {
	_, err := Parse(
		datatypes.JSON([]byte(`{
			"datasets":[
				{"id":"main","type":"table","source_code":"demo_order","primary":true},
				{"id":"company","type":"table","source_code":"demo_company"}
			],
			"dataset_joins":[{"left_dataset_id":"main","left_field":"company_id","right_dataset_id":"company","right_field":"id","join_type":"left"}]
		}`)),
		datatypes.JSON([]byte(`{"view":"sheet"}`)),
	)
	if err == nil || !strings.Contains(err.Error(), "关联配置不完整") {
		t.Fatalf("expected single-field join rejection, got %v", err)
	}
}

func TestParseReportConfigRejectsBadDataset(t *testing.T) {
	_, err := Parse(
		datatypes.JSON([]byte(`{"datasets":[{"id":"bad","type":"sql"}]}`)),
		datatypes.JSON([]byte(`{"view":"sheet"}`)),
	)
	if err == nil || !strings.Contains(err.Error(), "SQL 数据集缺少 sql") {
		t.Fatalf("expected sql validation error, got %v", err)
	}
}

func TestParseReportConfigRejectsBadDatasetJoin(t *testing.T) {
	_, err := Parse(
		datatypes.JSON([]byte(`{
			"datasets":[{"id":"main","type":"table","source_code":"demo_order","primary":true}],
			"dataset_joins":[{"left_dataset_id":"main","right_dataset_id":"missing","join_type":"left","conditions":[{"left_field":"company_id","right_field":"id"}]}]
		}`)),
		datatypes.JSON([]byte(`{"view":"sheet"}`)),
	)
	if err == nil || !strings.Contains(err.Error(), "右侧数据集不存在") {
		t.Fatalf("expected join validation error, got %v", err)
	}
}
