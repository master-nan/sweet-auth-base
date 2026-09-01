package service

import (
	"backend/dto/request"
	"backend/internal/audit"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

type notificationAuditWriter struct {
	records []TransactionalAuditRecord
}

func (writer *notificationAuditWriter) RecordTransactionalAuditContext(
	_ context.Context,
	_ *gorm.DB,
	record TransactionalAuditRecord,
) error {
	writer.records = append(writer.records, record)
	return nil
}

func TestNotificationServiceSendDedupIsolationAndRead(t *testing.T) {
	service, db, writer := newNotificationTestSubject(t)
	ctxA := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(101, "user-a"))
	ctxB := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(102, "user-b"))
	command := NotificationCommand{
		Recipients: []int{101, 102, 101}, Category: model.NotificationCategoryReminder,
		Level: model.NotificationLevelInfo, Title: "学习计划已发布", Content: "请按时完成学习任务。",
		SourceModule: "learning", SourceType: "learning_plan", SourceId: "9001",
		DedupKey: "learning-plan-9001-published",
	}

	first, err := service.Send(ctxA, command)
	if err != nil {
		t.Fatalf("send notification: %v", err)
	}
	if first.Deduplicated || first.CreatedRecipientCount != 2 {
		t.Fatalf("unexpected first send result: %+v", first)
	}
	command.Recipients = []int{102, 101, 102}
	second, err := service.Send(ctxA, command)
	if err != nil {
		t.Fatalf("deduplicated send: %v", err)
	}
	if !second.Deduplicated || second.NotificationId != first.NotificationId ||
		second.CreatedRecipientCount != 0 || second.ExistingRecipientCount != 2 {
		t.Fatalf("unexpected deduplicated result: %+v", second)
	}
	moreRecipients := command
	moreRecipients.Recipients = []int{101, 102, 103}
	if _, err := service.Send(ctxA, moreRecipients); !errors.Is(err, myerrors.ErrNotificationDedupConflict) {
		t.Fatalf("expanded recipient set error=%v", err)
	}
	fewerRecipients := command
	fewerRecipients.Recipients = []int{101}
	if _, err := service.Send(ctxA, fewerRecipients); !errors.Is(err, myerrors.ErrNotificationDedupConflict) {
		t.Fatalf("reduced recipient set error=%v", err)
	}
	changedContent := command
	changedContent.Content = "已变更的学习任务内容。"
	if _, err := service.Send(ctxA, changedContent); !errors.Is(err, myerrors.ErrNotificationDedupConflict) {
		t.Fatalf("changed notification content error=%v", err)
	}
	var notificationCount int64
	var recipientCount int64
	if err := db.Model(&model.Notification{}).Count(&notificationCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.NotificationRecipient{}).Count(&recipientCount).Error; err != nil {
		t.Fatal(err)
	}
	if notificationCount != 1 || recipientCount != 2 {
		t.Fatalf("notification count=%d recipient count=%d", notificationCount, recipientCount)
	}

	pageA, err := service.Query(ctxA, request.NotificationQueryReq{ReadStatus: model.NotificationReadAll})
	if err != nil || pageA.Total != 1 || len(pageA.Data) != 1 {
		t.Fatalf("user A page=%+v err=%v", pageA, err)
	}
	pageB, err := service.Query(ctxB, request.NotificationQueryReq{ReadStatus: model.NotificationReadAll})
	if err != nil || pageB.Total != 1 {
		t.Fatalf("user B page=%+v err=%v", pageB, err)
	}
	if _, err := service.MarkRead(ctxB, first.NotificationId); err != nil {
		t.Fatalf("recipient B mark read: %v", err)
	}
	if _, err := service.Detail(
		audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(103, "user-c")),
		first.NotificationId,
	); !errors.Is(err, myerrors.ErrNotificationNotVisible) {
		t.Fatalf("cross-user detail error=%v", err)
	}
	if _, err := service.MarkRead(
		audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(103, "user-c")),
		first.NotificationId,
	); !errors.Is(err, myerrors.ErrNotificationNotVisible) {
		t.Fatalf("cross-user mark read error=%v", err)
	}
	ctxC := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(103, "user-c"))
	pageC, err := service.Query(ctxC, request.NotificationQueryReq{ReadStatus: model.NotificationReadAll})
	if err != nil || pageC.Total != 0 || len(pageC.Data) != 0 {
		t.Fatalf("cross-user query page=%+v err=%v", pageC, err)
	}
	recentC, err := service.Recent(ctxC, 8)
	if err != nil || len(recentC) != 0 {
		t.Fatalf("cross-user recent=%+v err=%v", recentC, err)
	}
	unreadC, err := service.UnreadCount(ctxC)
	if err != nil || unreadC.UnreadCount != 0 {
		t.Fatalf("cross-user unread=%+v err=%v", unreadC, err)
	}
	if len(writer.records) != 2 || writer.records[0].Changes["recipient_count"].NewValue != 2 ||
		writer.records[1].Changes["deduplicated"].NewValue != true {
		t.Fatalf("unexpected safe audit records: %+v", writer.records)
	}
}

func TestNotificationServiceActionAvailabilityAndDTO(t *testing.T) {
	service, db, _ := newNotificationTestSubject(t)
	rootMenu := model.SysMenu{Basic: model.Basic{Id: 200, State: true}, Name: "integration", Path: "integration"}
	menu := model.SysMenu{Basic: model.Basic{Id: 201, State: true}, Pid: rootMenu.Id, Name: "integration_execution", Path: "execution"}
	otherMenu := model.SysMenu{Basic: model.Basic{Id: 202, State: true}, Pid: rootMenu.Id, Name: "integration_credential", Path: "credential"}
	role := model.SysRole{Basic: model.Basic{Id: 301, State: true}, Name: "notification-reader"}
	testutil.MustCreate(t, db, &rootMenu)
	testutil.MustCreate(t, db, &menu)
	testutil.MustCreate(t, db, &otherMenu)
	testutil.MustCreate(t, db, &role)
	testutil.MustCreate(t, db, &model.SysRoleMenu{RoleId: role.Id, MenuId: menu.Id})
	testutil.MustCreate(t, db, &model.SysUserRole{UserId: 101, RoleId: role.Id})
	command := NotificationCommand{
		Recipients: []int{101, 102}, Category: model.NotificationCategoryIntegration,
		Level: model.NotificationLevelWarning, Title: "同步执行失败", Content: "<script>alert('xss')</script>\n请检查执行日志。",
		SourceModule: "integration", SourceType: "execution", SourceId: "7001",
		ActionMenuName: menu.Name, ActionPath: "/admin/integration/execution",
		DedupKey: "exam-7001-result",
	}
	result, err := service.Send(context.Background(), command)
	if err != nil {
		t.Fatalf("send action notification: %v", err)
	}
	ctxA := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(101, "user-a"))
	ctxB := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(102, "user-b"))
	detailA, err := service.Detail(ctxA, result.NotificationId)
	if err != nil || detailA.Action == nil || !detailA.Action.Available || detailA.Action.Path == "" {
		t.Fatalf("authorized detail=%+v err=%v", detailA, err)
	}
	detailB, err := service.Detail(ctxB, result.NotificationId)
	if err != nil || detailB.Action == nil || detailB.Action.Available || detailB.Action.Path != "" {
		t.Fatalf("unauthorized action detail=%+v err=%v", detailB, err)
	}
	if detailA.Content != command.Content || !strings.Contains(detailA.Content, "<script>") {
		t.Fatalf("content must remain plain DTO text: %q", detailA.Content)
	}

	invalidCases := []struct {
		name     string
		menuName string
		path     string
	}{
		{name: "missing menu", menuName: "missing_menu", path: "/admin/integration/execution"},
		{name: "another menu path", menuName: menu.Name, path: "/admin/integration/credential"},
		{name: "unbound hidden detail", menuName: menu.Name, path: "/admin/detail/integration/execution/7001"},
	}
	for _, testCase := range invalidCases {
		t.Run(testCase.name, func(t *testing.T) {
			invalid := command
			invalid.ActionMenuName = testCase.menuName
			invalid.ActionPath = testCase.path
			invalid.DedupKey = ""
			if _, err := service.Send(context.Background(), invalid); !errors.Is(err, myerrors.ErrNotificationInvalidAction) {
				t.Fatalf("invalid action error=%v", err)
			}
		})
	}

	invalidStored := model.Notification{
		Basic: model.Basic{Id: 9901, State: true}, Category: model.NotificationCategoryIntegration, Level: model.NotificationLevelWarning,
		Title: "异常历史跳转", Content: "内容仍然可见。", SourceModule: "integration", SourceType: "execution",
		ActionMenuName: menu.Name, ActionPath: "/admin/integration/credential",
	}
	testutil.MustCreate(t, db, &invalidStored)
	testutil.MustCreate(t, db, &model.NotificationRecipient{NotificationId: invalidStored.Id, UserId: 101})
	storedDetail, err := service.Detail(ctxA, invalidStored.Id)
	if err != nil || storedDetail.Action == nil || storedDetail.Action.Available || storedDetail.Action.Path != "" {
		t.Fatalf("invalid stored action was exposed: detail=%+v err=%v", storedDetail, err)
	}
}

func TestNotificationServiceValidationAndExamContract(t *testing.T) {
	valid := NotificationCommand{
		Recipients: []int{1}, Category: model.NotificationCategoryReminder,
		Level: model.NotificationLevelInfo, Title: "考试即将开始", Content: "请提前进入考试页面。",
		SourceModule: "exam", SourceType: "exam", SourceId: "456",
		ActionMenuName: "exam", ActionPath: "/admin/exam/exams/456", DedupKey: "exam:456:starts:2026",
	}
	if normalized, err := normalizeNotificationCommand(valid); err != nil || normalized.SourceModule != "exam" {
		t.Fatalf("exam command normalization=%+v err=%v", normalized, err)
	}
	invalidAction := valid
	invalidAction.ActionPath = "https://example.com/exam"
	if _, err := normalizeNotificationCommand(invalidAction); !errors.Is(err, myerrors.ErrNotificationInvalidAction) {
		t.Fatalf("external action error=%v", err)
	}
	for _, invalidPath := range []string{
		"/admin/exam/exams/456?source=notification",
		"/admin/exam/exams/456#detail",
	} {
		invalidAction.ActionPath = invalidPath
		if _, err := normalizeNotificationCommand(invalidAction); !errors.Is(err, myerrors.ErrNotificationInvalidAction) {
			t.Fatalf("query or fragment action %q error=%v", invalidPath, err)
		}
	}
	missingPath := valid
	missingPath.ActionPath = ""
	if _, err := normalizeNotificationCommand(missingPath); !errors.Is(err, myerrors.ErrNotificationInvalidAction) {
		t.Fatalf("menu without action path error=%v", err)
	}
	tooMany := valid
	tooMany.Recipients = make([]int, notificationRecipientLimit+1)
	for index := range tooMany.Recipients {
		tooMany.Recipients[index] = index + 1
	}
	if _, err := normalizeNotificationCommand(tooMany); !errors.Is(err, myerrors.ErrNotificationRecipientLimit) {
		t.Fatalf("recipient limit error=%v", err)
	}
	tooLarge := valid
	tooLarge.Content = strings.Repeat("界", notificationContentRunes+1)
	if _, err := normalizeNotificationCommand(tooLarge); !errors.Is(err, myerrors.ErrNotificationPayloadTooLarge) {
		t.Fatalf("payload limit error=%v", err)
	}
}

func newNotificationTestSubject(t *testing.T) (*NotificationService, *gorm.DB, *notificationAuditWriter) {
	t.Helper()
	db := testutil.OpenSQLite(t,
		&model.Notification{}, &model.NotificationRecipient{}, &model.SysUser{},
		&model.SysMenu{}, &model.SysRole{}, &model.SysRoleMenu{}, &model.SysUserRole{},
	)
	if err := db.Exec(`CREATE UNIQUE INDEX ux_notification_source_dedup
		ON notification (source_module, dedup_key) WHERE dedup_key IS NOT NULL AND dedup_key <> ''`).Error; err != nil {
		t.Fatalf("create notification dedup index: %v", err)
	}
	for _, userId := range []int{101, 102, 103} {
		testutil.MustCreate(t, db, &model.SysUser{
			Basic: model.Basic{Id: userId, State: true}, UserName: "user-" + string(rune(userId)),
		})
	}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	writer := &notificationAuditWriter{}
	repository := impl.NewNotificationRepositoryImpl(&database.PrimaryDB{DB: db})
	return NewNotificationService(repository, sf, writer), db, writer
}
