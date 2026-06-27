package service

import (
	"backend/dto/request"
	"backend/internal/cache"
	"backend/internal/database"
	"backend/model"
	"backend/repository/impl"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func TestSysConfigureQueryReturnsRepositoryDataOnCacheMiss(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysConfigure{}); err != nil {
		t.Fatalf("migrate sys configure: %v", err)
	}
	if err := db.Create(&model.SysConfigure{
		Basic:               model.Basic{Id: 1, State: true},
		EnableCaptcha:       true,
		PasswordLength:      12,
		PasswordComplexity:  3,
		PasswordExpireTime:  45,
		PasswordErrorCount:  2,
		PasswordLockMinutes: 12,
		PasswordPolicy:      "strong",
		SystemName:          "Runtime Admin",
		SystemVersion:       "2.1",
		SystemDescription:   "runtime config",
		SmtpPort:            587,
	}).Error; err != nil {
		t.Fatalf("seed sys configure: %v", err)
	}

	repo := impl.NewSysConfigureRepositoryImpl(&database.PrimaryDB{DB: db})
	cacheStore := &missThenStoreCache{}
	svc := NewSysConfigureService(repo, cache.NewSysConfigureCache(cacheStore))

	cfg, err := svc.Query()
	if err != nil {
		t.Fatalf("query sys configure: %v", err)
	}
	if cfg.PasswordLength != 12 || cfg.PasswordLockMinutes != 12 || cfg.PasswordPolicy != "strong" || cfg.SystemName != "Runtime Admin" {
		t.Fatalf("expected repository config after cache miss, got %+v", cfg)
	}
	if cacheStore.stored.PasswordLength != 12 || cacheStore.stored.SystemName != "Runtime Admin" {
		t.Fatalf("expected repository config to be cached, got %+v", cacheStore.stored)
	}
}

func TestSendEmailWithConfigureRejectsDisabledEmail(t *testing.T) {
	err := SendEmailWithConfigure(model.SysConfigure{
		EnableEmail: false,
		SmtpServer:  "smtp.example.com",
		SmtpPort:    587,
		SenderEmail: "admin@example.com",
	}, "user@example.com", "测试", "body")

	if err == nil || !strings.Contains(err.Error(), "邮件服务未启用") {
		t.Fatalf("expected disabled email error, got %v", err)
	}
}

func TestBuildEmailMessageUsesUTF8SubjectAndBody(t *testing.T) {
	message := buildEmailMessage("admin@example.com", "user@example.com", "测试邮件", "邮件内容")

	if !strings.Contains(message, "From: admin@example.com\r\n") {
		t.Fatalf("expected From header, got %q", message)
	}
	if !strings.Contains(message, "To: user@example.com\r\n") {
		t.Fatalf("expected To header, got %q", message)
	}
	if !strings.Contains(message, "Subject: =?UTF-8?") {
		t.Fatalf("expected encoded UTF-8 subject, got %q", message)
	}
	if !strings.Contains(message, "\r\n\r\n邮件内容") {
		t.Fatalf("expected body after header separator, got %q", message)
	}
}

func TestValidateConfigureUpdateRequiresCompleteEmailSettingsWhenEnabled(t *testing.T) {
	enableEmail := true
	err := validateConfigureUpdate(request.ConfigureUpdateReq{
		EnableCaptcha:       boolPtr(false),
		PasswordLength:      8,
		PasswordComplexity:  2,
		PasswordExpireTime:  90,
		PasswordErrorCount:  5,
		PasswordLockMinutes: 15,
		PasswordPolicy:      "medium",
		SystemName:          "Sweet Admin",
		EnableEmail:         &enableEmail,
		SmtpPort:            465,
		SenderEmail:         "admin@example.com",
	})

	if err == nil || !strings.Contains(err.Error(), "启用邮件服务") {
		t.Fatalf("expected incomplete email settings error, got %v", err)
	}
}

func TestValidateConfigureUpdateAllowsKnownPasswordPolicies(t *testing.T) {
	enableEmail := false
	for _, policy := range []string{"low", "medium", "high", "custom", "strong"} {
		t.Run(policy, func(t *testing.T) {
			err := validateConfigureUpdate(request.ConfigureUpdateReq{
				EnableCaptcha:       boolPtr(false),
				PasswordLength:      8,
				PasswordComplexity:  2,
				PasswordExpireTime:  90,
				PasswordErrorCount:  5,
				PasswordLockMinutes: 15,
				PasswordPolicy:      policy,
				SystemName:          "Sweet Admin",
				EnableEmail:         &enableEmail,
			})
			if err != nil {
				t.Fatalf("expected policy %q to pass, got %v", policy, err)
			}
		})
	}

	err := validateConfigureUpdate(request.ConfigureUpdateReq{
		EnableCaptcha:       boolPtr(false),
		PasswordLength:      8,
		PasswordComplexity:  2,
		PasswordExpireTime:  90,
		PasswordErrorCount:  5,
		PasswordLockMinutes: 15,
		PasswordPolicy:      "unknown",
		SystemName:          "Sweet Admin",
		EnableEmail:         &enableEmail,
	})
	if err == nil || !strings.Contains(err.Error(), "密码策略不正确") {
		t.Fatalf("expected invalid policy error, got %v", err)
	}
}

func TestValidateConfigureUpdateRequiresPositiveLockMinutes(t *testing.T) {
	enableEmail := false
	err := validateConfigureUpdate(request.ConfigureUpdateReq{
		EnableCaptcha:       boolPtr(false),
		PasswordLength:      8,
		PasswordComplexity:  2,
		PasswordExpireTime:  90,
		PasswordErrorCount:  5,
		PasswordLockMinutes: 0,
		PasswordPolicy:      "medium",
		SystemName:          "Sweet Admin",
		EnableEmail:         &enableEmail,
	})

	if err == nil || !strings.Contains(err.Error(), "锁定时长") {
		t.Fatalf("expected lock minutes validation error, got %v", err)
	}
}

type missThenStoreCache struct {
	stored model.SysConfigure
}

func (c *missThenStoreCache) Get(_ string, _ interface{}) error {
	return cache.ErrCacheMiss
}

func (c *missThenStoreCache) Set(_ string, value interface{}, _ time.Duration) error {
	cfg, ok := value.(*model.SysConfigure)
	if !ok {
		return errors.Errorf("unexpected cache value type %T", value)
	}
	c.stored = *cfg
	return nil
}

func (c *missThenStoreCache) Del(_ string) error {
	return nil
}

func (c *missThenStoreCache) Exists(_ ...string) (int64, error) {
	return 0, nil
}

func (c *missThenStoreCache) Expire(_ string, _ time.Duration) (bool, error) {
	return true, nil
}
