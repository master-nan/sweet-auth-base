package response

type ReportPreviewRes struct {
	Columns  []ReportPreviewColumn    `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	Total    int                      `json:"total"`
	Meta     ReportPreviewMeta        `json:"meta"`
	Datasets []ReportPreviewDataset   `json:"datasets,omitempty"`
	Joins    []ReportPreviewJoin      `json:"joins,omitempty"`
}

type ReportDataSourceRes struct {
	Id          int                   `json:"id"`
	Name        string                `json:"name"`
	Code        string                `json:"code"`
	Type        string                `json:"type"`
	Description string                `json:"description"`
	Fields      []ReportPreviewColumn `json:"fields"`
}

type ReportPreviewColumn struct {
	Name  string `json:"name"`
	Field string `json:"field"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type ReportPreviewMeta struct {
	ReportId    int    `json:"report_id"`
	ReportCode  string `json:"report_code"`
	SourceCode  string `json:"source_code"`
	DatasetId   string `json:"dataset_id,omitempty"`
	DatasetType string `json:"dataset_type,omitempty"`
	AppliedMenu int    `json:"applied_menu_id"`
}

type ReportPreviewDataset struct {
	Id         string                `json:"id"`
	Name       string                `json:"name"`
	Type       string                `json:"type"`
	SourceCode string                `json:"source_code,omitempty"`
	Primary    bool                  `json:"primary"`
	Fields     []ReportPreviewColumn `json:"fields,omitempty"`
}

type ReportPreviewJoin struct {
	Id             string `json:"id"`
	LeftDatasetId  string `json:"left_dataset_id"`
	LeftField      string `json:"left_field"`
	RightDatasetId string `json:"right_dataset_id"`
	RightField     string `json:"right_field"`
	JoinType       string `json:"join_type"`
}
