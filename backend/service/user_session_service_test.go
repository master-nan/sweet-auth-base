package service

import (
	"backend/config"
	"backend/dto/request"
	"backend/internal/cache"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/internal/token"
	"backend/internal/utils"
	"backend/model"
	"context"
	"testing"
	"time"
)

func TestUserSessionLifecycleUsesSnowflakeIDAndRevokesRedisSession(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.SysUser{}, &model.SysUserSession{})
	testutil.MustCreate(t, db, &model.SysUser{Basic: model.Basic{Id: 1, State: true}, UserName: "admin"})
	sf, err := utils.NewSnowflake(0)
	if err != nil {
		t.Fatal(err)
	}
	server := &config.Server{Name: "sweet_admin"}
	server.Conf.Salt = "session-test-salt"
	tokenState := cache.NewTokenBlackCache(newAuthMemoryCache())
	tokens := NewAuthTokenService(token.JWTToken{Generator: token.NewJWTGenerator()}, tokenState, server)
	service := NewUserSessionService(&database.PrimaryDB{DB: db}, sf, tokens)
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }

	pair, err := tokens.Issue(context.Background(), 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Open(context.Background(), 1, "admin", pair.SessionID, now, now.Add(authRefreshTokenTTL), UserSessionClient{
		IPAddress: "127.0.0.1", UserAgent: "Mozilla/5.0 (Macintosh) Chrome/120.0", Channel: string(AuthChannelAdminPassword),
	}); err != nil {
		t.Fatal(err)
	}

	result, err := service.Query(context.Background(), request.UserSessionQueryReq{Status: "online", Page: 1, Num: 20}, pair.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].ID <= 0 || !result.Items[0].Current || !result.Items[0].Online {
		t.Fatalf("unexpected online session: %+v", result)
	}
	if result.Items[0].UserName != "admin" || result.Items[0].UserDeleted {
		t.Fatalf("unexpected user snapshot: %+v", result.Items[0])
	}
	if result.OnlineUsers != 1 || result.OnlineSessions != 1 || result.OnlineDevices != 1 {
		t.Fatalf("unexpected online counters: %+v", result)
	}

	if err := service.RevokeSession(context.Background(), result.Items[0].ID, UserSessionClosure{
		Reason: "管理员测试下线", OperatorID: 1, OperatorName: "admin",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tokens.ValidateAccess(context.Background(), pair.AccessToken); err == nil {
		t.Fatal("revoked session still accepts access token")
	}
	closed, err := service.Query(context.Background(), request.UserSessionQueryReq{Status: "closed", Page: 1, Num: 20}, pair.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(closed.Items) != 1 || closed.Items[0].Status != model.UserSessionStatusForcedOffline || closed.Items[0].LogoutReason != "管理员测试下线" || closed.Items[0].ClosedByUserName != "admin" {
		t.Fatalf("unexpected closed session: %+v", closed.Items)
	}

	if err := db.Delete(&model.SysUser{}, 1).Error; err != nil {
		t.Fatal(err)
	}
	history, err := service.Query(context.Background(), request.UserSessionQueryReq{Status: "all", Page: 1, Num: 20}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Items) != 1 || history.Items[0].UserName != "admin" || !history.Items[0].UserDeleted {
		t.Fatalf("deleted account history was not preserved: %+v", history.Items)
	}
	exported, err := service.Export(context.Background(), request.UserSessionQueryReq{Status: "all"})
	if err != nil {
		t.Fatal(err)
	}
	if len(exported) != 1 || exported[0].ClosedByUserID != 1 {
		t.Fatalf("unexpected exported sessions: %+v", exported)
	}
}
