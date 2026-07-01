package response

type ReportPreviewRes struct {
	Columns []ReportPreviewColumn    `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Total   int                      `json:"total"`
	Meta    ReportPreviewMeta        `json:"meta"`
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
	AppliedMenu int    `json:"applied_menu_id"`
}
