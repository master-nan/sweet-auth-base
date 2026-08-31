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

	"gorm.io/gorm"
)

const userSessionOnlineWindow = 2 * time.Minute
const userSessionExportLimit = 10000

type UserSessionClient struct {
	IPAddress string
	UserAgent string
	Channel   string
}

type UserSessionClosure struct {
	Reason       string
	OperatorID   int
	OperatorName string
}

type userSessionRow struct {
	model.SysUserSession
	UserName    string `gorm:"column:user_name"`
	UserDeleted bool   `gorm:"column:user_deleted"`
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

func (s *UserSessionService) Open(ctx context.Context, userID int, userName, sessionID string, loginAt, expiresAt time.Time, client UserSessionClient) error {
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return appErrors.WrapSystemError(err)
	}
	deviceType, browser, operatingSystem := describeUserAgent(client.UserAgent)
	record := model.SysUserSession{
		Basic:            model.Basic{Id: int(id), State: true},
		SessionKeyHash:   cache.SessionFingerprint(sessionID),
		UserID:           userID,
		UserNameSnapshot: strings.TrimSpace(userName),
		Status:           model.UserSessionStatusActive,
		LoginAt:          model.CustomTime(loginAt.UTC()),
		LastSeenAt:       model.CustomTime(loginAt.UTC()),
		ExpiresAt:        model.CustomTime(expiresAt.UTC()),
		LoginChannel:     strings.TrimSpace(client.Channel),
		IPAddress:        strings.TrimSpace(client.IPAddress),
		UserAgent:        strings.TrimSpace(client.UserAgent),
		DeviceType:       deviceType,
		Browser:          browser,
		OperatingSystem:  operatingSystem,
	}
	if err := s.db.DB.WithContext(ctx).Create(&record).Error; err != nil {
		return appErrors.WrapDatabaseError(err)
	}
	return nil
}

// Touch 在刷新 Token 时延长同一个设备会话；旧版本遗留会话没有记录时会补建一条。
func (s *UserSessionService) Touch(ctx context.Context, userID int, userName, sessionID string, seenAt, expiresAt time.Time, client UserSessionClient) error {
	digest := cache.SessionFingerprint(sessionID)
	deviceType, browser, operatingSystem := describeUserAgent(client.UserAgent)
	updates := map[string]any{
		"last_seen_at":        model.CustomTime(seenAt.UTC()),
		"expires_at":          model.CustomTime(expiresAt.UTC()),
		"status":              model.UserSessionStatusActive,
		"logout_at":           nil,
		"logout_reason":       "",
		"closed_by_user_id":   0,
		"closed_by_user_name": "",
	}
	if value := strings.TrimSpace(userName); value != "" {
		updates["user_name_snapshot"] = value
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
	return s.Open(ctx, userID, userName, sessionID, seenAt, expiresAt, client)
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
	return s.closeDigest(ctx, userID, cache.SessionFingerprint(sessionID), status, UserSessionClosure{Reason: reason})
}

func (s *UserSessionService) RevokeSession(ctx context.Context, id int, closure UserSessionClosure) error {
	var record model.SysUserSession
	if err := s.db.DB.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return appErrors.WrapDatabaseError(err)
	}
	if record.Status != model.UserSessionStatusActive || !time.Time(record.ExpiresAt).After(s.now().UTC()) {
		return nil
	}
	if err := s.tokens.RevokeSessionDigest(record.UserID, record.SessionKeyHash); err != nil {
		return err
	}
	return s.closeDigest(ctx, record.UserID, record.SessionKeyHash, model.UserSessionStatusForcedOffline, closure)
}

func (s *UserSessionService) RevokeUser(ctx context.Context, userID int, status string, closure UserSessionClosure) error {
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
	updates := map[string]any{
		"status": status, "logout_reason": strings.TrimSpace(closure.Reason), "logout_at": &now,
		"closed_by_user_id": closure.OperatorID, "closed_by_user_name": strings.TrimSpace(closure.OperatorName),
	}
	if err := s.db.DB.WithContext(ctx).Model(&model.SysUserSession{}).
		Where("user_id = ? AND status = ?", userID, model.UserSessionStatusActive).
		Updates(updates).Error; err != nil {
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
	base, now, onlineSince, err := s.buildUserSessionQuery(ctx, req)
	if err != nil {
		return response.UserSessionListRes{}, err
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return response.UserSessionListRes{}, appErrors.WrapDatabaseError(err)
	}
	var rows []userSessionRow
	selectFields := "session.*, COALESCE(NULLIF(session.user_name_snapshot, ''), NULLIF(account.user_name, ''), '已删除用户') AS user_name, CASE WHEN account.id IS NULL OR account.gmt_delete IS NOT NULL THEN TRUE ELSE FALSE END AS user_deleted"
	if err := base.Select(selectFields).
		Order("session.login_at DESC, session.id DESC").
		Offset((page - 1) * num).Limit(num).Scan(&rows).Error; err != nil {
		return response.UserSessionListRes{}, appErrors.WrapDatabaseError(err)
	}
	items := mapUserSessionRows(rows, currentSessionID, now, onlineSince)
	db := s.db.DB.WithContext(ctx)
	onlineBase := db.Model(&model.SysUserSession{}).
		Where("gmt_delete IS NULL AND status = ? AND last_seen_at >= ? AND expires_at > ?", model.UserSessionStatusActive, onlineSince, now)
	var onlineSessions int64
	if err := onlineBase.Count(&onlineSessions).Error; err != nil {
		return response.UserSessionListRes{}, appErrors.WrapDatabaseError(err)
	}
	var onlineUsers int64
	if err := db.Model(&model.SysUserSession{}).
		Where("gmt_delete IS NULL AND status = ? AND last_seen_at >= ? AND expires_at > ?", model.UserSessionStatusActive, onlineSince, now).
		Distinct("user_id").Count(&onlineUsers).Error; err != nil {
		return response.UserSessionListRes{}, appErrors.WrapDatabaseError(err)
	}
	return response.UserSessionListRes{
		Items: items, Total: int(total), OnlineUsers: int(onlineUsers),
		OnlineSessions: int(onlineSessions), OnlineDevices: int(onlineSessions),
	}, nil
}

func (s *UserSessionService) Export(ctx context.Context, req request.UserSessionQueryReq) ([]response.UserSessionRes, error) {
	base, now, onlineSince, err := s.buildUserSessionQuery(ctx, req)
	if err != nil {
		return nil, err
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, appErrors.WrapDatabaseError(err)
	}
	if total > userSessionExportLimit {
		return nil, appErrors.ErrUserSessionExportTooLarge
	}
	var rows []userSessionRow
	selectFields := "session.*, COALESCE(NULLIF(session.user_name_snapshot, ''), NULLIF(account.user_name, ''), '已删除用户') AS user_name, CASE WHEN account.id IS NULL OR account.gmt_delete IS NOT NULL THEN TRUE ELSE FALSE END AS user_deleted"
	if err := base.Select(selectFields).
		Order("session.login_at DESC, session.id DESC").Limit(userSessionExportLimit).Scan(&rows).Error; err != nil {
		return nil, appErrors.WrapDatabaseError(err)
	}
	return mapUserSessionRows(rows, "", now, onlineSince), nil
}

func (s *UserSessionService) buildUserSessionQuery(ctx context.Context, req request.UserSessionQueryReq) (*gorm.DB, time.Time, time.Time, error) {
	now := s.now().UTC()
	onlineSince := now.Add(-userSessionOnlineWindow)
	db := s.db.DB.WithContext(ctx)
	sessionTable := db.NamingStrategy.TableName("SysUserSession")
	userTable := db.NamingStrategy.TableName("SysUser")
	base := db.Table(sessionTable + " AS session").
		Joins("LEFT JOIN " + userTable + " AS account ON account.id = session.user_id").
		Where("session.gmt_delete IS NULL")
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("LOWER(COALESCE(NULLIF(session.user_name_snapshot, ''), account.user_name, '')) LIKE ? OR LOWER(session.ip_address) LIKE ? OR LOWER(session.browser) LIKE ? OR LOWER(session.operating_system) LIKE ?", like, like, like, like)
	}
	if req.LoginStartedAt != nil && !req.LoginStartedAt.IsZero() {
		base = base.Where("session.login_at >= ?", time.Time(*req.LoginStartedAt).UTC())
	}
	if req.LoginEndedAt != nil && !req.LoginEndedAt.IsZero() {
		base = base.Where("session.login_at <= ?", time.Time(*req.LoginEndedAt).UTC())
	}
	if req.LoginStartedAt != nil && req.LoginEndedAt != nil && !req.LoginStartedAt.IsZero() && !req.LoginEndedAt.IsZero() && time.Time(*req.LoginStartedAt).After(time.Time(*req.LoginEndedAt)) {
		return nil, time.Time{}, time.Time{}, appErrors.ErrParamInvalid
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
		return nil, time.Time{}, time.Time{}, appErrors.ErrParamInvalid
	}
	return base, now, onlineSince, nil
}

func mapUserSessionRows(rows []userSessionRow, currentSessionID string, now, onlineSince time.Time) []response.UserSessionRes {
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
			ID: row.Id, UserID: row.UserID, UserName: row.UserName, UserDeleted: row.UserDeleted, Status: status,
			Online: online, Current: currentDigest != "" && row.SessionKeyHash == currentDigest,
			LoginAt: row.LoginAt, LastSeenAt: row.LastSeenAt, ExpiresAt: row.ExpiresAt,
			LogoutAt: row.LogoutAt, LogoutReason: row.LogoutReason,
			ClosedByUserID: row.ClosedByUserID, ClosedByUserName: row.ClosedByUserName,
			LoginChannel: row.LoginChannel, IPAddress: row.IPAddress, UserAgent: row.UserAgent,
			DeviceType: row.DeviceType, Browser: row.Browser, OperatingSystem: row.OperatingSystem,
		})
	}
	return items
}

func (s *UserSessionService) closeDigest(ctx context.Context, userID int, digest, status string, closure UserSessionClosure) error {
	now := model.CustomTime(s.now().UTC())
	result := s.db.DB.WithContext(ctx).Model(&model.SysUserSession{}).
		Where("user_id = ? AND session_key_hash = ? AND status = ?", userID, digest, model.UserSessionStatusActive).
		Updates(map[string]any{
			"status": status, "logout_reason": strings.TrimSpace(closure.Reason), "logout_at": &now,
			"closed_by_user_id": closure.OperatorID, "closed_by_user_name": strings.TrimSpace(closure.OperatorName),
		})
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
