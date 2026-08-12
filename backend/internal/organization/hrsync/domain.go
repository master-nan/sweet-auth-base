package hrsync

import (
	"backend/internal/integration"
	"context"
	"errors"
	"strings"
	"time"
)

const OrganizationHRSourceSystemCode = "hr_source"

const (
	ConsumerCodeLegalEntity          = "org.hr.legal_entity"
	ConsumerCodeManagementCompany    = "org.hr.management_company"
	ConsumerCodeManagementDepartment = "org.hr.management_department"
	ConsumerCodeLegalDepartment      = "org.hr.legal_department"
	ConsumerVersionV1                = 1
)

const OrganizationHRSyncChunkSize = 500

var ErrSourceContractInvalid = errors.New("org_sync_source_contract_invalid")

// SourceContract 只由已有权威证据的部署装配；默认生产装配不会构造它。
type SourceContract struct {
	sourceSystemCode string
	sourceLocation   *time.Location
}

func NewExplicitSourceContract(sourceSystemCode string, sourceLocation *time.Location) (SourceContract, error) {
	sourceSystemCode = strings.TrimSpace(sourceSystemCode)
	if sourceSystemCode != OrganizationHRSourceSystemCode || sourceLocation == nil {
		return SourceContract{}, ErrSourceContractInvalid
	}
	return SourceContract{sourceSystemCode: sourceSystemCode, sourceLocation: sourceLocation}, nil
}

func (c SourceContract) valid() bool {
	return c.sourceSystemCode != "" && c.sourceLocation != nil
}

func (c SourceContract) normalizer() Normalizer {
	return Normalizer{SourceSystemCode: c.sourceSystemCode, SourceLocation: c.sourceLocation}
}

type BusinessSyncContext struct {
	ExecutionNo string
	SyncBatchNo string
	TaskCode    string
	TaskVersion int
	SliceNo     int
}

func NewBusinessSyncContext(request integration.SyncConsumptionRequest) BusinessSyncContext {
	return BusinessSyncContext{
		ExecutionNo: request.ExecutionNo(), SyncBatchNo: request.SyncBatchNo(), TaskCode: request.TaskCode(),
		TaskVersion: request.TaskVersion(), SliceNo: request.SliceNo(),
	}
}

type SourceIssue struct {
	ObjectKind      ObjectKind
	SourceIDSummary string
	ReasonCode      ReasonCode
	Action          string
	DependencyType  string
	DependencyKey   string
}

type BusinessSyncSummary struct {
	SuccessCount int
	FailedCount  int
	ReasonCode   ReasonCode
	BatchNo      string
}

func (s BusinessSyncSummary) Success() bool {
	return s.FailedCount == 0 && s.ReasonCode == ""
}

// OrganizationSyncDomain 是 Source Adapter 可见的唯一业务端口。
type OrganizationSyncDomain interface {
	SynchronizeLegalEntities(context.Context, BusinessSyncContext, []LegalEntitySyncInput, []SourceIssue) (BusinessSyncSummary, error)
	SynchronizeOrgUnits(context.Context, BusinessSyncContext, ObjectKind, string, []OrgUnitSyncInput, []SourceIssue) (BusinessSyncSummary, error)
}
