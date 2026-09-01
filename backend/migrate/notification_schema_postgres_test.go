package main

import (
	"backend/internal/audit"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	servicepkg "backend/service"
	"context"
	"errors"
	"sync"
	"testing"

	"gorm.io/gorm"
)

type notificationPostgresAuditWriter struct{}

func (notificationPostgresAuditWriter) RecordTransactionalAuditContext(
	context.Context,
	*gorm.DB,
	servicepkg.TransactionalAuditRecord,
) error {
	return nil
}

func TestNotificationPostgreSQLSchemaAndRuntimeContract(t *testing.T) {
	db := openMigrationLedgerPostgreSQL(t)
	createNotificationPostgresRuntimeFixtures(t, db)
	if err := db.Exec(`INSERT INTO sys_user (id, state)
		SELECT value, true FROM generate_series(1, 1000) AS value`).Error; err != nil {
		t.Fatalf("create user fixtures: %v", err)
	}
	if err := migrateNotificationCenterSchema(db); err != nil {
		t.Fatalf("migrate notification schema: %v", err)
	}
	if err := migrateNotificationStandardBaseFields(db); err != nil {
		t.Fatalf("migrate notification standard fields: %v", err)
	}
	if err := migrateNotificationCenterSchema(db); err != nil {
		t.Fatalf("repeat notification migration: %v", err)
	}
	if err := migrateNotificationStandardBaseFields(db); err != nil {
		t.Fatalf("repeat notification standard fields: %v", err)
	}

	assertNotificationPostgresChecks(t, db)
	assertNotificationPostgresIndexes(t, db)

	sf, err := utils.NewSnowflake(2)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	repository := impl.NewNotificationRepositoryImpl(&database.PrimaryDB{DB: db})
	service := servicepkg.NewNotificationService(repository, sf, notificationPostgresAuditWriter{})
	ctx := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(1, "postgres-user"))
	assertNotificationPostgresActionContract(t, db, service, ctx)
	recipients := make([]int, 1000)
	for index := range recipients {
		recipients[index] = index + 1
	}
	bulk, err := service.Send(ctx, servicepkg.NotificationCommand{
		Recipients: recipients, Category: model.NotificationCategoryBusiness,
		Level: model.NotificationLevelInfo, Title: "批量学习计划", Content: "一千名用户的同步投递边界。",
		SourceModule: "learning", SourceType: "learning_plan", SourceId: "pg-1000",
		DedupKey: "learning-plan-pg-1000-published",
	})
	if err != nil || bulk.CreatedRecipientCount != 1000 {
		t.Fatalf("send 1000 recipients=%+v err=%v", bulk, err)
	}
	if count, err := repository.UnreadCount(context.Background(), 1); err != nil || count != 2 {
		t.Fatalf("unread count=%d err=%v", count, err)
	}

	readAt := model.CustomTime(model.Now())
	changed, err := repository.MarkRead(context.Background(), 1, bulk.NotificationId, readAt)
	if err != nil || !changed {
		t.Fatalf("first mark read changed=%t err=%v", changed, err)
	}
	first, err := repository.Detail(context.Background(), 1, bulk.NotificationId)
	if err != nil || first.ReadAt == nil {
		t.Fatalf("first read detail=%+v err=%v", first, err)
	}
	changed, err = repository.MarkRead(context.Background(), 1, bulk.NotificationId, model.CustomTime(model.Now()))
	if err != nil || changed {
		t.Fatalf("repeated mark read changed=%t err=%v", changed, err)
	}
	second, err := repository.Detail(context.Background(), 1, bulk.NotificationId)
	if err != nil || second.ReadAt == nil || second.ReadAt.String() != first.ReadAt.String() {
		t.Fatalf("read_at changed: first=%v second=%v err=%v", first.ReadAt, second.ReadAt, err)
	}
	updated, err := repository.MarkAllRead(context.Background(), 2, model.CustomTime(model.Now()))
	if err != nil || updated != 2 {
		t.Fatalf("mark all user 2 updated=%d err=%v", updated, err)
	}
	if count, err := repository.UnreadCount(context.Background(), 3); err != nil || count != 1 {
		t.Fatalf("mark all crossed user boundary: count=%d err=%v", count, err)
	}

	assertNotificationConcurrentDedup(t, db, service, ctx)
	assertNotificationConcurrentRecipientConflict(t, db, service, ctx)
	isolated, err := service.Send(ctx, servicepkg.NotificationCommand{
		Recipients: []int{1}, Category: model.NotificationCategorySecurity,
		Level: model.NotificationLevelWarning, Title: "账号安全提醒", Content: "请检查最近登录记录。",
		SourceModule: "auth", SourceType: "login", SourceId: "1", DedupKey: "auth-login-1",
	})
	if err != nil {
		t.Fatalf("send isolated notification: %v", err)
	}
	ctxOther := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(2, "other"))
	if _, err := service.Detail(ctxOther, isolated.NotificationId); !errors.Is(err, myerrors.ErrNotificationNotVisible) {
		t.Fatalf("IDOR detail error=%v", err)
	}
	if _, err := service.MarkRead(ctxOther, isolated.NotificationId); !errors.Is(err, myerrors.ErrNotificationNotVisible) {
		t.Fatalf("IDOR mark read error=%v", err)
	}
}

func createNotificationPostgresRuntimeFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`CREATE TABLE sys_user (
		id bigint PRIMARY KEY,
		state boolean NOT NULL DEFAULT true,
		gmt_delete timestamptz
	)`).Error; err != nil {
		t.Fatalf("create user fixture table: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE sys_menu (id bigint PRIMARY KEY, pid bigint NOT NULL DEFAULT 0, name varchar(32) NOT NULL, path varchar(128) NOT NULL, state boolean NOT NULL DEFAULT true, gmt_delete timestamptz)`,
		`CREATE TABLE sys_role (id bigint PRIMARY KEY, state boolean NOT NULL DEFAULT true, gmt_delete timestamptz)`,
		`CREATE TABLE sys_role_menu (role_id bigint NOT NULL, menu_id bigint NOT NULL)`,
		`CREATE TABLE sys_user_role (user_id bigint NOT NULL, role_id bigint NOT NULL)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create notification runtime fixture table: %v", err)
		}
	}
}

func assertNotificationPostgresActionContract(
	t *testing.T,
	db *gorm.DB,
	service *servicepkg.NotificationService,
	ctx context.Context,
) {
	t.Helper()
	if err := db.Exec(`INSERT INTO sys_menu (id, pid, name, path) VALUES
		(200, 0, 'integration', 'integration'),
		(201, 200, 'integration_execution', 'execution'),
		(202, 200, 'integration_credential', 'credential')`).Error; err != nil {
		t.Fatalf("seed notification action menus: %v", err)
	}
	if err := db.Exec(`INSERT INTO sys_role (id) VALUES (301);
		INSERT INTO sys_role_menu (role_id, menu_id) VALUES (301, 201);
		INSERT INTO sys_user_role (user_id, role_id) VALUES (1, 301)`).Error; err != nil {
		t.Fatalf("seed notification action permission: %v", err)
	}
	command := servicepkg.NotificationCommand{
		Recipients: []int{1, 2}, Category: model.NotificationCategoryIntegration,
		Level: model.NotificationLevelWarning, Title: "执行失败", Content: "请检查执行日志。",
		SourceModule: "integration", SourceType: "execution", SourceId: "7001",
		ActionMenuName: "integration_execution", ActionPath: "/admin/integration/execution",
		DedupKey: "pg-action-contract",
	}
	result, err := service.Send(ctx, command)
	if err != nil {
		t.Fatalf("send valid action notification: %v", err)
	}
	authorized, err := service.Detail(ctx, result.NotificationId)
	if err != nil || authorized.Action == nil || !authorized.Action.Available || authorized.Action.Path != command.ActionPath {
		t.Fatalf("authorized action=%+v err=%v", authorized.Action, err)
	}
	ctxWithoutMenu := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(2, "without-menu"))
	unauthorized, err := service.Detail(ctxWithoutMenu, result.NotificationId)
	if err != nil || unauthorized.Action == nil || unauthorized.Action.Available || unauthorized.Action.Path != "" {
		t.Fatalf("unauthorized action=%+v err=%v", unauthorized.Action, err)
	}

	for _, invalid := range []servicepkg.NotificationCommand{
		func() servicepkg.NotificationCommand {
			value := command
			value.ActionMenuName = "missing_menu"
			value.DedupKey = ""
			return value
		}(),
		func() servicepkg.NotificationCommand {
			value := command
			value.ActionPath = "/admin/integration/credential"
			value.DedupKey = ""
			return value
		}(),
		func() servicepkg.NotificationCommand {
			value := command
			value.ActionPath = "https://example.com/execution"
			value.DedupKey = ""
			return value
		}(),
		func() servicepkg.NotificationCommand {
			value := command
			value.ActionPath += "?source=notification"
			value.DedupKey = ""
			return value
		}(),
		func() servicepkg.NotificationCommand {
			value := command
			value.ActionPath += "#detail"
			value.DedupKey = ""
			return value
		}(),
	} {
		if _, err := service.Send(ctx, invalid); !errors.Is(err, myerrors.ErrNotificationInvalidAction) {
			t.Fatalf("invalid action command=%+v error=%v", invalid, err)
		}
	}
}

func assertNotificationPostgresChecks(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, field := range []string{
		"gmt_create", "create_user", "create_name", "gmt_modify", "modify_user", "modify_name",
		"gmt_delete", "delete_user", "delete_name", "state",
	} {
		if !db.Migrator().HasColumn("notification", field) {
			t.Fatalf("notification missing standard field %s", field)
		}
	}
	if db.Migrator().HasColumn("notification", "created_at") {
		t.Fatal("notification retained legacy created_at column")
	}
	base := `INSERT INTO notification (
		id, category, level, title, content, source_module, source_type
	) VALUES (?, ?, ?, ?, ?, ?, ?)`
	if err := db.Exec(base, 8001, "UNKNOWN", "INFO", "标题", "内容", "test", "object").Error; err == nil {
		t.Fatal("category CHECK accepted an unknown value")
	}
	if err := db.Exec(base, 8002, "SYSTEM", "DEBUG", "标题", "内容", "test", "object").Error; err == nil {
		t.Fatal("level CHECK accepted an unknown value")
	}
	if err := db.Exec(base, 8003, "SYSTEM", "INFO", "标题", "内容", "test", "object").Error; err != nil {
		t.Fatalf("insert canonical notification: %v", err)
	}
	if err := db.Exec(`INSERT INTO notification_recipient (notification_id, user_id) VALUES (8003, 999999)`).Error; err == nil {
		t.Fatal("recipient user FK accepted an unknown user")
	}
	if err := db.Exec(`INSERT INTO notification_recipient (notification_id, user_id) VALUES (999999, 1)`).Error; err == nil {
		t.Fatal("recipient notification FK accepted an unknown notification")
	}
	if err := db.Exec(`INSERT INTO notification (
		id, category, level, title, content, source_module, source_type, dedup_key
	) VALUES (8010, 'SYSTEM', 'INFO', 'A', 'A', 'system', 'test', 'same')`).Error; err != nil {
		t.Fatalf("insert dedup fixture: %v", err)
	}
	if err := db.Exec(`INSERT INTO notification (
		id, category, level, title, content, source_module, source_type, dedup_key
	) VALUES (8011, 'SYSTEM', 'INFO', 'B', 'B', 'system', 'test', 'same')`).Error; err == nil {
		t.Fatal("partial unique index accepted duplicate source_module/dedup_key")
	}
}

func assertNotificationPostgresIndexes(t *testing.T, db *gorm.DB) {
	t.Helper()
	var names []string
	if err := db.Raw(`SELECT indexname FROM pg_indexes
		WHERE schemaname = current_schema() AND tablename IN ('notification', 'notification_recipient')`).
		Scan(&names).Error; err != nil {
		t.Fatalf("query notification indexes: %v", err)
	}
	for _, expected := range []string{
		"ux_notification_source_dedup",
		"idx_notification_created",
		"idx_notification_recipient_user_created",
		"idx_notification_recipient_user_unread",
	} {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing notification index %s; indexes=%v", expected, names)
		}
	}
}

func assertNotificationConcurrentDedup(
	t *testing.T,
	db *gorm.DB,
	service *servicepkg.NotificationService,
	ctx context.Context,
) {
	t.Helper()
	command := servicepkg.NotificationCommand{
		Recipients: []int{1, 2}, Category: model.NotificationCategoryReminder,
		Level: model.NotificationLevelInfo, Title: "考试即将开始", Content: "请进入考试页面。",
		SourceModule: "exam", SourceType: "exam", SourceId: "42", DedupKey: "exam-42-start",
	}
	start := make(chan struct{})
	results := make(chan servicepkg.NotificationSendResult, 2)
	errorsChannel := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			result, err := service.Send(ctx, command)
			if err != nil {
				errorsChannel <- err
				return
			}
			results <- result
		}()
	}
	close(start)
	group.Wait()
	close(results)
	close(errorsChannel)
	for err := range errorsChannel {
		t.Fatalf("concurrent dedup send: %v", err)
	}
	ids := make(map[int]struct{})
	count := 0
	for result := range results {
		ids[result.NotificationId] = struct{}{}
		count++
	}
	if count != 2 || len(ids) != 1 {
		t.Fatalf("concurrent dedup results=%d unique ids=%v", count, ids)
	}
	var notification model.Notification
	if err := db.Where("source_module = ? AND dedup_key = ?", command.SourceModule, command.DedupKey).
		First(&notification).Error; err != nil {
		t.Fatalf("load concurrent notification: %v", err)
	}
	var recipientCount int64
	if err := db.Model(&model.NotificationRecipient{}).
		Where("notification_id = ?", notification.Id).Count(&recipientCount).Error; err != nil || recipientCount != 2 {
		t.Fatalf("concurrent recipient count=%d err=%v", recipientCount, err)
	}
}

func assertNotificationConcurrentRecipientConflict(
	t *testing.T,
	db *gorm.DB,
	service *servicepkg.NotificationService,
	ctx context.Context,
) {
	t.Helper()
	base := servicepkg.NotificationCommand{
		Recipients: []int{1}, Category: model.NotificationCategoryReminder,
		Level: model.NotificationLevelInfo, Title: "补考通知", Content: "请按时参加补考。",
		SourceModule: "exam", SourceType: "exam", SourceId: "43", DedupKey: "exam-43-resit",
	}
	commands := []servicepkg.NotificationCommand{base, base}
	commands[1].Recipients = []int{2}
	start := make(chan struct{})
	errorsChannel := make(chan error, len(commands))
	var group sync.WaitGroup
	for _, command := range commands {
		command := command
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := service.Send(ctx, command)
			errorsChannel <- err
		}()
	}
	close(start)
	group.Wait()
	close(errorsChannel)
	successCount, conflictCount := 0, 0
	for err := range errorsChannel {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, myerrors.ErrNotificationDedupConflict):
			conflictCount++
		default:
			t.Fatalf("unexpected concurrent recipient error: %v", err)
		}
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("concurrent recipient results: success=%d conflict=%d", successCount, conflictCount)
	}
	var notification model.Notification
	if err := db.Where("source_module = ? AND dedup_key = ?", base.SourceModule, base.DedupKey).
		First(&notification).Error; err != nil {
		t.Fatalf("load recipient conflict notification: %v", err)
	}
	var userIds []int
	if err := db.Model(&model.NotificationRecipient{}).
		Where("notification_id = ?", notification.Id).Order("user_id ASC").Pluck("user_id", &userIds).Error; err != nil {
		t.Fatalf("load recipient conflict set: %v", err)
	}
	if len(userIds) != 1 || (userIds[0] != 1 && userIds[0] != 2) {
		t.Fatalf("concurrent recipient sets were merged: %v", userIds)
	}
}
