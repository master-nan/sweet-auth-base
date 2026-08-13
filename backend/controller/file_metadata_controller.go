package controller

import (
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/middleware"
	"backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FileMetadataController struct {
	metadata   *service.FileMetadataService
	access     *service.FileAccessService
	authorizer *FileBusinessAccessAdapter
}

func NewFileMetadataController(metadata *service.FileMetadataService, access *service.FileAccessService, authorizer *FileBusinessAccessAdapter) *FileMetadataController {
	return &FileMetadataController{metadata: metadata, access: access, authorizer: authorizer}
}

// GetFileById godoc
// @Summary 获取文件详情
// @Tags 文件
// @Produce json
// @Router /admin/file/{id} [get]
func (c *FileMetadataController) GetFileById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	resource, err := c.metadata.FindByID(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err = c.authorizer.Authorize(ctx, resource, enum.ButtonActionDetail, true); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(c.metadata.Detail(resource))
}

// DeleteFileById godoc
// @Summary 删除文件
// @Tags 文件
// @Produce json
// @Router /admin/file/{id} [delete]
func (c *FileMetadataController) DeleteFileById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "delete", ResourceType: "file", ResourceCode: "file", ResourceID: strconv.Itoa(id)})
	resource, err := c.metadata.FindForDelete(ctx.Request.Context(), id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err = c.authorizer.Authorize(ctx, resource, enum.ButtonActionDelete, false); err != nil {
		_ = ctx.Error(err)
		return
	}
	if err = c.metadata.Delete(ctx.Request.Context(), resource); err != nil {
		_ = ctx.Error(err)
	}
}
