package controller

import (
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/middleware"
	"backend/service"
	"io"

	"github.com/gin-gonic/gin"
)

type FileAccessController struct {
	service    *service.FileAccessService
	authorizer *FileBusinessAccessAdapter
}

func NewFileAccessController(access *service.FileAccessService, authorizer *FileBusinessAccessAdapter) *FileAccessController {
	return &FileAccessController{service: access, authorizer: authorizer}
}

// Download godoc
// @Summary 下载文件
// @Tags 文件
// @Produce octet-stream
// @Router /admin/file/download/{uuid} [get]
func (c *FileAccessController) Download(ctx *gin.Context) {
	c.authorizedStream(ctx, true)
}

// Preview godoc
// @Summary 预览文件
// @Tags 文件
// @Produce octet-stream
// @Router /admin/file/preview/{uuid} [get]
func (c *FileAccessController) Preview(ctx *gin.Context) {
	c.authorizedStream(ctx, false)
}

// PublicPreview godoc
// @Summary 公开预览文件
// @Tags 文件
// @Produce octet-stream
// @Router /files/{uuid} [get]
func (c *FileAccessController) PublicPreview(ctx *gin.Context) {
	if !c.service.PublicPreviewEnabled() {
		_ = ctx.Error(myerrors.ErrFileNotFound)
		return
	}
	resource, err := c.service.FindByUUID(ctx.Request.Context(), ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrFileNotFound)
		return
	}
	c.stream(ctx, resource, false)
}

// GetFilePreviewAccessURL godoc
// @Summary 获取文件签名预览地址
// @Tags 文件
// @Produce json
// @Router /admin/file/preview-url/{uuid} [get]
func (c *FileAccessController) GetFilePreviewAccessURL(ctx *gin.Context) {
	c.issueSignedURL(ctx, service.FileAccessPurposePreview)
}

// GetFileDownloadAccessURL godoc
// @Summary 获取文件签名下载地址
// @Tags 文件
// @Produce json
// @Router /admin/file/download-url/{uuid} [get]
func (c *FileAccessController) GetFileDownloadAccessURL(ctx *gin.Context) {
	c.issueSignedURL(ctx, service.FileAccessPurposeDownload)
}

// SignedPreview godoc
// @Summary 通过签名预览文件
// @Tags 文件
// @Produce octet-stream
// @Router /files/access/preview/{uuid} [get]
func (c *FileAccessController) SignedPreview(ctx *gin.Context) {
	c.signedStream(ctx, service.FileAccessPurposePreview, false)
}

// SignedDownload godoc
// @Summary 通过签名下载文件
// @Tags 文件
// @Produce octet-stream
// @Router /files/access/download/{uuid} [get]
func (c *FileAccessController) SignedDownload(ctx *gin.Context) {
	c.signedStream(ctx, service.FileAccessPurposeDownload, true)
}

func (c *FileAccessController) authorizedStream(ctx *gin.Context, attachment bool) {
	resource, err := c.service.FindByUUID(ctx.Request.Context(), ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err = c.authorizer.Authorize(ctx, resource, enum.ButtonActionDetail, true); err != nil {
		_ = ctx.Error(err)
		return
	}
	c.stream(ctx, resource, attachment)
}

func (c *FileAccessController) issueSignedURL(ctx *gin.Context, purpose service.FileAccessPurpose) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "issue_" + string(purpose), ResourceType: "file", ResourceCode: "file", ResourceID: ctx.Param("uuid")})
	resource, err := c.service.FindByUUID(ctx.Request.Context(), ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err = c.authorizer.Authorize(ctx, resource, enum.ButtonActionDetail, true); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := c.service.IssueSignedURL(resource, purpose, ctx.Query("ttl"))
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

func (c *FileAccessController) signedStream(ctx *gin.Context, purpose service.FileAccessPurpose, attachment bool) {
	resource, err := c.service.ResolveSigned(ctx.Request.Context(), ctx.Param("uuid"), ctx.Query("token"), purpose)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	c.stream(ctx, resource, attachment)
}

func (c *FileAccessController) stream(ctx *gin.Context, resource service.FileAccessResource, attachment bool) {
	stream, err := c.service.Open(resource, attachment)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	defer stream.Reader.Close()
	for key, value := range stream.Headers {
		ctx.Header(key, value)
	}
	_, _ = io.Copy(ctx.Writer, stream.Reader)
}
