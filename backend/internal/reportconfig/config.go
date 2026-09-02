package reportconfig

import (
	myerrors "backend/internal/errors"
	"encoding/json"
	"strings"

	"gorm.io/datatypes"
)

const (
	SourceTypeTable = "table"
	SourceTypeSQL   = "sql"
)

type Config struct {
	Query  QueryConfig
	Layout LayoutConfig
}

type QueryConfig struct {
	Datasets     []Dataset     `json:"datasets"`
	DatasetJoins []DatasetJoin `json:"dataset_joins"`
	Fields       []Field       `json:"fields"`
	Parameters   []Parameter   `json:"parameters"`
}

type LayoutConfig struct {
	View            string        `json:"view"`
	Title           string        `json:"title"`
	Subtitle        string        `json:"subtitle"`
	Kind            string        `json:"kind"`
	Datasets        []Dataset     `json:"datasets"`
	DatasetJoins    []DatasetJoin `json:"dataset_joins"`
	Parameters      []Parameter   `json:"parameters"`
	Sheet           SheetConfig   `json:"sheet"`
	RuntimeDisplay  string        `json:"runtime_display"`
	RuntimePageSize int           `json:"runtime_page_size"`
}

type Dataset struct {
	Id         string  `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	SourceCode string  `json:"source_code"`
	SQL        string  `json:"sql"`
	Fields     []Field `json:"fields"`
	Primary    bool    `json:"primary"`
}

type DatasetJoin struct {
	Id             string `json:"id"`
	LeftDatasetId  string `json:"left_dataset_id"`
	LeftField      string `json:"left_field"`
	RightDatasetId string `json:"right_dataset_id"`
	RightField     string `json:"right_field"`
	JoinType       string `json:"join_type"`
}

type Field struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	Field     string `json:"field"`
	Label     string `json:"label"`
	Type      string `json:"type"`
	Role      string `json:"role"`
	Aggregate string `json:"aggregate"`
	Selected  bool   `json:"selected"`
}

type Parameter struct {
	Id           string `json:"id"`
	Label        string `json:"label"`
	DatasetId    string `json:"dataset_id"`
	Field        string `json:"field"`
	Type         string `json:"type"`
	Operator     string `json:"operator"`
	Placeholder  string `json:"placeholder"`
	DefaultValue any    `json:"default_value"`
}

type SheetConfig struct {
	Rows             int            `json:"rows"`
	Cols             int            `json:"cols"`
	Scale            float64        `json:"scale"`
	ActiveCell       string         `json:"active_cell"`
	DetailRows       []int          `json:"detail_rows"`
	SummaryRows      []int          `json:"summary_rows"`
	GroupSummaryRows []int          `json:"group_summary_rows"`
	ColumnWidths     map[string]int `json:"column_widths"`
	RowHeights       map[string]int `json:"row_heights"`
	Cells            []SheetCell    `json:"cells"`
}

type SheetCell struct {
	Id      string         `json:"id"`
	Row     int            `json:"row"`
	Col     int            `json:"col"`
	Value   string         `json:"value"`
	Binding CellBinding    `json:"binding"`
	Style   map[string]any `json:"style"`
	Colspan int            `json:"colspan"`
	Rowspan int            `json:"rowspan"`
}

type CellBinding struct {
	Type      string `json:"type"`
	DatasetId string `json:"dataset_id"`
	Field     string `json:"field"`
	Formula   string `json:"formula"`
}

func Parse(queryRaw datatypes.JSON, layoutRaw datatypes.JSON) (Config, error) {
	var config Config
	if err := unmarshalJSON(queryRaw, &config.Query); err != nil {
		return config, myerrors.NewValidationError("报表 query_config 结构不合法")
	}
	if err := unmarshalJSON(layoutRaw, &config.Layout); err != nil {
		return config, myerrors.NewValidationError("报表 layout_config 结构不合法")
	}
	if err := validateDatasets(config.Datasets()); err != nil {
		return config, err
	}
	if err := validateDatasetJoins(config.Datasets(), config.DatasetJoins()); err != nil {
		return config, err
	}
	return config, nil
}

func unmarshalJSON(raw datatypes.JSON, target any) error {
	if len(raw) == 0 {
		return nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func (c Config) Datasets() []Dataset {
	if len(c.Layout.Datasets) > 0 {
		return c.Layout.Datasets
	}
	return c.Query.Datasets
}

func (c Config) DatasetJoins() []DatasetJoin {
	if len(c.Layout.DatasetJoins) > 0 {
		return c.Layout.DatasetJoins
	}
	return c.Query.DatasetJoins
}

func (c Config) Parameters() []Parameter {
	if len(c.Layout.Parameters) > 0 {
		return c.Layout.Parameters
	}
	return c.Query.Parameters
}

func (c Config) HasDatasets() bool {
	return len(c.Layout.Datasets) > 0 || len(c.Query.Datasets) > 0
}

func (c Config) PrimaryTableDataset() (Dataset, bool) {
	for _, dataset := range c.Datasets() {
		if dataset.Primary && NormalizeDatasetType(dataset.Type) == SourceTypeTable {
			return NormalizeDataset(dataset), true
		}
	}
	return Dataset{}, false
}

func (c Config) DatasetByID(id string) (Dataset, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Dataset{}, false
	}
	for _, dataset := range c.Datasets() {
		if strings.TrimSpace(dataset.Id) == id {
			return NormalizeDataset(dataset), true
		}
	}
	return Dataset{}, false
}

func validateDatasets(datasets []Dataset) error {
	for _, dataset := range datasets {
		normalized := NormalizeDataset(dataset)
		switch normalized.Type {
		case SourceTypeTable:
			if normalized.SourceCode == "" {
				return myerrors.NewValidationError("报表表数据集缺少 source_code")
			}
		case SourceTypeSQL:
			if normalized.SQL == "" {
				return myerrors.NewValidationError("报表 SQL 数据集缺少 sql")
			}
		default:
			return myerrors.NewValidationError("报表数据集类型不合法")
		}
	}
	return nil
}

func validateDatasetJoins(datasets []Dataset, joins []DatasetJoin) error {
	if len(joins) == 0 {
		return nil
	}
	datasetByID := make(map[string]Dataset, len(datasets))
	for _, dataset := range datasets {
		dataset = NormalizeDataset(dataset)
		if dataset.Id != "" {
			datasetByID[dataset.Id] = dataset
		}
	}
	for _, join := range joins {
		leftID := strings.TrimSpace(join.LeftDatasetId)
		rightID := strings.TrimSpace(join.RightDatasetId)
		leftField := strings.TrimSpace(join.LeftField)
		rightField := strings.TrimSpace(join.RightField)
		if leftID == "" || rightID == "" || leftField == "" || rightField == "" {
			return myerrors.NewValidationError("报表数据集关联配置不完整")
		}
		if leftID == rightID {
			return myerrors.NewValidationError("报表数据集关联不能指向同一数据集")
		}
		if _, ok := datasetByID[leftID]; !ok {
			return myerrors.NewValidationError("报表数据集关联左侧数据集不存在")
		}
		if _, ok := datasetByID[rightID]; !ok {
			return myerrors.NewValidationError("报表数据集关联右侧数据集不存在")
		}
		switch strings.ToLower(strings.TrimSpace(join.JoinType)) {
		case "", "left", "inner":
		default:
			return myerrors.NewValidationError("报表数据集关联类型不合法")
		}
	}
	return nil
}

func NormalizeDataset(dataset Dataset) Dataset {
	dataset.Id = strings.TrimSpace(dataset.Id)
	dataset.Name = strings.TrimSpace(dataset.Name)
	dataset.Type = NormalizeDatasetType(dataset.Type)
	dataset.SourceCode = strings.TrimSpace(dataset.SourceCode)
	dataset.SQL = strings.TrimSpace(dataset.SQL)
	return dataset
}

func NormalizeDatasetType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case SourceTypeTable, "":
		return SourceTypeTable
	case SourceTypeSQL:
		return SourceTypeSQL
	default:
		return ""
	}
}
