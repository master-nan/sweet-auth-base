package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"backend/dto/response"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type lowCodeResourceLookup func(*gin.Context, int) ([]model.DataResource, error)
type lowCodeOwnershipLookup func(*gin.Context, int) ([]model.DataOwnershipField, error)
type lowCodeSubjectBuilder func(*gin.Context, int) (datapermission.SubjectContext, error)
type lowCodeResolver func(*gin.Context, datapermission.ResolverInput) (datapermission.DataScopeResult, error)
type lowCodeMetadataAdapter func(context.Context, datapermission.AdapterInput) (datapermission.AdapterExecution, error)

type lowCodePermissionResolution struct {
	permission        repository.GeneralizationPermission
	ownershipFieldIds map[int]struct{}
}

func (resolution lowCodePermissionResolution) modifiesOwnership(
	table model.SysTable,
	data map[string]interface{},
) bool {
	if len(resolution.ownershipFieldIds) == 0 || len(data) == 0 {
		return false
	}
	for _, field := range table.TableFields {
		if _, protected := resolution.ownershipFieldIds[field.Id]; !protected {
			continue
		}
		if _, submitted := data[field.FieldCode]; submitted {
			return true
		}
	}
	return false
}

// LowCodeDataPermissionRuntime 是通用元数据读写使用的唯一数据权限边界。
type LowCodeDataPermissionRuntime struct {
	findResources  lowCodeResourceLookup
	findOwnerships lowCodeOwnershipLookup
	buildSubject   lowCodeSubjectBuilder
	resolveScope   lowCodeResolver
	applyMetadata  lowCodeMetadataAdapter
}

func NewLowCodeDataPermissionRuntime(
	resourceRepo repository.DataResourceRepository,
	ownershipRepo repository.DataOwnershipFieldRepository,
	subjectBuilder *SubjectContextBuilder,
	resolver datapermission.Resolver,
	metadataAdapter *datapermission.MetadataFieldAdapter,
) *LowCodeDataPermissionRuntime {
	return newLowCodeDataPermissionRuntime(
		func(ctx *gin.Context, tableId int) ([]model.DataResource, error) {
			return resourceRepo.ListByTableId(ctx, tableId)
		},
		func(ctx *gin.Context, resourceId int) ([]model.DataOwnershipField, error) {
			return ownershipRepo.ListByResource(ctx, resourceId)
		},
		subjectBuilder.Build,
		resolver.Resolve,
		metadataAdapter.Apply,
	)
}

func newLowCodeDataPermissionRuntime(
	findResources lowCodeResourceLookup,
	findOwnerships lowCodeOwnershipLookup,
	buildSubject lowCodeSubjectBuilder,
	resolveScope lowCodeResolver,
	applyMetadata lowCodeMetadataAdapter,
) *LowCodeDataPermissionRuntime {
	return &LowCodeDataPermissionRuntime{
		findResources:  findResources,
		findOwnerships: findOwnerships,
		buildSubject:   buildSubject,
		resolveScope:   resolveScope,
		applyMetadata:  applyMetadata,
	}
}

func (runtime *LowCodeDataPermissionRuntime) Resolve(
	ctx *gin.Context,
	table model.SysTable,
	operation string,
) (lowCodePermissionResolution, error) {
	startedAt := time.Now()
	if err := runtime.validate(); err != nil {
		return lowCodePermissionResolution{}, err
	}
	operation = strings.ToLower(strings.TrimSpace(operation))
	if table.Id <= 0 || operation == "" {
		return lowCodePermissionResolution{}, myerrors.ErrDataPermissionRuntimeFailed
	}

	resources, err := runtime.findResources(ctx, table.Id)
	if err != nil {
		return lowCodePermissionResolution{}, mapLowCodeRuntimeDependencyError(err)
	}
	if len(resources) == 0 {
		return runtime.resolveNotApplicable(ctx, table, operation, "", "")
	}
	if len(resources) != 1 {
		return lowCodePermissionResolution{}, myerrors.ErrDataPermissionRuntimeRouteConflict
	}
	resource := resources[0]
	if resource.Id <= 0 || resource.TableId == nil || *resource.TableId != table.Id ||
		resource.ResourceType != model.DataResourceTypeLowCodeTable {
		return lowCodePermissionResolution{}, myerrors.ErrDataPermissionRuntimeRouteConflict
	}
	if !resource.State || !resource.PermissionEnabled {
		return runtime.resolveNotApplicable(
			ctx,
			table,
			operation,
			resource.ResourceCode,
			resource.AdapterCode,
		)
	}

	user, err := trustedRuntimeUser(ctx)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	subject, err := runtime.buildSubject(ctx, user.Id)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	resolverInput, err := datapermission.NewResolverInput(subject, resource.ResourceCode, operation)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	result, err := runtime.resolveScope(ctx, resolverInput)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	if result.Decision() == datapermission.DataScopeDecisionNotApplicable {
		return lowCodePermissionResolution{}, myerrors.ErrDataPermissionRuntimeRouteConflict
	}
	ownerships, err := runtime.findOwnerships(ctx, resource.Id)
	if err != nil {
		return lowCodePermissionResolution{}, mapLowCodeRuntimeDependencyError(err)
	}
	definitions, fieldIds, err := lowCodeOwnershipDefinitions(ownerships)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	resourceContext, err := datapermission.NewAdapterResourceContext(
		datapermission.AdapterResourceContextInput{
			ResourceCode: resource.ResourceCode,
			Operation:    operation,
			AdapterCode:  resource.AdapterCode,
			TableId:      table.Id,
		},
	)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	adapterInput, err := datapermission.NewAdapterInput(resourceContext, result, definitions)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	execution, err := runtime.applyMetadata(runtimeContext(ctx), adapterInput)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}

	zapLowCodePermissionDecision(ctx, result, execution.Mode(), startedAt)
	return lowCodePermissionResolution{
		permission: repository.GeneralizationPermission{
			AdapterExecution: &execution,
		},
		ownershipFieldIds: fieldIds,
	}, nil
}

func (runtime *LowCodeDataPermissionRuntime) resolveNotApplicable(
	ctx *gin.Context,
	table model.SysTable,
	operation string,
	resourceCode string,
	adapterCode string,
) (lowCodePermissionResolution, error) {
	if resourceCode == "" {
		resourceCode = "unconfigured." + strconv.Itoa(table.Id)
	}
	if adapterCode == "" {
		adapterCode = "metadata"
	}
	result, err := datapermission.NewNotApplicableResult(resourceCode, operation)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	resourceContext, err := datapermission.NewAdapterResourceContext(
		datapermission.AdapterResourceContextInput{
			ResourceCode: resourceCode,
			Operation:    operation,
			AdapterCode:  adapterCode,
			TableId:      table.Id,
		},
	)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	adapterInput, err := datapermission.NewAdapterInput(resourceContext, result, nil)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	execution, err := runtime.applyMetadata(runtimeContext(ctx), adapterInput)
	if err != nil {
		return lowCodePermissionResolution{}, err
	}
	zapLowCodePermissionDecision(ctx, result, execution.Mode(), time.Now())
	return lowCodePermissionResolution{
		permission: repository.GeneralizationPermission{
			AdapterExecution: &execution,
		},
	}, nil
}

func (runtime *LowCodeDataPermissionRuntime) validate() error {
	if runtime == nil || runtime.findResources == nil || runtime.findOwnerships == nil ||
		runtime.buildSubject == nil || runtime.resolveScope == nil || runtime.applyMetadata == nil {
		return myerrors.ErrDataPermissionRuntimeFailed
	}
	return nil
}

func lowCodeOwnershipDefinitions(
	ownerships []model.DataOwnershipField,
) ([]datapermission.AdapterOwnershipDefinition, map[int]struct{}, error) {
	definitions := make([]datapermission.AdapterOwnershipDefinition, 0, len(ownerships))
	fieldIds := make(map[int]struct{})
	for _, ownership := range ownerships {
		if !ownership.State {
			continue
		}
		tableFieldId := 0
		if ownership.TableFieldId != nil {
			tableFieldId = *ownership.TableFieldId
		}
		adapterFieldCode := ""
		if ownership.AdapterFieldCode != nil {
			adapterFieldCode = *ownership.AdapterFieldCode
		}
		definition, err := datapermission.NewAdapterOwnershipDefinition(
			datapermission.AdapterOwnershipDefinitionInput{
				OwnershipCode:    ownership.OwnershipCode,
				DimensionId:      ownership.DimensionId,
				BindingType:      datapermission.AdapterBindingType(ownership.BindingType),
				TableFieldId:     tableFieldId,
				AdapterFieldCode: adapterFieldCode,
				ValueType:        datapermission.DataScopeValueType(ownership.ValueType),
			},
		)
		if err != nil {
			return nil, nil, err
		}
		definitions = append(definitions, definition)
		if definition.BindingType() == datapermission.AdapterBindingTypeMetadataField {
			fieldIds[definition.TableFieldId()] = struct{}{}
		}
	}
	return definitions, fieldIds, nil
}

func trustedRuntimeUser(ctx *gin.Context) (model.SysUser, error) {
	if ctx == nil {
		return model.SysUser{}, myerrors.ErrDataPermissionSubjectContextInvalid
	}
	value, exists := ctx.Get("user")
	user, ok := value.(model.SysUser)
	if !exists || !ok || user.Id <= 0 {
		return model.SysUser{}, myerrors.ErrDataPermissionSubjectContextInvalid
	}
	trustedId, exists := ctx.Get("id")
	id, ok := trustedId.(int)
	if !exists || !ok || id != user.Id {
		return model.SysUser{}, myerrors.ErrDataPermissionSubjectContextInvalid
	}
	return user, nil
}

func runtimeContext(ctx *gin.Context) context.Context {
	if ctx != nil && ctx.Request != nil {
		return ctx.Request.Context()
	}
	return context.Background()
}

func mapLowCodeRuntimeDependencyError(err error) error {
	var adminError *response.AdminError
	if errors.As(err, &adminError) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataPermissionRuntimeRouteConflict
	}
	return myerrors.WrapDatabaseError(err)
}

func zapLowCodePermissionDecision(
	ctx *gin.Context,
	result datapermission.DataScopeResult,
	mode datapermission.AdapterExecutionMode,
	startedAt time.Time,
) {
	fields := []zap.Field{
		zap.String("resource_code", result.ResourceCode()),
		zap.String("operation", result.Operation()),
		zap.String("decision", string(result.Decision())),
		zap.String("execution_mode", string(mode)),
		zap.Duration("elapsed", time.Since(startedAt)),
	}
	if summary, ok := datapermission.ResolverSummaryFromContext(ctx); ok {
		fields = append(fields,
			zap.Int("checked_grant_count", summary.CheckedGrantCount()),
			zap.Int("checked_policy_count", summary.CheckedPolicyCount()),
		)
	}
	zap.L().Debug("low-code data permission resolved", fields...)
}
