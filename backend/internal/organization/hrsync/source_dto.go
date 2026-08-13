package hrsync

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

var ErrSourceEnumInvalid = errors.New("org_sync_source_enum_invalid")

type SourceEnableStatus int

const (
	SourceEnableUnknown SourceEnableStatus = iota
	SourceEnableDisabled
	SourceEnableEnabled
)

func (value *SourceEnableStatus) UnmarshalJSON(raw []byte) error {
	text := strings.TrimSpace(string(bytes.TrimSpace(raw)))
	if unquoted, err := strconv.Unquote(text); err == nil {
		text = strings.TrimSpace(unquoted)
	}
	switch strings.ToLower(text) {
	case "1", "true":
		*value = SourceEnableEnabled
	case "0", "2", "false":
		*value = SourceEnableDisabled
	default:
		return ErrSourceEnumInvalid
	}
	return nil
}

// 源 DTO 只描述 HR V1 映射需要的字段。未知 JSON 字段由 encoding/json 忽略。
type HRCompanySourceDTO struct {
	SourceID       string             `json:"zjkid_ignore"`
	SourceRecordID string             `json:"id"`
	SourceCode     string             `json:"pk_corp"`
	Name           string             `json:"name"`
	ShortName      string             `json:"shortname"`
	ParentSourceID string             `json:"fatherpkzjkid_ignore"`
	Enabled        SourceEnableStatus `json:"isenable"`
	ChangeTime     string             `json:"changeTime"`
	Level          int                `json:"level"`
}

type HRDepartmentSourceDTO struct {
	SourceID            string             `json:"zjkid_ignore"`
	SourceRecordID      string             `json:"id"`
	SourceCode          string             `json:"code"`
	Name                string             `json:"name"`
	ParentSourceID      string             `json:"pk_fathedeptzjkid_ignore"`
	LegalEntitySourceID string             `json:"orgidzjkid_ignore"`
	Enabled             SourceEnableStatus `json:"isenable"`
	ChangeTime          string             `json:"changeTime"`
	Level               int                `json:"ilevel"`
	Sort                string             `json:"disorder"`
}

type HRPositionSourceDTO struct {
	SourceID        string             `json:"postidzjkid_ignore"`
	SourceCode      string             `json:"postCode"`
	Name            string             `json:"postname"`
	OrgUnitSourceID string             `json:"deptidzjkid_ignore"`
	JobLevel        string             `json:"posLevel"`
	Enabled         SourceEnableStatus `json:"isenable"`
	ChangeTime      string             `json:"changeTime"`
}

type HREmployeeSourceDTO struct {
	SourceID            string             `json:"psnidzjkid_ignore"`
	EmployeeNo          string             `json:"jhcode"`
	Name                string             `json:"name"`
	Mobile              string             `json:"mobile"`
	Email               string             `json:"email"`
	Enabled             SourceEnableStatus `json:"isenable"`
	ChangeTime          string             `json:"changeTime"`
	LegalEntitySourceID string             `json:"corporationCompanyId"`
	ManagementUnitID    string             `json:"deptidzjkid_ignore"`
	PositionSourceID    string             `json:"postidzjkid_ignore"`
	EmbeddedAssignments string             `json:"sendpost"`
}

type HRResignedEmployeeSourceDTO struct {
	SourceID   string `json:"psnidzjkid_ignore"`
	EmployeeNo string `json:"jhcode"`
	Name       string `json:"psnname"`
	ChangeTime string `json:"changeTime"`
	ResignedAt string `json:"lzdate"`
}

// HRAssignmentSourceDTO 仅冻结结构边界；P0 关闭前不提供落库 Normalizer。
type HRAssignmentSourceDTO struct {
	SourceID         string `json:"ID"`
	LegalEntityID    string `json:"公司ID"`
	StructureType    int    `json:"兼职架构"`
	AssignmentTypeID string `json:"兼职类型"`
	OrgUnitID        string `json:"部门主键ID"`
	PositionID       string `json:"岗位ID"`
	StartedAt        string `json:"开始时间"`
	EndedAt          string `json:"结束时间"`
	OnPost           string `json:"在岗"`
	Ended            string `json:"结束兼职"`
}
