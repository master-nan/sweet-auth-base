/**
 * @Author: Nan
 * @Date: 2024/6/13 下午11:32
 */

package service

import (
	"backend/dto/request"
	"backend/enum"
	error2 "backend/internal/errors"
	platformmetadata "backend/internal/metadata"
	"backend/internal/security"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GeneralizationService struct {
	generalizationRepo repository.GeneralizationRepository
	permissionRepo     repository.GeneralizationPermissionRepository
	sf                 *utils.Snowflake
	permissionRuntime  *LowCodeDataPermissionRuntime
	metadataRuntime    platformmetadata.RuntimeReader
}

func NewGeneralizationServiceWithRuntimeAndDataPermission(
	generalizationRepo repository.GeneralizationRepository,
	sf *utils.Snowflake,
	metadataRuntime platformmetadata.RuntimeReader,
	permissionRuntime *LowCodeDataPermissionRuntime,
) *GeneralizationService {
	service := NewGeneralizationServiceWithDataPermission(generalizationRepo, sf, permissionRuntime)
	service.metadataRuntime = metadataRuntime
	return service
}

// ResolveRuntimeTable is the compatibility edge between stable platform
// metadata and the existing dynamic query engine. Controllers and runtime
// consumers do not load SysTable persistence models themselves.
func (gs *GeneralizationService) ResolveRuntimeTable(
	ctx context.Context,
	tableCode string,
) (model.SysTable, error) {
	if gs == nil || gs.metadataRuntime == nil {
		return model.SysTable{}, error2.WrapSystemError(fmt.Errorf("metadata runtime is not initialized"))
	}
	metadata, err := gs.metadataRuntime.GetTable(ctx, tableCode)
	if errors.Is(err, error2.ErrDataNotFound) {
		return model.SysTable{}, nil
	}
	if err != nil {
		return model.SysTable{}, err
	}
	return metadata.QueryModel(), nil
}

func NewGeneralizationService(generalizationRepo repository.GeneralizationRepository, sf *utils.Snowflake) *GeneralizationService {
	permissionRepo, _ := generalizationRepo.(repository.GeneralizationPermissionRepository)
	return &GeneralizationService{
		generalizationRepo: generalizationRepo,
		permissionRepo:     permissionRepo,
		sf:                 sf,
	}
}

func NewGeneralizationServiceWithDataPermission(
	generalizationRepo repository.GeneralizationRepository,
	sf *utils.Snowflake,
	permissionRuntime *LowCodeDataPermissionRuntime,
) *GeneralizationService {
	permissionRepo, _ := generalizationRepo.(repository.GeneralizationPermissionRepository)
	return &GeneralizationService{
		generalizationRepo: generalizationRepo,
		permissionRepo:     permissionRepo,
		sf:                 sf,
		permissionRuntime:  permissionRuntime,
	}
}

func (gs *GeneralizationService) Query(basic *request.Basic, table model.SysTable) (repository.GeneralizationListResult, error) {
	result, err := gs.generalizationRepo.Query(basic, table)
	if err != nil {
		return repository.GeneralizationListResult{}, err
	}
	return result, nil
}

func (gs *GeneralizationService) QueryWithDataPermission(
	ctx *gin.Context,
	basic *request.Basic,
	table model.SysTable,
	operation string,
) (repository.GeneralizationListResult, error) {
	resolution, err := gs.resolveLowCodePermission(ctx, table, operation)
	if err != nil {
		return repository.GeneralizationListResult{}, err
	}
	if gs.permissionRepo == nil {
		return repository.GeneralizationListResult{}, error2.ErrDataPermissionRuntimeFailed
	}
	return gs.permissionRepo.QueryWithPermission(basic, table, resolution.permission)
}

func (gs *GeneralizationService) QueryWithResolvedDataPermission(
	basic *request.Basic,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (repository.GeneralizationListResult, error) {
	if gs.permissionRepo == nil {
		return repository.GeneralizationListResult{}, error2.ErrDataPermissionRuntimeFailed
	}
	return gs.permissionRepo.QueryWithPermission(basic, table, permission)
}

func (gs *GeneralizationService) ResolveDataPermission(
	ctx *gin.Context,
	table model.SysTable,
	operation string,
) (repository.GeneralizationPermission, error) {
	resolution, err := gs.resolveLowCodePermission(ctx, table, operation)
	if err != nil {
		return repository.GeneralizationPermission{}, err
	}
	return resolution.permission, nil
}

func (gs *GeneralizationService) GetById(table model.SysTable, id int) (map[string]interface{}, error) {
	return gs.generalizationRepo.GetById(table, id)
}

func (gs *GeneralizationService) GetByIdWithDataPermission(
	ctx *gin.Context,
	table model.SysTable,
	id int,
	operation string,
) (map[string]interface{}, error) {
	resolution, err := gs.resolveLowCodePermission(ctx, table, operation)
	if err != nil {
		return nil, err
	}
	if gs.permissionRepo == nil {
		return nil, error2.ErrDataPermissionRuntimeFailed
	}
	return gs.permissionRepo.GetByIdWithPermission(table, id, resolution.permission)
}

// GetFieldById 获取指定行的指定字段值
func (gs *GeneralizationService) GetFieldById(tableCode string, id int, fieldName string) (interface{}, error) {
	return gs.generalizationRepo.GetFieldById(tableCode, id, fieldName)
}

func (gs *GeneralizationService) Create(ctx *gin.Context, table model.SysTable, data map[string]interface{}) error {
	if isProtectedTable(table.TableCode) {
		return error2.NewBadRequestError(fmt.Sprintf("表 %s 为受保护的系统表，不允许通过通用接口操作", table.TableCode))
	}
	filtered := filterDataByFields(table, data, true)
	applyDefaultValues(table, filtered)
	if err := validateDataByBindings(table, filtered, true); err != nil {
		return err
	}
	normalizeDataByFieldTypes(table, filtered)
	// 生成雪花ID
	if utils.HasTableField(table, "id") {
		id, err := gs.sf.GenerateUniqueID()
		if err != nil {
			return err
		}
		filtered["id"] = int(id)
	}
	// 填充审计字段
	now := model.Now()
	setIfFieldExists(table, filtered, "gmt_create", now)
	setIfFieldExists(table, filtered, "gmt_modify", now)
	user := ctx.MustGet("user").(model.SysUser)
	setIfFieldExists(table, filtered, "gmt_create_user", user.Id)
	setIfFieldExists(table, filtered, "gmt_modify_user", user.Id)
	return gs.generalizationRepo.Create(table, filtered)
}

func (gs *GeneralizationService) Update(ctx *gin.Context, table model.SysTable, id int, data map[string]interface{}) error {
	if isProtectedTable(table.TableCode) {
		return error2.NewBadRequestError(fmt.Sprintf("表 %s 为受保护的系统表，不允许通过通用接口操作", table.TableCode))
	}
	filtered := filterDataByFields(table, data, false)
	delete(filtered, "id")
	if err := validateDataByBindings(table, filtered, false); err != nil {
		return err
	}
	if err := gs.ensureWritableRowExists(table, id); err != nil {
		return err
	}
	normalizeDataByFieldTypes(table, filtered)
	// 填充审计字段
	setIfFieldExists(table, filtered, "gmt_modify", model.Now())
	user := ctx.MustGet("user").(model.SysUser)
	setIfFieldExists(table, filtered, "gmt_modify_user", user.Id)
	return gs.generalizationRepo.Update(table, id, filtered)
}

func (gs *GeneralizationService) UpdateWithDataPermission(
	ctx *gin.Context,
	table model.SysTable,
	id int,
	data map[string]interface{},
) error {
	if isProtectedTable(table.TableCode) {
		return error2.NewBadRequestError(fmt.Sprintf("表 %s 为受保护的系统表，不允许通过通用接口操作", table.TableCode))
	}
	resolution, err := gs.resolveLowCodePermission(
		ctx,
		table,
		model.DataPermissionOperationUpdate,
	)
	if err != nil {
		return err
	}
	if resolution.modifiesOwnership(table, data) {
		return error2.ErrDataPermissionOwnershipUpdateDenied
	}
	filtered := filterDataByFields(table, data, false)
	delete(filtered, "id")
	if err = validateDataByBindings(table, filtered, false); err != nil {
		return err
	}
	normalizeDataByFieldTypes(table, filtered)
	setIfFieldExists(table, filtered, "gmt_modify", model.Now())
	user := ctx.MustGet("user").(model.SysUser)
	setIfFieldExists(table, filtered, "gmt_modify_user", user.Id)
	if gs.permissionRepo == nil {
		return error2.ErrDataPermissionRuntimeFailed
	}
	updated, err := gs.permissionRepo.UpdateWithPermission(
		table,
		id,
		filtered,
		resolution.permission,
	)
	if err != nil {
		return err
	}
	if !updated {
		return permissionMissError(resolution.permission)
	}
	return nil
}

func (gs *GeneralizationService) Delete(ctx *gin.Context, table model.SysTable, id int) error {
	if isProtectedTable(table.TableCode) {
		return error2.NewBadRequestError(fmt.Sprintf("表 %s 为受保护的系统表，不允许通过通用接口操作", table.TableCode))
	}
	if err := gs.ensureWritableRowExists(table, id); err != nil {
		return err
	}
	if !utils.HasTableField(table, "gmt_delete") {
		// 表没有软删除字段，执行硬删除
		return gs.generalizationRepo.HardDelete(table, id)
	}
	deleteData := map[string]interface{}{
		"gmt_delete": model.Now(),
	}
	user := ctx.MustGet("user").(model.SysUser)
	setIfFieldExists(table, deleteData, "gmt_delete_user", user.Id)
	return gs.generalizationRepo.SoftDelete(table, id, deleteData)
}

func (gs *GeneralizationService) DeleteWithDataPermission(
	ctx *gin.Context,
	table model.SysTable,
	id int,
) error {
	if isProtectedTable(table.TableCode) {
		return error2.NewBadRequestError(fmt.Sprintf("表 %s 为受保护的系统表，不允许通过通用接口操作", table.TableCode))
	}
	resolution, err := gs.resolveLowCodePermission(
		ctx,
		table,
		model.DataPermissionOperationDelete,
	)
	if err != nil {
		return err
	}
	if !utils.HasTableField(table, "gmt_delete") {
		if gs.permissionRepo == nil {
			return error2.ErrDataPermissionRuntimeFailed
		}
		deleted, deleteErr := gs.permissionRepo.HardDeleteWithPermission(table, id, resolution.permission)
		if deleteErr != nil {
			return deleteErr
		}
		if !deleted {
			return permissionMissError(resolution.permission)
		}
		return nil
	}
	deleteData := map[string]interface{}{"gmt_delete": model.Now()}
	user := ctx.MustGet("user").(model.SysUser)
	setIfFieldExists(table, deleteData, "gmt_delete_user", user.Id)
	if gs.permissionRepo == nil {
		return error2.ErrDataPermissionRuntimeFailed
	}
	deleted, err := gs.permissionRepo.SoftDeleteWithPermission(
		table,
		id,
		deleteData,
		resolution.permission,
	)
	if err != nil {
		return err
	}
	if !deleted {
		return permissionMissError(resolution.permission)
	}
	return nil
}

func (gs *GeneralizationService) BatchDeleteWithDataPermission(
	ctx *gin.Context,
	table model.SysTable,
	ids []int,
) error {
	if isProtectedTable(table.TableCode) {
		return error2.NewBadRequestError(fmt.Sprintf("表 %s 为受保护的系统表，不允许通过通用接口操作", table.TableCode))
	}
	ids, err := normalizeGeneralizationIds(ids)
	if err != nil {
		return err
	}
	resolution, err := gs.resolveLowCodePermission(
		ctx,
		table,
		model.DataPermissionOperationDelete,
	)
	if err != nil {
		return err
	}
	if gs.permissionRepo == nil {
		return error2.ErrDataPermissionRuntimeFailed
	}
	db := gs.permissionRepo.DBWithContext(ctx.Request.Context())
	if !utils.HasTableField(table, "gmt_delete") {
		return RunInTransaction(ctx.Request.Context(), db, func(tx *gorm.DB) error {
			deleted, deleteErr := gs.permissionRepo.BatchHardDeleteWithPermission(tx, table, ids, resolution.permission)
			if deleteErr != nil {
				return deleteErr
			}
			if !deleted {
				return permissionMissError(resolution.permission)
			}
			return nil
		})
	}
	deleteData := map[string]interface{}{"gmt_delete": model.Now()}
	user := ctx.MustGet("user").(model.SysUser)
	setIfFieldExists(table, deleteData, "gmt_delete_user", user.Id)
	return RunInTransaction(ctx.Request.Context(), db, func(tx *gorm.DB) error {
		deleted, err := gs.permissionRepo.BatchSoftDeleteWithPermission(tx, table, ids, deleteData, resolution.permission)
		if err != nil {
			return err
		}
		if !deleted {
			return permissionMissError(resolution.permission)
		}
		return nil
	})
}

func normalizeGeneralizationIds(ids []int) ([]int, error) {
	if len(ids) == 0 {
		return nil, error2.ErrParamInvalid
	}
	normalized := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, error2.ErrParamInvalid
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, nil
}

func (gs *GeneralizationService) resolveLowCodePermission(
	ctx *gin.Context,
	table model.SysTable,
	operation string,
) (lowCodePermissionResolution, error) {
	if gs == nil || gs.permissionRuntime == nil {
		return lowCodePermissionResolution{}, error2.ErrDataPermissionRuntimeFailed
	}
	return gs.permissionRuntime.Resolve(ctx, table, operation)
}

func permissionMissError(repository.GeneralizationPermission) error {
	return error2.ErrPermissionDenied
}

func (gs *GeneralizationService) ensureWritableRowExists(table model.SysTable, id int) error {
	if id <= 0 {
		return error2.ErrDataNotFound
	}
	exists, err := gs.generalizationRepo.RowExists(table, id)
	if err != nil {
		return err
	}
	if !exists {
		return error2.ErrDataNotFound
	}
	return nil
}

// IsProtectedGeneralizationTable 判断表是否为禁止通过通用低代码写入 API 修改的平台核心数据。
func IsProtectedGeneralizationTable(tableCode string) bool {
	code := strings.ToLower(tableCode)
	protectedPrefixes := []string{"sys_", "casbin_"}
	for _, prefix := range protectedPrefixes {
		if strings.HasPrefix(code, prefix) {
			return true
		}
	}
	protectedTables := []string{"application", "access_log", "login_log", "sms_log", "sms_template"}
	for _, t := range protectedTables {
		if code == t {
			return true
		}
	}
	return false
}

// setIfFieldExists 如果表中存在该字段则设置值
func setIfFieldExists(table model.SysTable, data map[string]interface{}, fieldCode string, value interface{}) {
	if utils.HasTableField(table, fieldCode) {
		data[fieldCode] = value
	}
}

// isProtectedTable 检查是否为受保护的系统表，防止通过通用接口操作核心数据
func isProtectedTable(tableCode string) bool {
	return IsProtectedGeneralizationTable(tableCode)
}

func filterDataByFields(table model.SysTable, data map[string]interface{}, isCreate bool) map[string]interface{} {
	allowed := make(map[string]model.SysTableField)
	for _, field := range table.TableFields {
		if isManagedField(field.FieldCode) {
			continue
		}
		if isCreate && !field.IsInsertShow {
			continue
		}
		if !isCreate && !field.IsUpdateShow {
			continue
		}
		allowed[field.FieldCode] = field
	}
	filtered := make(map[string]interface{})
	for key, value := range data {
		if _, ok := allowed[key]; ok {
			filtered[key] = value
		}
	}
	return filtered
}

func isManagedField(fieldCode string) bool {
	return security.IsManagedMetadataField(fieldCode)
}

func applyDefaultValues(table model.SysTable, data map[string]interface{}) {
	for _, field := range table.TableFields {
		if field.DefaultValue == nil || *field.DefaultValue == "" || !field.IsInsertShow || isManagedField(field.FieldCode) {
			continue
		}
		if !isEmptyValue(data[field.FieldCode], fieldValueExists(data, field.FieldCode)) {
			continue
		}
		data[field.FieldCode] = parseDefaultValue(field, *field.DefaultValue)
	}
}

func fieldValueExists(data map[string]interface{}, fieldCode string) bool {
	_, exists := data[fieldCode]
	return exists
}

func parseDefaultValue(field model.SysTableField, raw string) interface{} {
	raw = strings.TrimSpace(raw)
	switch field.FieldType {
	case enum.BigIntFieldType, enum.IntFieldType, enum.TinyintFieldType:
		if val, err := parseInt(raw); err == nil {
			return val
		}
	case enum.FloatFieldType:
		if val, err := parseFloat(raw); err == nil {
			return val
		}
	case enum.BooleanFieldType:
		if val, err := strconv.ParseBool(strings.ToLower(raw)); err == nil {
			return val
		}
		if raw == "1" {
			return true
		}
		if raw == "0" {
			return false
		}
	}
	return raw
}

func normalizeDataByFieldTypes(table model.SysTable, data map[string]interface{}) {
	for _, field := range table.TableFields {
		value, exists := data[field.FieldCode]
		if !exists {
			continue
		}
		if isEmptyValue(value, true) {
			if field.IsNull && shouldNormalizeEmptyValueToNil(field) {
				data[field.FieldCode] = nil
			}
			continue
		}
		if normalized, ok := normalizeFieldValue(field, value); ok {
			data[field.FieldCode] = normalized
		}
	}
}

func normalizeFieldValue(field model.SysTableField, value interface{}) (interface{}, bool) {
	switch field.FieldType {
	case enum.BigIntFieldType:
		return toInt(value)
	case enum.IntFieldType, enum.TinyintFieldType:
		if val, ok := toInt(value); ok {
			return int(val), true
		}
	case enum.FloatFieldType:
		return toFloat(value)
	case enum.BooleanFieldType:
		return toBool(value)
	case enum.DateFieldType:
		return toDate(value)
	case enum.DatetimeFieldType:
		return toDateTime(value)
	case enum.TimeFieldType:
		return toTimeValue(value)
	}
	return value, false
}

func validateDataByBindings(table model.SysTable, data map[string]interface{}, isCreate bool) error {
	if len(table.TableFields) == 0 {
		return nil
	}
	fieldsMap := make(map[string]model.SysTableField, len(table.TableFields))
	for _, field := range table.TableFields {
		fieldsMap[field.FieldCode] = field
	}

	for _, field := range table.TableFields {
		value, exists := data[field.FieldCode]
		bindings := parseBindings(field.Binding)
		isRequired := !field.IsNull && field.IsInsertShow && !isManagedField(field.FieldCode)
		if isCreate && isRequired {
			if isEmptyValue(value, exists) {
				return error2.NewBadRequestError(fmt.Sprintf("%s不能为空", field.FieldName))
			}
		}
		if !exists {
			continue
		}
		if isEmptyValue(value, true) {
			if !field.IsNull && !isManagedField(field.FieldCode) {
				return error2.NewBadRequestError(fmt.Sprintf("%s不能为空", field.FieldName))
			}
			continue
		}

		if err := validateValueType(field, value); err != nil {
			return err
		}
		if err := validateValueByBindings(field, value, bindings); err != nil {
			return err
		}
	}

	// 针对只传递部分字段的更新：若传入字段不在元数据中，直接跳过
	for key := range data {
		if _, ok := fieldsMap[key]; !ok {
			continue
		}
	}
	return nil
}

func validateValueType(field model.SysTableField, value interface{}) error {
	switch field.FieldType {
	case enum.BigIntFieldType, enum.IntFieldType, enum.TinyintFieldType:
		if _, ok := toInt(value); !ok {
			return error2.NewBadRequestError(fmt.Sprintf("%s必须是整数", field.FieldName))
		}
	case enum.FloatFieldType:
		if _, ok := toFloat(value); !ok {
			return error2.NewBadRequestError(fmt.Sprintf("%s必须是数字", field.FieldName))
		}
	case enum.BooleanFieldType:
		if _, ok := toBool(value); !ok {
			return error2.NewBadRequestError(fmt.Sprintf("%s必须是布尔值", field.FieldName))
		}
	case enum.DateFieldType:
		if _, ok := toDate(value); !ok {
			return error2.NewBadRequestError(fmt.Sprintf("%s必须是日期", field.FieldName))
		}
	case enum.DatetimeFieldType:
		if _, ok := toDateTime(value); !ok {
			return error2.NewBadRequestError(fmt.Sprintf("%s必须是日期时间", field.FieldName))
		}
	case enum.TimeFieldType:
		if _, ok := toTimeValue(value); !ok {
			return error2.NewBadRequestError(fmt.Sprintf("%s必须是时间", field.FieldName))
		}
	}
	return nil
}

func validateValueByBindings(field model.SysTableField, value interface{}, bindings []string) error {
	if hasBinding(bindings, "min") {
		minVal := getBindingValue(bindings, "min")
		if minVal != "" {
			if err := validateMin(field, value, minVal); err != nil {
				return err
			}
		}
	}
	if hasBinding(bindings, "max") {
		maxVal := getBindingValue(bindings, "max")
		if maxVal != "" {
			if err := validateMax(field, value, maxVal); err != nil {
				return err
			}
		}
	}
	if hasBinding(bindings, "email") {
		if !matchRegex(`^[^\s@]+@[^\s@]+\.[^\s@]+$`, value) {
			return error2.NewBadRequestError(fmt.Sprintf("%s格式不正确", field.FieldName))
		}
	}
	if hasBinding(bindings, "url") {
		if !isValidURL(value) {
			return error2.NewBadRequestError(fmt.Sprintf("%s格式不正确", field.FieldName))
		}
	}
	if hasBinding(bindings, "phone") || hasBinding(bindings, "mobile") {
		if !matchRegex(`^1\d{10}$`, value) {
			return error2.NewBadRequestError(fmt.Sprintf("%s格式不正确", field.FieldName))
		}
	}
	if hasBinding(bindings, "regex") || hasBinding(bindings, "regexp") {
		pattern := getBindingValue(bindings, "regex")
		if pattern == "" {
			pattern = getBindingValue(bindings, "regexp")
		}
		if pattern != "" {
			if !matchRegex(pattern, value) {
				return error2.NewBadRequestError(fmt.Sprintf("%s格式不正确", field.FieldName))
			}
		}
	}
	return nil
}

func parseBindings(binding string) []string {
	return strings.FieldsFunc(binding, func(r rune) bool {
		return r == '|' || r == ','
	})
}

func hasBinding(bindings []string, name string) bool {
	for _, item := range bindings {
		item = strings.TrimSpace(item)
		if item == name || strings.HasPrefix(item, name+"=") {
			return true
		}
	}
	return false
}

func getBindingValue(bindings []string, name string) string {
	for _, item := range bindings {
		item = strings.TrimSpace(item)
		if strings.HasPrefix(item, name+"=") {
			return strings.TrimSpace(item[len(name)+1:])
		}
	}
	return ""
}

func isEmptyValue(value interface{}, exists bool) bool {
	if !exists {
		return true
	}
	if value == nil {
		return true
	}
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str) == ""
	}
	return false
}

func validateMin(field model.SysTableField, value interface{}, minStr string) error {
	if isNumberField(field) {
		minVal, err := parseFloat(minStr)
		if err == nil {
			val, ok := toFloat(value)
			if ok && val < minVal {
				return error2.NewBadRequestError(fmt.Sprintf("%s不能小于%v", field.FieldName, minVal))
			}
		}
		return nil
	}
	minVal, err := parseInt(minStr)
	if err == nil {
		val := fmt.Sprintf("%v", value)
		if len(val) < minVal {
			return error2.NewBadRequestError(fmt.Sprintf("%s长度不能小于%d", field.FieldName, minVal))
		}
	}
	return nil
}

func validateMax(field model.SysTableField, value interface{}, maxStr string) error {
	if isNumberField(field) {
		maxVal, err := parseFloat(maxStr)
		if err == nil {
			val, ok := toFloat(value)
			if ok && val > maxVal {
				return error2.NewBadRequestError(fmt.Sprintf("%s不能大于%v", field.FieldName, maxVal))
			}
		}
		return nil
	}
	maxVal, err := parseInt(maxStr)
	if err == nil {
		val := fmt.Sprintf("%v", value)
		if len(val) > maxVal {
			return error2.NewBadRequestError(fmt.Sprintf("%s长度不能超过%d", field.FieldName, maxVal))
		}
	}
	return nil
}

func isNumberField(field model.SysTableField) bool {
	return field.FieldType == enum.BigIntFieldType ||
		field.FieldType == enum.FloatFieldType ||
		field.FieldType == enum.TinyintFieldType ||
		field.FieldType == enum.IntFieldType
}

func toFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case float64:
		return v, isFiniteFloat(v)
	case float32:
		f := float64(v)
		return f, isFiniteFloat(f)
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, false
		}
		f, err := parseFloat(v)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

func toInt(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case int32:
		return int64(v), true
	case uint:
		if uint64(v) > uint64(math.MaxInt64) {
			return 0, false
		}
		return int64(v), true
	case uint64:
		if v > uint64(math.MaxInt64) {
			return 0, false
		}
		return int64(v), true
	case uint32:
		return int64(v), true
	case float64:
		if !isFiniteFloat(v) || math.Trunc(v) != v {
			return 0, false
		}
		return int64(v), true
	case float32:
		f := float64(v)
		if !isFiniteFloat(f) || math.Trunc(f) != f {
			return 0, false
		}
		return int64(f), true
	case string:
		raw := strings.TrimSpace(v)
		if raw == "" {
			return 0, false
		}
		i, err := strconv.ParseInt(raw, 10, 64)
		return i, err == nil
	}
	return 0, false
}

func toBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case int:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case int64:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case int32:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case float64:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case float32:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case string:
		raw := strings.ToLower(strings.TrimSpace(v))
		switch raw {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
	}
	return false, false
}

func shouldNormalizeEmptyValueToNil(field model.SysTableField) bool {
	switch field.FieldType {
	case enum.VarcharFieldType, enum.TextFieldType, enum.JsonFieldType:
		return false
	default:
		return true
	}
}

func toDate(value interface{}) (time.Time, bool) {
	if t, ok := value.(time.Time); ok {
		return t, true
	}
	raw, ok := value.(string)
	if !ok {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(raw), model.AppLocation())
	return t, err == nil
}

func toDateTime(value interface{}) (time.Time, bool) {
	if t, ok := value.(time.Time); ok {
		return t, true
	}
	raw, ok := value.(string)
	if !ok {
		return time.Time{}, false
	}
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.DateTime, "2006-01-02T15:04", "2006-01-02T15:04:05", time.RFC3339, time.RFC3339Nano} {
		if t, err := time.ParseInLocation(layout, raw, model.AppLocation()); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func toTimeValue(value interface{}) (string, bool) {
	if t, ok := value.(time.Time); ok {
		return t.Format(time.TimeOnly), true
	}
	raw, ok := value.(string)
	if !ok {
		return "", false
	}
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.TimeOnly, "15:04"} {
		if t, err := time.ParseInLocation(layout, raw, model.AppLocation()); err == nil {
			return t.Format(time.TimeOnly), true
		}
	}
	return "", false
}

func isFiniteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func parseFloat(val string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
	if err != nil {
		return 0, err
	}
	if !isFiniteFloat(f) {
		return 0, fmt.Errorf("not a finite number")
	}
	return f, nil
}

func parseInt(val string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(val))
}

func matchRegex(pattern string, value interface{}) bool {
	str := fmt.Sprintf("%v", value)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(str)
}

func isValidURL(value interface{}) bool {
	str := fmt.Sprintf("%v", value)
	if strings.TrimSpace(str) == "" {
		return true
	}
	_, err := url.ParseRequestURI(str)
	return err == nil
}
