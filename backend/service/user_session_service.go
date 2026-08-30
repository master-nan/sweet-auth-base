package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/cache"
	"backend/internal/database"
	appErrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"context"
	"strings"
	"time"
)

const userSessionOnlineWindow = 2 * time.Minute

type UserSessionClient struct {
	IPAddress string
	UserAgent string
	Channel   string
}

// UserSessionService 保存设备登录记录，并通过 Redis 撤销仍在使用的会话。
type UserSessionService struct {
	db     *database.PrimaryDB
	sf     *utils.Snowflake
	tokens *AuthTokenService
	now    func() time.Time
}

func NewUserSessionService(db *database.PrimaryDB, sf *utils.Snowflake, tokens *AuthTokenService) *UserSessionService {
	return &UserSessionService{db: db, sf: sf, tokens: tokens, now: time.Now}
}

func (s *UserSessionService) Open(ctx context.Context, userID int, sessionID string, loginAt, expiresAt time.Time, client UserSessionClient) error {
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return appErrors.WrapSystemError(err)
	}
	deviceType, browser, operatingSystem := describeUserAgent(client.UserAgent)
	record := model.SysUserSession{
		Basic:           model.Basic{Id: int(id), State: true},
		SessionKeyHash:  cache.SessionFingerprint(sessionID),
		UserID:          userID,
		Status:          model.UserSessionStatusActive,
		LoginAt:         model.CustomTime(loginAt.UTC()),
		LastSeenAt:      model.CustomTime(loginAt.UTC()),
		ExpiresAt:       model.CustomTime(expiresAt.UTC()),
		LoginChannel:    strings.TrimSpace(client.Channel),
		IPAddress:       strings.TrimSpace(client.IPAddress),
		UserAgent:       strings.TrimSpace(client.UserAgent),
		DeviceType:      deviceType,
		Browser:         browser,
		OperatingSystem: operatingSystem,
	}
	if err := s.db.DB.WithContext(ctx).Create(&record).Error; err != nil {
		return appErrors.WrapDatabaseError(err)
	}
	return nil
}

// Touch 在刷新 Token 时延长同一个设备会话；旧版本遗留会话没有记录时会补建一条。
func (s *UserSessionService) Touch(ctx context.Context, userID int, sessionID string, seenAt, expiresAt time.Time, client UserSessionClient) error {
	digest := cache.SessionFingerprint(sessionID)
	deviceType, browser, operatingSystem := describeUserAgent(client.UserAgent)
	updates := map[string]any{
		"last_seen_at":  model.CustomTime(seenAt.UTC()),
		"expires_at":    model.CustomTime(expiresAt.UTC()),
		"status":        model.UserSessionStatusActive,
		"logout_at":     nil,
		"logout_reason": "",
	}
	if value := strings.TrimSpace(client.IPAddress); value != "" {
		updates["ip_address"] = value
	}
	if value := strings.TrimSpace(client.UserAgent); value != "" {
		updates["user_agent"] = value
		updates["device_type"] = deviceType
		updates["browser"] = browser
		updates["operating_system"] = operatingSystem
	}
	result := s.db.DB.WithContext(ctx).Model(&model.SysUserSession{}).
		Where("user_id = ? AND session_key_hash = ?", userID, digest).
		Updates(updates)
	if result.Error != nil {
		return appErrors.WrapDatabaseError(result.Error)
	}
	if result.RowsAffected > 0 {
		return nil
	}
	return s.Open(ctx, userID, sessionID, seenAt, expiresAt, client)
}

func (s *UserSessionService) Heartbeat(ctx context.Context, userID int, sessionID string) error {
	result := s.db.DB.WithContext(ctx).Model(&model.SysUserSession{}).
		Where("user_id = ? AND session_key_hash = ? AND status = ?", userID, cache.SessionFingerprint(sessionID), model.UserSessionStatusActive).
		Update("last_seen_at", model.CustomTime(s.now().UTC()))
	if result.Error != nil {
		return appErrors.WrapDatabaseError(result.Error)
	}
	if result.RowsAffected == 0 {
		return appErrors.ErrUserNotLogin
	}
	return nil
}

func (s *UserSessionService) Close(ctx context.Context, userID int, sessionID, status, reason string) error {
	return s.closeDigest(ctx, userID, cache.SessionFingerprint(sessionID), status, reason)
}

func (s *UserSessionService) RevokeSession(ctx context.Context, id int, reason string) error {
	var record model.SysUserSession
	if err := s.db.DB.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return appErrors.WrapDatabaseError(err)
	}
	if err := s.tokens.RevokeSessionDigest(record.UserID, record.SessionKeyHash); err != nil {
		return err
	}
	return s.closeDigest(ctx, record.UserID, record.SessionKeyHash, model.UserSessionStatusForcedOffline, reason)
}

func (s *UserSessionService) RevokeUser(ctx context.Context, userID int, status, reason string) error {
	var records []model.SysUserSession
	if err := s.db.DB.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, model.UserSessionStatusActive).
		Find(&records).Error; err != nil {
		return appErrors.WrapDatabaseError(err)
	}
	for _, record := range records {
		if err := s.tokens.RevokeSessionDigest(userID, record.SessionKeyHash); err != nil {
			return err
		}
	}
	now := model.CustomTime(s.now().UTC())
	if err := s.db.DB.WithContext(ctx).Model(&model.SysUserSession{}).
		Where("user_id = ? AND status = ?", userID, model.UserSessionStatusActive).
		Updates(map[string]any{"status": status, "logout_reason": reason, "logout_at": &now}).Error; err != nil {
		return appErrors.WrapDatabaseError(err)
	}
	if err := s.db.DB.WithContext(ctx).Model(&model.SysUser{}).Where("id = ?", userID).Update("access_tokens", "").Error; err != nil {
		return appErrors.WrapDatabaseError(err)
	}
	return nil
}

func (s *UserSessionService) IsActive(userID int, sessionID string) (bool, error) {
	return s.tokens.IsSessionActive(userID, sessionID)
}

func (s *UserSessionService) ClosureReason(ctx context.Context, userID int, sessionID string) string {
	var record model.SysUserSession
	err := s.db.DB.WithContext(ctx).
		Select("status", "logout_reason").
		Where("user_id = ? AND session_key_hash = ?", userID, cache.SessionFingerprint(sessionID)).
		First(&record).Error
	if err != nil {
		return "登录会话已失效"
	}
	if strings.TrimSpace(record.LogoutReason) != "" {
		return record.LogoutReason
	}
	switch record.Status {
	case model.UserSessionStatusPasswordChanged:
		return "密码已修改，请重新登录"
	case model.UserSessionStatusAccountDisabled:
		return "账号已停用"
	case model.UserSessionStatusAccountDeleted:
		return "账号已删除"
	case model.UserSessionStatusForcedOffline:
		return "管理员已将当前设备下线"
	default:
		return "登录会话已失效"
	}
}

func (s *UserSessionService) Query(ctx context.Context, req request.UserSessionQueryReq, currentSessionID string) (response.UserSessionListRes, error) {
	page, num := req.Page, req.Num
	if page < 1 {
		page = 1
	}
	if num < 1 || num > 200 {
		num = 20
	}
	now := s.now().UTC()
	onlineSince := now.Add(-userSessionOnlineWindow)
	db := s.db.DB.WithContext(ctx)
	sessionTable := db.NamingStrategy.TableName("SysUserSession")
	userTable := db.NamingStrategy.TableName("SysUser")
	base := db.Table(sessionTable + " AS session").
		Joins("JOIN " + userTable + " AS account ON account.id = session.user_id AND account.gmt_delete IS NULL").
		Where("session.gmt_delete IS NULL")
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	if keyword != "" {
		base = base.Where("LOWER(account.user_name) LIKE ? OR LOWER(session.ip_address) LIKE ? OR LOWER(session.browser) LIKE ? OR LOWER(session.operating_system) LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	switch strings.ToLower(strings.TrimSpace(req.Status)) {
	case "online", "":
		base = base.Where("session.status = ? AND session.last_seen_at >= ? AND session.expires_at > ?", model.UserSessionStatusActive, onlineSince, now)
	case "active":
		base = base.Where("session.status = ? AND session.expires_at > ?", model.UserSessionStatusActive, now)
	case "closed":
		base = base.Where("session.status <> ? OR session.expires_at <= ?", model.UserSessionStatusActive, now)
	case "all":
	default:
		return response.UserSessionListRes{}, appErrors.ErrParamInvalid
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return response.UserSessionListRes{}, appErrors.WrapDatabaseError(err)
	}
	type sessionRow struct {
		model.SysUserSession
		UserName string `gorm:"column:user_name"`
	}
	var rows []sessionRow
	selectFields := "session.*, account.user_name AS user_name"
	if err := base.Select(selectFields).
		Order("session.last_seen_at DESC, session.id DESC").
		Offset((page - 1) * num).Limit(num).Scan(&rows).Error; err != nil {
		return response.UserSessionListRes{}, appErrors.WrapDatabaseError(err)
	}
	currentDigest := ""
	if currentSessionID != "" {
		currentDigest = cache.SessionFingerprint(currentSessionID)
	}
	items := make([]response.UserSessionRes, 0, len(rows))
	for _, row := range rows {
		lastSeen := time.Time(row.LastSeenAt)
		expiresAt := time.Time(row.ExpiresAt)
		online := row.Status == model.UserSessionStatusActive && !lastSeen.Before(onlineSince) && expiresAt.After(now)
		status := row.Status
		if status == model.UserSessionStatusActive && !expiresAt.After(now) {
			status = model.UserSessionStatusExpired
		}
		items = append(items, response.UserSessionRes{
			ID: row.Id, UserID: row.UserID, UserName: row.UserName, Status: status,
			Online: online, Current: currentDigest != "" && row.SessionKeyHash == currentDigest,
			LoginAt: row.LoginAt, LastSeenAt: row.LastSeenAt, ExpiresAt: row.ExpiresAt,
			LogoutAt: row.LogoutAt, LogoutReason: row.LogoutReason, LoginChannel: row.LoginChannel,
			IPAddress: row.IPAddress, DeviceType: row.DeviceType, Browser: row.Browser, OperatingSystem: row.OperatingSystem,
		})
	}
	onlineBase := db.Model(&model.SysUserSession{}).
		Where("status = ? AND last_seen_at >= ? AND expires_at > ?", model.UserSessionStatusActive, onlineSince, now)
	var onlineDevices int64
	if err := onlineBase.Count(&onlineDevices).Error; err != nil {
		return response.UserSessionListRes{}, appErrors.WrapDatabaseError(err)
	}
	var onlineUsers int64
	if err := db.Model(&model.SysUserSession{}).
		Where("status = ? AND last_seen_at >= ? AND expires_at > ?", model.UserSessionStatusActive, onlineSince, now).
		Distinct("user_id").Count(&onlineUsers).Error; err != nil {
		return response.UserSessionListRes{}, appErrors.WrapDatabaseError(err)
	}
	return response.UserSessionListRes{Items: items, Total: int(total), OnlineUsers: int(onlineUsers), OnlineDevices: int(onlineDevices)}, nil
}

func (s *UserSessionService) closeDigest(ctx context.Context, userID int, digest, status, reason string) error {
	now := model.CustomTime(s.now().UTC())
	result := s.db.DB.WithContext(ctx).Model(&model.SysUserSession{}).
		Where("user_id = ? AND session_key_hash = ?", userID, digest).
		Updates(map[string]any{"status": status, "logout_reason": strings.TrimSpace(reason), "logout_at": &now})
	if result.Error != nil {
		return appErrors.WrapDatabaseError(result.Error)
	}
	return nil
}

func describeUserAgent(value string) (deviceType, browser, operatingSystem string) {
	ua := strings.ToLower(value)
	deviceType = "桌面设备"
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		deviceType = "移动设备"
	}
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "chrome/") || strings.Contains(ua, "crios/"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox/") || strings.Contains(ua, "fxios/"):
		browser = "Firefox"
	case strings.Contains(ua, "safari/"):
		browser = "Safari"
	default:
		browser = "未知浏览器"
	}
	switch {
	case strings.Contains(ua, "windows"):
		operatingSystem = "Windows"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		operatingSystem = "iOS"
	case strings.Contains(ua, "android"):
		operatingSystem = "Android"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		operatingSystem = "macOS"
	case strings.Contains(ua, "linux"):
		operatingSystem = "Linux"
	default:
		operatingSystem = "未知系统"
	}
	return deviceType, browser, operatingSystem
}
