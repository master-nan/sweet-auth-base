package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/audit"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	notificationRecipientLimit = 1000
	notificationContentBytes   = 16 * 1024
	notificationContentRunes   = 4000
	notificationPreviewRunes   = 120
)

type NotificationCommand struct {
	Recipients     []int
	Category       model.NotificationCategory
	Level          model.NotificationLevel
	Title          string
	Content        string
	SourceModule   string
	SourceType     string
	SourceId       string
	ActionMenuName string
	ActionPath     string
	DedupKey       string
}

type NotificationSendResult struct {
	NotificationId         int
	Deduplicated           bool
	CreatedRecipientCount  int
	ExistingRecipientCount int
}

// NotificationService 是站内消息发送与当前用户收件箱的唯一应用边界。
// 发送端只提交稳定业务事实，读取端始终从审计 Context 取得认证用户。
type NotificationService struct {
	repository repository.NotificationRepository
	sf         *utils.Snowflake
	audit      StandardContextAuditWriter
}

func NewNotificationService(
	repository repository.NotificationRepository,
	sf *utils.Snowflake,
	audit StandardContextAuditWriter,
) *NotificationService {
	return &NotificationService{repository: repository, sf: sf, audit: audit}
}

// Send 在一个短事务中创建不可变消息事实及全部收件人。
// dedup_key 重复时复用既有消息，只补齐缺少的收件人。
func (service *NotificationService) Send(
	ctx context.Context,
	command NotificationCommand,
) (NotificationSendResult, error) {
	normalized, err := normalizeNotificationCommand(command)
	if err != nil {
		return NotificationSendResult{}, err
	}
	if normalized.DedupKey != "" {
		existing, findErr := service.repository.FindByDedup(ctx, normalized.SourceModule, normalized.DedupKey)
		if findErr == nil {
			return service.sendDeduplicated(ctx, existing, normalized)
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return NotificationSendResult{}, myerrors.WrapDatabaseError(findErr)
		}
	}

	id, err := service.sf.GenerateUniqueID()
	if err != nil {
		return NotificationSendResult{}, myerrors.WrapSystemError(err)
	}
	value := notificationFromCommand(int(id), normalized)
	err = RunInTransaction(ctx, service.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		if err := service.validateReferences(tx, normalized); err != nil {
			return err
		}
		if err := service.repository.CreateNotification(tx, &value); err != nil {
			return err
		}
		recipients := notificationRecipients(value.Id, normalized.Recipients)
		created, err := service.repository.CreateRecipients(tx, recipients, false)
		if err != nil {
			return err
		}
		if int(created) != len(recipients) {
			return myerrors.ErrNotificationInvalidRecipients
		}
		return service.writeSendAudit(ctx, tx, value, len(recipients), false)
	})
	if err == nil {
		return NotificationSendResult{
			NotificationId:        value.Id,
			CreatedRecipientCount: len(normalized.Recipients),
		}, nil
	}
	if normalized.DedupKey != "" && isNotificationDedupConflict(err) {
		existing, findErr := service.repository.FindByDedup(ctx, normalized.SourceModule, normalized.DedupKey)
		if findErr != nil {
			return NotificationSendResult{}, myerrors.WrapDatabaseError(findErr)
		}
		return service.sendDeduplicated(ctx, existing, normalized)
	}
	if _, ok := myerrors.AsApplicationError(err); ok {
		return NotificationSendResult{}, err
	}
	return NotificationSendResult{}, myerrors.WrapDatabaseError(err)
}

func (service *NotificationService) sendDeduplicated(
	ctx context.Context,
	existing model.Notification,
	command NotificationCommand,
) (NotificationSendResult, error) {
	if !notificationMatchesCommand(existing, command) {
		return NotificationSendResult{}, myerrors.ErrNotificationDedupConflict
	}
	result := NotificationSendResult{NotificationId: existing.Id, Deduplicated: true}
	err := RunInTransaction(ctx, service.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		if err := service.validateReferences(tx, command); err != nil {
			return err
		}
		currentIds, err := service.repository.RecipientUserIDs(tx, existing.Id)
		if err != nil {
			return err
		}
		current := make(map[int]struct{}, len(currentIds))
		for _, userId := range currentIds {
			current[userId] = struct{}{}
		}
		missing := make([]int, 0, len(command.Recipients))
		for _, userId := range command.Recipients {
			if _, exists := current[userId]; exists {
				result.ExistingRecipientCount++
			} else {
				missing = append(missing, userId)
			}
		}
		created, err := service.repository.CreateRecipients(tx, notificationRecipients(existing.Id, missing), true)
		if err != nil {
			return err
		}
		result.CreatedRecipientCount = int(created)
		return service.writeSendAudit(ctx, tx, existing, len(command.Recipients), true)
	})
	if err != nil {
		if _, ok := myerrors.AsApplicationError(err); ok {
			return NotificationSendResult{}, err
		}
		return NotificationSendResult{}, myerrors.WrapDatabaseError(err)
	}
	return result, nil
}

func (service *NotificationService) UnreadCount(ctx context.Context) (response.NotificationUnreadCountRes, error) {
	userId, err := notificationActor(ctx)
	if err != nil {
		return response.NotificationUnreadCountRes{}, err
	}
	count, err := service.repository.UnreadCount(ctx, userId)
	if err != nil {
		return response.NotificationUnreadCountRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NotificationUnreadCountRes{UnreadCount: count}, nil
}

func (service *NotificationService) Recent(
	ctx context.Context,
	limit int,
) ([]response.NotificationSummaryRes, error) {
	userId, err := notificationActor(ctx)
	if err != nil {
		return nil, err
	}
	if limit == 0 {
		limit = 8
	}
	if limit < 1 || limit > 10 {
		return nil, myerrors.ErrParamInvalid
	}
	values, err := service.repository.Recent(ctx, userId, limit)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	return service.summaries(ctx, userId, values)
}

func (service *NotificationService) Query(
	ctx context.Context,
	req request.NotificationQueryReq,
) (response.ListResult[response.NotificationSummaryRes], error) {
	userId, err := notificationActor(ctx)
	if err != nil {
		return response.ListResult[response.NotificationSummaryRes]{}, err
	}
	if req.ReadStatus == "" {
		req.ReadStatus = model.NotificationReadAll
	}
	if !req.ReadStatus.Valid() || (req.Category != "" && !req.Category.Valid()) {
		return response.ListResult[response.NotificationSummaryRes]{}, myerrors.ErrParamInvalid
	}
	page, err := service.repository.Query(ctx, repository.NotificationListFilter{
		UserId: userId, Page: req.Page, Num: req.Num, Keyword: req.Keyword,
		ReadStatus: req.ReadStatus, Category: req.Category,
	})
	if err != nil {
		return response.ListResult[response.NotificationSummaryRes]{}, myerrors.WrapDatabaseError(err)
	}
	items, err := service.summaries(ctx, userId, page.Data)
	if err != nil {
		return response.ListResult[response.NotificationSummaryRes]{}, err
	}
	return response.ListResult[response.NotificationSummaryRes]{Data: items, Total: int(page.Total)}, nil
}

func (service *NotificationService) Detail(
	ctx context.Context,
	notificationId int,
) (response.NotificationDetailRes, error) {
	userId, err := notificationActor(ctx)
	if err != nil {
		return response.NotificationDetailRes{}, err
	}
	value, err := service.repository.Detail(ctx, userId, notificationId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.NotificationDetailRes{}, myerrors.ErrNotificationNotVisible
	}
	if err != nil {
		return response.NotificationDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	summaries, err := service.summaries(ctx, userId, []repository.NotificationRecord{value})
	if err != nil {
		return response.NotificationDetailRes{}, err
	}
	return response.NotificationDetailRes{
		NotificationSummaryRes: summaries[0],
		Content:                value.Content,
		Source: response.NotificationSourceRes{
			Module: value.SourceModule, Type: value.SourceType, Id: value.SourceId,
		},
	}, nil
}

func (service *NotificationService) MarkRead(
	ctx context.Context,
	notificationId int,
) (response.NotificationDetailRes, error) {
	userId, err := notificationActor(ctx)
	if err != nil {
		return response.NotificationDetailRes{}, err
	}
	_, err = service.repository.MarkRead(ctx, userId, notificationId, model.CustomTime(model.Now()))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.NotificationDetailRes{}, myerrors.ErrNotificationNotVisible
	}
	if err != nil {
		return response.NotificationDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return service.Detail(ctx, notificationId)
}

func (service *NotificationService) MarkAllRead(
	ctx context.Context,
) (response.NotificationMarkAllReadRes, error) {
	userId, err := notificationActor(ctx)
	if err != nil {
		return response.NotificationMarkAllReadRes{}, err
	}
	count, err := service.repository.MarkAllRead(ctx, userId, model.CustomTime(model.Now()))
	if err != nil {
		return response.NotificationMarkAllReadRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NotificationMarkAllReadRes{UpdatedCount: count}, nil
}

func (service *NotificationService) summaries(
	ctx context.Context,
	userId int,
	values []repository.NotificationRecord,
) ([]response.NotificationSummaryRes, error) {
	menuNames := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.ActionMenuName != "" {
			if _, exists := seen[value.ActionMenuName]; !exists {
				seen[value.ActionMenuName] = struct{}{}
				menuNames = append(menuNames, value.ActionMenuName)
			}
		}
	}
	accessible, err := service.repository.AccessibleMenuNames(ctx, userId, menuNames)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	result := make([]response.NotificationSummaryRes, 0, len(values))
	for _, value := range values {
		item := response.NotificationSummaryRes{
			Id: value.Id, Category: value.Category, Level: value.Level,
			Title: value.Title, ContentPreview: notificationContentPreview(value.Content),
			Read: value.ReadAt != nil, ReadAt: value.ReadAt, CreatedAt: value.CreatedAt,
		}
		if value.ActionMenuName != "" {
			allowed := accessible[value.ActionMenuName]
			item.Action = &response.NotificationActionRes{Available: allowed}
			if allowed {
				item.Action.Path = value.ActionPath
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func (service *NotificationService) validateReferences(db *gorm.DB, command NotificationCommand) error {
	count, err := service.repository.CountActiveUsers(db, command.Recipients)
	if err != nil {
		return err
	}
	if int(count) != len(command.Recipients) {
		return myerrors.ErrNotificationInvalidRecipients
	}
	if command.ActionMenuName == "" {
		return nil
	}
	exists, err := service.repository.ActiveMenuExists(db, command.ActionMenuName)
	if err != nil {
		return err
	}
	if !exists {
		return myerrors.ErrNotificationInvalidAction
	}
	return nil
}

func (service *NotificationService) writeSendAudit(
	ctx context.Context,
	tx *gorm.DB,
	value model.Notification,
	recipientCount int,
	deduplicated bool,
) error {
	return service.audit.RecordTransactionalAuditContext(ctx, tx, TransactionalAuditRecord{
		Action: "send", ResourceType: "notification", ResourceCode: value.SourceModule,
		ResourceId: strconv.Itoa(value.Id),
		Changes: map[string]TransactionalAuditChange{
			"category":        {NewValue: value.Category},
			"source_type":     {NewValue: value.SourceType},
			"source_id":       {NewValue: value.SourceId},
			"recipient_count": {NewValue: recipientCount},
			"has_action":      {NewValue: value.ActionMenuName != ""},
			"dedup_present":   {NewValue: value.DedupKey != nil},
			"deduplicated":    {NewValue: deduplicated},
		},
	})
}

func notificationActor(ctx context.Context) (int, error) {
	subject, ok := audit.GetAuditSubject(ctx)
	if !ok {
		return 0, myerrors.ErrUserNotLogin
	}
	return subject.UserID, nil
}

func normalizeNotificationCommand(command NotificationCommand) (NotificationCommand, error) {
	for _, userId := range command.Recipients {
		if userId <= 0 {
			return NotificationCommand{}, myerrors.ErrNotificationInvalidRecipients
		}
	}
	command.Recipients = sortedUniquePositiveInts(command.Recipients)
	if len(command.Recipients) == 0 {
		return NotificationCommand{}, myerrors.ErrNotificationInvalidRecipients
	}
	if len(command.Recipients) > notificationRecipientLimit {
		return NotificationCommand{}, myerrors.ErrNotificationRecipientLimit
	}
	command.Title = strings.TrimSpace(command.Title)
	command.Content = strings.TrimSpace(command.Content)
	command.SourceModule = strings.TrimSpace(command.SourceModule)
	command.SourceType = strings.TrimSpace(command.SourceType)
	command.SourceId = strings.TrimSpace(command.SourceId)
	command.ActionMenuName = strings.TrimSpace(command.ActionMenuName)
	command.ActionPath = strings.TrimSpace(command.ActionPath)
	command.DedupKey = strings.TrimSpace(command.DedupKey)
	if !command.Category.Valid() {
		return NotificationCommand{}, myerrors.ErrNotificationInvalidCategory
	}
	if !command.Level.Valid() {
		return NotificationCommand{}, myerrors.ErrNotificationInvalidLevel
	}
	if !validRuneLength(command.Title, 1, 160) ||
		!validRuneLength(command.SourceModule, 1, 64) ||
		!validRuneLength(command.SourceType, 1, 64) ||
		!validRuneLength(command.SourceId, 0, 128) ||
		!validRuneLength(command.ActionMenuName, 0, 32) ||
		!validRuneLength(command.DedupKey, 0, 128) {
		return NotificationCommand{}, myerrors.ErrParamInvalid
	}
	if !utf8.ValidString(command.Content) || utf8.RuneCountInString(command.Content) < 1 ||
		utf8.RuneCountInString(command.Content) > notificationContentRunes || len(command.Content) > notificationContentBytes {
		return NotificationCommand{}, myerrors.ErrNotificationPayloadTooLarge
	}
	if err := validateNotificationAction(command.ActionMenuName, command.ActionPath); err != nil {
		return NotificationCommand{}, err
	}
	return command, nil
}

func validateNotificationAction(menuName, actionPath string) error {
	if menuName == "" && actionPath == "" {
		return nil
	}
	if menuName == "" || !validRuneLength(actionPath, 0, 512) {
		return myerrors.ErrNotificationInvalidAction
	}
	if actionPath == "" {
		return nil
	}
	if !strings.HasPrefix(actionPath, "/admin/") || strings.HasPrefix(actionPath, "//") ||
		strings.ContainsAny(actionPath, "?#\\") || path.Clean(actionPath) != actionPath {
		return myerrors.ErrNotificationInvalidAction
	}
	for _, char := range actionPath {
		if unicode.IsControl(char) {
			return myerrors.ErrNotificationInvalidAction
		}
	}
	return nil
}

func validRuneLength(value string, min, max int) bool {
	if !utf8.ValidString(value) {
		return false
	}
	length := utf8.RuneCountInString(value)
	return length >= min && length <= max
}

func sortedUniquePositiveInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	sort.Ints(result)
	return result
}

func notificationFromCommand(id int, command NotificationCommand) model.Notification {
	value := model.Notification{
		Id: id, Category: command.Category, Level: command.Level,
		Title: command.Title, Content: command.Content,
		SourceModule: command.SourceModule, SourceType: command.SourceType, SourceId: command.SourceId,
		ActionMenuName: command.ActionMenuName, ActionPath: command.ActionPath,
	}
	if command.DedupKey != "" {
		value.DedupKey = &command.DedupKey
	}
	return value
}

func notificationRecipients(notificationId int, userIds []int) []model.NotificationRecipient {
	values := make([]model.NotificationRecipient, 0, len(userIds))
	for _, userId := range userIds {
		values = append(values, model.NotificationRecipient{NotificationId: notificationId, UserId: userId})
	}
	return values
}

func notificationMatchesCommand(value model.Notification, command NotificationCommand) bool {
	return value.Category == command.Category && value.Level == command.Level &&
		value.Title == command.Title && value.Content == command.Content &&
		value.SourceModule == command.SourceModule && value.SourceType == command.SourceType &&
		value.SourceId == command.SourceId && value.ActionMenuName == command.ActionMenuName &&
		value.ActionPath == command.ActionPath
}

func notificationContentPreview(content string) string {
	normalized := strings.Join(strings.Fields(content), " ")
	runes := []rune(normalized)
	if len(runes) <= notificationPreviewRunes {
		return normalized
	}
	return string(runes[:notificationPreviewRunes]) + "…"
}

func isNotificationDedupConflict(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == "ux_notification_source_dedup"
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "unique") && strings.Contains(lower, "notification") && strings.Contains(lower, "dedup")
}
