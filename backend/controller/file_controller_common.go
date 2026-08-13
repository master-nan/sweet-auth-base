package controller

import (
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type fileBusinessContext struct {
	TableCode string
	RecordID  int
	Action    enum.SysMenuButtonEventAction
}

// FileBusinessAccessAdapter is the HTTP-only bridge to the legacy low-code
// data-permission API. Gin does not cross into File services or repositories.
type FileBusinessAccessAdapter struct {
	files          *service.FileAccessService
	generalization *service.GeneralizationService
}

func NewFileBusinessAccessAdapter(files *service.FileAccessService, generalization *service.GeneralizationService) *FileBusinessAccessAdapter {
	return &FileBusinessAccessAdapter{files: files, generalization: generalization}
}

func (a *FileBusinessAccessAdapter) Authorize(ctx *gin.Context, resource service.FileAccessResource, fallback enum.SysMenuButtonEventAction, allowOverride bool) error {
	business, found, err := parseFileBusinessContext(ctx, fallback, allowOverride)
	if err != nil {
		return err
	}
	if !found {
		return a.files.AuthorizeActor(fileAccessActor(ctx), resource)
	}
	if a.generalization == nil {
		return myerrors.ErrPermissionDenied
	}
	table, err := a.generalization.ResolveRuntimeTable(ctx.Request.Context(), business.TableCode)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrParamInvalid
	}
	record, err := a.generalization.GetByIdWithDataPermission(ctx, table, business.RecordID, fileDataPermissionOperation(business.Action))
	if err != nil {
		return err
	}
	ref := resource.Reference()
	if !recordContainsFileReference(table, record, ref.UUID) && !recordContainsFileReference(table, record, strconv.Itoa(ref.ID)) {
		return myerrors.ErrPermissionDenied
	}
	return nil
}

func fileAccessActor(ctx *gin.Context) service.FileAccessActor {
	user := ctx.MustGet("user").(model.SysUser)
	return service.FileAccessActor{UserID: user.Id, IsSuperAdmin: utils.IsSuperAdmin(user)}
}

func parseFileBusinessContext(ctx *gin.Context, fallback enum.SysMenuButtonEventAction, allowOverride bool) (fileBusinessContext, bool, error) {
	tableCode := strings.TrimSpace(ctx.Query("table_code"))
	recordIDRaw := strings.TrimSpace(firstNonEmpty(ctx.Query("record_id"), ctx.Query("row_id"), ctx.Query("id")))
	if tableCode == "" && recordIDRaw == "" {
		return fileBusinessContext{}, false, nil
	}
	if tableCode == "" || recordIDRaw == "" {
		return fileBusinessContext{}, true, myerrors.ErrParamInvalid
	}
	recordID, err := strconv.Atoi(recordIDRaw)
	if err != nil || recordID <= 0 {
		return fileBusinessContext{}, true, myerrors.ErrParamInvalid
	}
	action := fallback
	if raw := strings.TrimSpace(ctx.Query("action")); raw != "" {
		normalized, ok := enum.NormalizeSysMenuButtonEventAction(raw)
		if !ok || (!allowOverride && normalized != fallback) {
			return fileBusinessContext{}, true, myerrors.ErrParamInvalid
		}
		action = normalized
	}
	return fileBusinessContext{TableCode: tableCode, RecordID: recordID, Action: action}, true, nil
}

func fileDataPermissionOperation(action enum.SysMenuButtonEventAction) string {
	switch action {
	case enum.ButtonActionUpdate:
		return model.DataPermissionOperationUpdate
	case enum.ButtonActionDelete:
		return model.DataPermissionOperationDelete
	case enum.ButtonActionExport:
		return model.DataPermissionOperationExport
	default:
		return model.DataPermissionOperationDetail
	}
}

func recordContainsFileReference(table model.SysTable, record map[string]interface{}, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" || len(record) == 0 {
		return false
	}
	for _, field := range table.TableFields {
		if field.InputType != enum.FilePickerInputType && field.InputType != enum.RichTextInputType {
			continue
		}
		value, ok := record[field.FieldCode]
		if !ok || value == nil {
			continue
		}
		haystack := fmt.Sprintf("%v", value)
		if raw, ok := value.([]byte); ok {
			haystack = string(raw)
		}
		if _, err := strconv.Atoi(needle); err == nil {
			pattern := `(^|[^0-9])` + regexp.QuoteMeta(needle) + `([^0-9]|$)`
			if regexp.MustCompile(pattern).MatchString(haystack) {
				return true
			}
			continue
		}
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
