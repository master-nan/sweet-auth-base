package controller

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/middleware"
	"backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type FileUploadController struct {
	service     *service.FileUploadService
	translators map[string]ut.Translator
}

func NewFileUploadController(upload *service.FileUploadService, translators map[string]ut.Translator) *FileUploadController {
	return &FileUploadController{service: upload, translators: translators}
}

// Upload godoc
// @Summary 上传文件
// @Tags 文件
// @Accept multipart/form-data
// @Produce json
// @Router /admin/file/upload [post]
func (c *FileUploadController) Upload(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "upload", ResourceType: "file", ResourceCode: "file"})
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		_ = ctx.Error(myerrors.NewValidationError("请选择要上传的文件"))
		return
	}
	result, err := c.service.UploadResponse(ctx.Request.Context(), fileAccessActor(ctx), fileHeader)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "upload", ResourceType: "file", ResourceCode: "file", ResourceID: strconv.Itoa(result.Id)})
	resp.SetData(result)
}

// InitChunkUpload godoc
// @Summary 初始化分片上传
// @Tags 文件
// @Accept json
// @Produce json
// @Router /admin/file/upload/init [post]
func (c *FileUploadController) InitChunkUpload(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "init_chunk_upload", ResourceType: "file_upload", ResourceCode: "file_upload"})
	var req request.ChunkUploadInitReq
	if err := utils.ValidatorBody(ctx, &req, c.translators["zh"]); err != nil {
		_ = ctx.Error(err)
		return
	}
	result, err := c.service.InitChunkUpload(ctx.Request.Context(), fileAccessActor(ctx), req)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "init_chunk_upload", ResourceType: "file_upload", ResourceCode: "file_upload", ResourceID: result.UploadId})
	resp.SetData(result)
}

// UploadChunk godoc
// @Summary 上传文件分片
// @Tags 文件
// @Accept multipart/form-data
// @Produce json
// @Router /admin/file/upload/chunk [post]
func (c *FileUploadController) UploadChunk(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	uploadID := ctx.PostForm("upload_id")
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "upload_chunk", ResourceType: "file_upload", ResourceCode: "file_upload", ResourceID: uploadID})
	chunkIndex, err := strconv.Atoi(ctx.PostForm("chunk_index"))
	if uploadID == "" || err != nil || chunkIndex < 0 {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		_ = ctx.Error(myerrors.NewValidationError("请选择要上传的分片文件"))
		return
	}
	if err = c.service.UploadChunk(ctx.Request.Context(), fileAccessActor(ctx), uploadID, chunkIndex, fileHeader); err != nil {
		_ = ctx.Error(err)
		return
	}
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "upload_chunk", ResourceType: "file_upload", ResourceCode: "file_upload", ResourceID: uploadID})
}

// MergeChunks godoc
// @Summary 合并文件分片
// @Tags 文件
// @Produce json
// @Router /admin/file/upload/merge/{upload_id} [post]
func (c *FileUploadController) MergeChunks(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	uploadID := ctx.Param("upload_id")
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "merge", ResourceType: "file_upload", ResourceCode: "file_upload", ResourceID: uploadID})
	if uploadID == "" {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	result, err := c.service.MergeChunksResponse(ctx.Request.Context(), fileAccessActor(ctx), uploadID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	middleware.SetAuditContext(ctx, middleware.AuditContext{Action: "merge", ResourceType: "file", ResourceCode: "file", ResourceID: strconv.Itoa(result.Id)})
	resp.SetData(result)
}

// GetUploadProgress godoc
// @Summary 获取分片上传进度
// @Tags 文件
// @Produce json
// @Router /admin/file/upload/progress/{upload_id} [get]
func (c *FileUploadController) GetUploadProgress(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	uploadID := ctx.Param("upload_id")
	if uploadID == "" {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	result, err := c.service.GetUploadProgressForUser(ctx.Request.Context(), fileAccessActor(ctx), uploadID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}
