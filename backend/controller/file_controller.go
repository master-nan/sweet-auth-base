/**
 * @Author: Nan
 * @Date: 2024/8/5 下午11:55
 */

package controller

import (
	"backend/config"
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/service"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

// FileController 处理文件上传、下载、预览和短期签名访问。
// 文件接口默认按文件本人/超管授权；低代码业务记录里的文件可通过业务上下文走统一数据权限校验。
type FileController struct {
	fileService           *service.FileService
	sysTableService       *service.SysTableService
	dataPermissionService *service.DataPermissionService
	config                *config.Server
	translators           map[string]ut.Translator
}

type fileBusinessContext struct {
	TableCode string
	RecordId  int
	MenuId    int
	Action    enum.SysMenuButtonEventAction
}

type signedFileAccessClaims struct {
	FileUuid  string `json:"file_uuid"`
	ExpiresAt int64  `json:"expires_at"`
}

type fileAccessURLResponse struct {
	URL       string `json:"url"`
	ExpiresAt int64  `json:"expires_at"`
}

// NewFileController 创建文件控制器实例。
func NewFileController(
	fileService *service.FileService,
	sysTableService *service.SysTableService,
	dataPermissionService *service.DataPermissionService,
	config *config.Server,
	translators map[string]ut.Translator,
) *FileController {
	return &FileController{
		fileService:           fileService,
		sysTableService:       sysTableService,
		dataPermissionService: dataPermissionService,
		config:                config,
		translators:           translators,
	}
}

// Upload 上传文件（小文件直传）
// @Summary 上传文件
// @Description 上传文件，支持秒传（MD5去重）
// @Tags 文件
// @Accept multipart/form-data
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param file formData file true "上传文件"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/file/upload [post]
func (f *FileController) Upload(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		_ = ctx.Error(myerrors.NewBadRequestError("请选择要上传的文件"))
		return
	}

	file, err := f.fileService.Upload(ctx, fileHeader)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(file)
}

// InitChunkUpload 初始化分片上传
// @Summary 初始化分片上传
// @Description 初始化分片上传，返回uploadId和分片信息。支持秒传检测。
// @Tags 文件
// @Accept json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param body body request.ChunkUploadInitReq true "初始化参数"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/file/upload/init [post]
func (f *FileController) InitChunkUpload(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	var req request.ChunkUploadInitReq
	translator := f.translators["zh"]
	err := utils.ValidatorBody[request.ChunkUploadInitReq](ctx, &req, translator)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	result, err := f.fileService.InitChunkUpload(ctx, req)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(result)
}

// UploadChunk 上传单个分片
// @Summary 上传分片
// @Description 上传单个文件分片，支持断点续传（幂等操作）
// @Tags 文件
// @Accept multipart/form-data
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param upload_id formData string true "上传ID"
// @Param chunk_index formData int true "分片索引（从0开始）"
// @Param file formData file true "分片文件"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/file/upload/chunk [post]
func (f *FileController) UploadChunk(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	uploadId := ctx.PostForm("upload_id")
	if uploadId == "" {
		_ = ctx.Error(myerrors.NewBadRequestError("upload_id不能为空"))
		return
	}

	chunkIndexStr := ctx.PostForm("chunk_index")
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		_ = ctx.Error(myerrors.NewBadRequestError("chunk_index参数错误"))
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		_ = ctx.Error(myerrors.NewBadRequestError("请选择要上传的分片文件"))
		return
	}

	err = f.fileService.UploadChunk(ctx, uploadId, chunkIndex, fileHeader)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// MergeChunks 合并分片
// @Summary 合并分片
// @Description 所有分片上传完成后，调用此接口合并分片生成最终文件
// @Tags 文件
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param upload_id path string true "上传ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/file/upload/merge/{upload_id} [post]
func (f *FileController) MergeChunks(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	uploadId := ctx.Param("upload_id")
	if uploadId == "" {
		_ = ctx.Error(myerrors.NewBadRequestError("upload_id不能为空"))
		return
	}

	file, err := f.fileService.MergeChunks(ctx, uploadId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(file)
}

// GetUploadProgress 获取分片上传进度
// @Summary 获取上传进度
// @Description 获取分片上传进度，返回已上传的分片索引列表（用于断点续传）
// @Tags 文件
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param upload_id path string true "上传ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/file/upload/progress/{upload_id} [get]
func (f *FileController) GetUploadProgress(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	uploadId := ctx.Param("upload_id")
	if uploadId == "" {
		_ = ctx.Error(myerrors.NewBadRequestError("upload_id不能为空"))
		return
	}

	progress, err := f.fileService.GetUploadProgressForUser(ctx, uploadId)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(progress)
}

// GetFileById 根据ID获取文件信息
// @Summary 获取文件信息
// @Description 根据ID获取文件详情
// @Tags 文件
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "文件ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/file/{id} [get]
func (f *FileController) GetFileById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	file, err := f.fileService.GetFileById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := f.ensureFileAccess(ctx, file, enum.ButtonActionDetail); err != nil {
		_ = ctx.Error(err)
		return
	}
	resp.SetData(file)
}

// DeleteFileById 根据ID删除文件
// @Summary 删除文件
// @Description 根据ID删除文件（同时删除存储中的物理文件）
// @Tags 文件
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "文件ID"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/file/{id} [delete]
func (f *FileController) DeleteFileById(ctx *gin.Context) {
	resp := response.NewResponse()
	ctx.Set("response", resp)
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	file, err := f.fileService.GetFileById(id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := f.ensureFileAccess(ctx, file, enum.ButtonActionDelete); err != nil {
		_ = ctx.Error(err)
		return
	}
	err = f.fileService.DeleteFileById(ctx, id)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
}

// Download 下载文件
// @Summary 下载文件
// @Description 通过文件UUID下载文件（支持本地和OSS存储）
// @Tags 文件
// @Produce octet-stream
// @Param Authorization header string true "Bearer 用户令牌"
// @Param uuid path string true "文件UUID"
// @Success 200 {file} binary
// @Router /admin/file/download/{uuid} [get]
func (f *FileController) Download(ctx *gin.Context) {
	fileUuid := ctx.Param("uuid")
	if fileUuid == "" {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}
	file, err := f.fileService.GetFileByUuid(fileUuid)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := f.ensureFileAccess(ctx, file, enum.ButtonActionDetail); err != nil {
		_ = ctx.Error(err)
		return
	}
	f.streamFile(ctx, file, true)
}

// PublicPreview 公开预览文件，是否启用由 upload.public_preview 控制。
// @Summary 公开预览文件
// @Description 通过文件UUID公开预览文件，仅在 upload.public_preview 开启时可用
// @Tags 文件
// @Produce octet-stream
// @Param uuid path string true "文件UUID"
// @Success 200 {file} binary
// @Router /files/{uuid} [get]
func (f *FileController) PublicPreview(ctx *gin.Context) {
	if !f.config.Upload.PublicPreview {
		resp := response.NewResponse()
		ctx.Set("response", resp)
		_ = ctx.Error(myerrors.ErrFileNotFound)
		return
	}
	f.preview(ctx, false)
}

// GetFilePreviewAccessURL 生成短期签名文件预览 URL，用于富文本等无法携带 Authorization 头的场景。
// @Summary 获取文件签名预览地址
// @Description 为当前用户可访问的文件生成短期签名预览地址，用于富文本预览等无法携带 Authorization 头的场景
// @Tags 文件
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param uuid path string true "文件UUID"
// @Param ttl query int false "有效期秒数，最大3600秒"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/file/preview-url/{uuid} [get]
func (f *FileController) GetFilePreviewAccessURL(ctx *gin.Context) {
	f.getSignedFileAccessURL(ctx, "preview")
}

// GetFileDownloadAccessURL 生成短期签名文件下载 URL。
// @Summary 获取文件签名下载地址
// @Description 为当前用户可访问的文件生成短期签名下载地址
// @Tags 文件
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param uuid path string true "文件UUID"
// @Param ttl query int false "有效期秒数，最大3600秒"
// @Success 200 {object} response.Response "请求成功"
// @Router /admin/file/download-url/{uuid} [get]
func (f *FileController) GetFileDownloadAccessURL(ctx *gin.Context) {
	f.getSignedFileAccessURL(ctx, "download")
}

func (f *FileController) getSignedFileAccessURL(ctx *gin.Context, action string) {
	resp := response.NewResponse()
	ctx.Set("response", resp)

	fileUuid := ctx.Param("uuid")
	if fileUuid == "" {
		_ = ctx.Error(myerrors.ErrParamInvalid)
		return
	}

	file, err := f.fileService.GetFileByUuid(fileUuid)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	if err := f.ensureFileAccess(ctx, file, defaultFileBusinessAction(ctx, enum.ButtonActionDetail)); err != nil {
		_ = ctx.Error(err)
		return
	}

	ttl, err := parseFileAccessTTL(ctx.Query("ttl"), f.config.Upload)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	expiresAt := time.Now().Add(ttl).Unix()
	token, err := f.signFileAccessToken(signedFileAccessClaims{
		FileUuid:  file.FileUuid,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	resp.SetData(fileAccessURLResponse{
		URL:       signedFileAccessURL(f.config, action, file.FileUuid, token),
		ExpiresAt: expiresAt,
	})
}

// SignedPreview 通过短期签名 URL 预览文件。
// @Summary 签名预览文件
// @Description 通过短期签名 token 预览文件
// @Tags 文件
// @Produce octet-stream
// @Param uuid path string true "文件UUID"
// @Param token query string true "签名 token"
// @Success 200 {file} binary
// @Router /files/access/preview/{uuid} [get]
func (f *FileController) SignedPreview(ctx *gin.Context) {
	f.signedAccess(ctx, false)
}

// SignedDownload 通过短期签名 URL 下载文件。
// @Summary 签名下载文件
// @Description 通过短期签名 token 下载文件
// @Tags 文件
// @Produce octet-stream
// @Param uuid path string true "文件UUID"
// @Param token query string true "签名 token"
// @Success 200 {file} binary
// @Router /files/access/download/{uuid} [get]
func (f *FileController) SignedDownload(ctx *gin.Context) {
	f.signedAccess(ctx, true)
}

func (f *FileController) signedAccess(ctx *gin.Context, attachment bool) {
	claims, err := f.verifyFileAccessToken(ctx.Query("token"))
	if err != nil {
		_ = ctx.Error(myerrors.ErrFileNotFound)
		return
	}
	if ctx.Param("uuid") != claims.FileUuid {
		_ = ctx.Error(myerrors.ErrFileNotFound)
		return
	}
	file, err := f.fileService.GetFileByUuid(claims.FileUuid)
	if err != nil {
		_ = ctx.Error(myerrors.ErrFileNotFound)
		return
	}
	f.streamFile(ctx, file, attachment)
}

// Preview 预览文件（需要登录验证）
// @Summary 预览文件
// @Description 通过文件UUID预览文件（图片、PDF等），支持本地和OSS存储
// @Tags 文件
// @Produce octet-stream
// @Param Authorization header string true "Bearer 用户令牌"
// @Param uuid path string true "文件UUID"
// @Success 200 {file} binary
// @Router /admin/file/preview/{uuid} [get]
func (f *FileController) Preview(ctx *gin.Context) {
	f.preview(ctx, true)
}

func (f *FileController) preview(ctx *gin.Context, requireOwner bool) {
	fileUuid := ctx.Param("uuid")
	if fileUuid == "" {
		_ = ctx.Error(myerrors.ErrFileNotFound)
		return
	}
	file, err := f.fileService.GetFileByUuid(fileUuid)
	if err != nil {
		_ = ctx.Error(myerrors.ErrFileNotFound)
		return
	}
	if requireOwner {
		if err := f.ensureFileAccess(ctx, file, enum.ButtonActionDetail); err != nil {
			_ = ctx.Error(err)
			return
		}
	}
	f.streamFile(ctx, file, false)
}

func (f *FileController) ensureFileAccess(ctx *gin.Context, file model.File, fallbackAction enum.SysMenuButtonEventAction) error {
	bizCtx, hasBizCtx, err := parseFileBusinessContext(ctx, fallbackAction)
	if err != nil {
		return err
	}
	if hasBizCtx {
		return f.ensureBusinessFileAccess(ctx, file, bizCtx)
	}
	user := ctx.MustGet("user").(model.SysUser)
	if utils.IsSuperAdmin(user) || (file.CreateUser != nil && *file.CreateUser == user.Id) {
		return nil
	}
	return myerrors.ErrPermissionDenied
}

func (f *FileController) ensureBusinessFileAccess(ctx *gin.Context, file model.File, bizCtx fileBusinessContext) error {
	if f.sysTableService == nil || f.dataPermissionService == nil {
		return myerrors.ErrPermissionDenied
	}
	table, err := f.sysTableService.GetTableByTableCode(bizCtx.TableCode)
	if err != nil {
		return err
	}
	if table.Id == 0 {
		return myerrors.ErrParamInvalid
	}
	user := ctx.MustGet("user").(model.SysUser)
	if !utils.IsSuperAdmin(user) {
		if err := f.dataPermissionService.CheckRecordDataScope(user, bizCtx.MenuId, table, bizCtx.RecordId, bizCtx.Action); err != nil {
			return err
		}
	}
	if file.FileUuid == "" {
		return myerrors.ErrFileNotFound
	}
	matchesUuid, err := f.dataPermissionService.RecordContainsValue(table, bizCtx.RecordId, file.FileUuid)
	if err != nil {
		return err
	}
	matchesId := false
	if file.Id > 0 {
		matchesId, err = f.dataPermissionService.RecordContainsValue(table, bizCtx.RecordId, strconv.Itoa(file.Id))
		if err != nil {
			return err
		}
	}
	if !matchesUuid && !matchesId {
		return myerrors.ErrPermissionDenied
	}
	return nil
}

func parseFileBusinessContext(ctx *gin.Context, fallbackAction enum.SysMenuButtonEventAction) (fileBusinessContext, bool, error) {
	tableCode := strings.TrimSpace(ctx.Query("table_code"))
	recordIdRaw := strings.TrimSpace(firstNonEmpty(ctx.Query("record_id"), ctx.Query("row_id"), ctx.Query("id")))
	if tableCode == "" && recordIdRaw == "" {
		return fileBusinessContext{}, false, nil
	}
	if tableCode == "" || recordIdRaw == "" {
		return fileBusinessContext{}, true, myerrors.ErrParamInvalid
	}
	recordId, err := strconv.Atoi(recordIdRaw)
	if err != nil || recordId <= 0 {
		return fileBusinessContext{}, true, myerrors.ErrParamInvalid
	}
	menuId := 0
	if rawMenuId := strings.TrimSpace(ctx.Query("menu_id")); rawMenuId != "" {
		menuId, err = strconv.Atoi(rawMenuId)
		if err != nil || menuId < 0 {
			return fileBusinessContext{}, true, myerrors.ErrParamInvalid
		}
	}
	action := fallbackAction
	if rawAction := strings.TrimSpace(ctx.Query("action")); rawAction != "" {
		normalized, ok := enum.NormalizeSysMenuButtonEventAction(rawAction)
		if !ok {
			return fileBusinessContext{}, true, myerrors.ErrParamInvalid
		}
		action = normalized
	}
	return fileBusinessContext{TableCode: tableCode, RecordId: recordId, MenuId: menuId, Action: action}, true, nil
}

func defaultFileBusinessAction(ctx *gin.Context, fallbackAction enum.SysMenuButtonEventAction) enum.SysMenuButtonEventAction {
	if rawAction := strings.TrimSpace(ctx.Query("action")); rawAction != "" {
		if action, ok := enum.NormalizeSysMenuButtonEventAction(rawAction); ok {
			return action
		}
	}
	return fallbackAction
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// streamFile 输出文件内容，并根据预览/下载设置响应头。
func (f *FileController) streamFile(ctx *gin.Context, file model.File, attachment bool) {
	reader, err := f.fileService.GetFileContent(file)
	if err != nil {
		_ = ctx.Error(myerrors.ErrFileNotFound)
		return
	}
	defer reader.Close()

	contentType := safeContentType(file.FileType)
	forceAttachment := attachment
	if !attachment {
		var safeInline bool
		contentType, safeInline = safeInlinePreviewContentType(contentType, f.config.Upload.InlinePreviewMimes)
		forceAttachment = !safeInline
	}

	ctx.Header("Content-Type", contentType)
	ctx.Header("Content-Length", strconv.FormatInt(file.FileSize, 10))
	ctx.Header("X-Content-Type-Options", "nosniff")
	if forceAttachment {
		ctx.Header("Content-Disposition", contentDisposition("attachment", file.FileName))
		ctx.Header("Cache-Control", "private, no-store")
	} else {
		ctx.Header("Content-Disposition", contentDisposition("inline", file.FileName))
		ctx.Header("Cache-Control", "private, max-age=300")
		ctx.Header("Content-Security-Policy", "sandbox")
	}
	_, _ = io.Copy(ctx.Writer, reader)
}

// safeContentType 标准化文件 MIME，避免响应头注入。
func safeContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return "application/octet-stream"
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "application/octet-stream"
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if mediaType == "" || strings.ContainsAny(mediaType, "\r\n") {
		return "application/octet-stream"
	}
	return mediaType
}

// safeInlinePreviewContentType 校验文件是否允许以内联方式预览。
// 上传白名单只表示文件允许保存到系统，内联预览白名单表示浏览器可以直接渲染；
// 例如 Office、压缩包这类文件可以上传，但不应该 inline 输出，避免浏览器误解析带来安全风险。
func safeInlinePreviewContentType(contentType string, inlinePreviewMimes []string) (string, bool) {
	contentType = safeContentType(contentType)
	for _, item := range inlinePreviewMimes {
		if contentType == safeContentType(item) {
			return contentType, true
		}
	}
	return "application/octet-stream", false
}

// contentDisposition 生成安全的 Content-Disposition 文件名。
func contentDisposition(disposition string, fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		fileName = "download"
	}
	encoded := url.PathEscape(fileName)
	return fmt.Sprintf("%s; filename*=UTF-8''%s", disposition, encoded)
}

// parseFileAccessTTL 解析签名访问有效期，默认值和最大值来自 upload 配置。
func parseFileAccessTTL(raw string, upload config.Upload) (time.Duration, error) {
	defaultMinutes := upload.AccessTTLMinutes
	if defaultMinutes <= 0 {
		return 0, myerrors.NewBadRequestError("文件访问有效期配置错误")
	}
	maxMinutes := upload.MaxAccessTTLMinutes
	if maxMinutes <= 0 {
		maxMinutes = defaultMinutes
	}
	defaultTTL := time.Duration(defaultMinutes) * time.Minute
	maxTTL := time.Duration(maxMinutes) * time.Minute
	if raw == "" {
		return defaultTTL, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return defaultTTL, nil
	}
	ttl := time.Duration(seconds) * time.Second
	if ttl > maxTTL {
		return maxTTL, nil
	}
	return ttl, nil
}

// fileAccessBaseURL 返回文件访问 URL 前缀。
func fileAccessBaseURL(cfg *config.Server) string {
	baseURL := ""
	if cfg != nil {
		baseURL = strings.TrimRight(strings.TrimSpace(cfg.Upload.BaseURL), "/")
	}
	return baseURL
}

// signedFileAccessURL 拼接短期签名访问地址。
func signedFileAccessURL(cfg *config.Server, action string, fileUuid string, token string) string {
	return fmt.Sprintf("%s/access/%s/%s?token=%s", fileAccessBaseURL(cfg), action, url.PathEscape(fileUuid), url.QueryEscape(token))
}

// signFileAccessToken 生成短期文件访问 token。
func (f *FileController) signFileAccessToken(claims signedFileAccessClaims) (string, error) {
	if strings.TrimSpace(claims.FileUuid) == "" || claims.ExpiresAt <= 0 {
		return "", myerrors.ErrParamInvalid
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := f.signFileAccessPayload(encodedPayload)
	return encodedPayload + "." + signature, nil
}

// verifyFileAccessToken 校验短期文件访问 token。
func (f *FileController) verifyFileAccessToken(token string) (signedFileAccessClaims, error) {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return signedFileAccessClaims{}, myerrors.ErrParamInvalid
	}
	expectedSignature := f.signFileAccessPayload(parts[0])
	if !hmac.Equal([]byte(parts[1]), []byte(expectedSignature)) {
		return signedFileAccessClaims{}, myerrors.ErrParamInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return signedFileAccessClaims{}, err
	}
	var claims signedFileAccessClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return signedFileAccessClaims{}, err
	}
	if claims.FileUuid == "" || claims.ExpiresAt <= time.Now().Unix() {
		return signedFileAccessClaims{}, myerrors.ErrParamInvalid
	}
	return claims, nil
}

// signFileAccessPayload 对 token payload 做 HMAC 签名。
func (f *FileController) signFileAccessPayload(payload string) string {
	mac := hmac.New(sha256.New, f.fileAccessSecret())
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// fileAccessSecret 生成签名密钥，优先使用 session.secret。
func (f *FileController) fileAccessSecret() []byte {
	secret := ""
	if f != nil && f.config != nil {
		secret = strings.TrimSpace(f.config.Session.Secret)
		if secret == "" {
			secret = strings.TrimSpace(f.config.Conf.Salt)
		}
	}
	return []byte("sweet_admin_file_access:" + secret)
}
