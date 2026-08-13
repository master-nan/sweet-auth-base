/**
 * @Author: Nan
 * @Date: 2024/5/21 下午2:27
 */

package service

import (
	"backend/dto/request"
	"backend/internal/cache"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"
)

type SysConfigureService struct {
	sysConfigureRepo  repository.SysConfigureRepository
	sysConfigureCache *cache.SysConfigureCache
}

func NewSysConfigureService(sysConfigureRepo repository.SysConfigureRepository, sysConfigureCache *cache.SysConfigureCache) *SysConfigureService {
	return &SysConfigureService{
		sysConfigureRepo,
		sysConfigureCache,
	}
}

func (cs *SysConfigureService) Query() (model.SysConfigure, error) {
	var data model.SysConfigure
	data, err := cs.sysConfigureCache.Get("")
	if err != nil {
		repoData, e := cs.sysConfigureRepo.GetSysConfigure()
		if e != nil {
			return data, e
		}
		data = repoData
		if errors.Is(err, cache.ErrCacheMiss) {
			err = cs.sysConfigureCache.Set("", data)
			if err != nil {
				zap.L().Error("Failed to cache sysConfigure set: ", zap.Error(err))
			}
		}
	}
	return data, err
}

func (cs *SysConfigureService) Update(ctx context.Context, req request.ConfigureUpdateReq) error {
	tx := cs.sysConfigureRepo.DBWithContext(ctx)
	if strings.TrimSpace(req.SenderPassword) == "" {
		var existing model.SysConfigure
		if err := tx.First(&existing, req.Id).Error; err != nil {
			return err
		}
		req.SenderPassword = existing.SenderPassword
	}
	if err := validateConfigureUpdate(req); err != nil {
		return err
	}
	err := cs.sysConfigureRepo.Update(tx, &req, req.Id)
	if err != nil {
		zap.L().Error("Failed to sysConfigure update: ", zap.Error(err))
		return err
	}
	err = cs.sysConfigureCache.Delete("")
	if err != nil {
		zap.L().Error("Failed to cache sysConfigure delete: ", zap.Error(err))
		return err
	}
	return nil
}

func validateConfigureUpdate(req request.ConfigureUpdateReq) error {
	if req.PasswordLength < 6 {
		return myerrors.NewBadRequestError("密码长度不能小于6位")
	}
	if req.PasswordComplexity < 1 || req.PasswordComplexity > 3 {
		return myerrors.NewBadRequestError("密码复杂度不正确")
	}
	if req.PasswordExpireTime < 0 {
		return myerrors.NewBadRequestError("密码过期时间不能为负数")
	}
	if req.PasswordErrorCount <= 0 {
		return myerrors.NewBadRequestError("密码错误次数必须大于0")
	}
	if req.PasswordLockMinutes <= 0 {
		return myerrors.NewBadRequestError("密码错误锁定时长必须大于0")
	}
	if strings.TrimSpace(req.PasswordPolicy) == "" {
		return myerrors.NewBadRequestError("密码策略不能为空")
	}
	switch strings.ToLower(strings.TrimSpace(req.PasswordPolicy)) {
	case "low", "medium", "strong", "high", "custom":
	default:
		return myerrors.NewBadRequestError("密码策略不正确")
	}
	if strings.TrimSpace(req.SystemName) == "" {
		return myerrors.NewBadRequestError("系统名称不能为空")
	}
	if req.EnableEmail != nil && *req.EnableEmail {
		if strings.TrimSpace(req.SmtpServer) == "" || req.SmtpPort <= 0 || strings.TrimSpace(req.SenderEmail) == "" || strings.TrimSpace(req.SenderPassword) == "" {
			return myerrors.NewBadRequestError("启用邮件服务时，请完整配置 SMTP 服务器、端口、发件邮箱和授权码")
		}
	}
	return nil
}

func (cs *SysConfigureService) SendTestEmail(to string) error {
	cfg, err := cs.Query()
	if err != nil {
		return err
	}
	subject := "Sweet Admin 测试邮件"
	body := "这是一封来自 Sweet Admin 的测试邮件。收到此邮件说明当前 SMTP 配置可用。"
	return SendEmailWithConfigure(cfg, to, subject, body)
}

func SendEmailWithConfigure(cfg model.SysConfigure, to, subject, body string) error {
	recipient := strings.TrimSpace(to)
	if !cfg.EnableEmail {
		return myerrors.NewBadRequestError("邮件服务未启用")
	}
	if recipient == "" {
		return myerrors.NewBadRequestError("收件人邮箱不能为空")
	}
	host := strings.TrimSpace(cfg.SmtpServer)
	from := strings.TrimSpace(cfg.SenderEmail)
	password := strings.TrimSpace(cfg.SenderPassword)
	if host == "" || cfg.SmtpPort <= 0 || from == "" || password == "" {
		return myerrors.NewBadRequestError("邮件服务配置不完整")
	}

	message := buildEmailMessage(from, recipient, subject, body)
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", cfg.SmtpPort))
	auth := smtp.PlainAuth("", from, password, host)
	if cfg.SmtpPort == 465 {
		return sendMailWithImplicitTLS(addr, host, auth, from, []string{recipient}, []byte(message))
	}
	return smtp.SendMail(addr, auth, from, []string{recipient}, []byte(message))
}

func buildEmailMessage(from, to, subject, body string) string {
	headers := []struct {
		key   string
		value string
	}{
		{key: "From", value: from},
		{key: "To", value: to},
		{key: "Subject", value: mime.QEncoding.Encode("UTF-8", subject)},
		{key: "MIME-Version", value: "1.0"},
		{key: "Content-Type", value: `text/plain; charset="UTF-8"`},
	}
	var builder strings.Builder
	for _, header := range headers {
		builder.WriteString(header.key)
		builder.WriteString(": ")
		builder.WriteString(header.value)
		builder.WriteString("\r\n")
	}
	builder.WriteString("\r\n")
	builder.WriteString(body)
	return builder.String()
}

func sendMailWithImplicitTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: host,
	})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("AUTH"); ok {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
