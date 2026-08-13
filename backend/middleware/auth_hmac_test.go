package middleware

import (
	"backend/internal/cache"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/internal/token"
	"backend/model"
	"backend/repository/impl"
	"backend/service"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type hmacMemoryCache struct {
	mu     sync.Mutex
	values map[string]any
}

func (m *hmacMemoryCache) Get(key string, target any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.values[key]
	if !ok {
		return cache.ErrCacheMiss
	}
	*(target.(*model.Application)) = value.(model.Application)
	return nil
}
func (m *hmacMemoryCache) Set(key string, value any, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if application, ok := value.(*model.Application); ok {
		m.values[key] = *application
	} else {
		m.values[key] = value
	}
	return nil
}
func (m *hmacMemoryCache) Del(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
	return nil
}
func (m *hmacMemoryCache) Exists(keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, key := range keys {
		if _, ok := m.values[key]; ok {
			count++
		}
	}
	return count, nil
}
func (*hmacMemoryCache) Expire(string, time.Duration) (bool, error) { return true, nil }

func TestAuthHMACUsesAuthoritativeApplicationState(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.Application{})
	application := model.Application{
		Basic: model.Basic{Id: 1, State: true}, AppKey: "app-key", AppSecret: "old-secret", Expiration: 3600,
	}
	testutil.MustCreate(t, db, &application)
	store := &hmacMemoryCache{values: map[string]any{}}
	applicationCache := cache.NewApplicationCache(store)
	applicationService := service.NewApplicationService(impl.NewApplicationRepositoryImpl(&database.PrimaryDB{DB: db}), applicationCache, nil)
	generator := token.NewHMACGenerator()
	value, err := generator.GenerateToken(token.Claims{ID: "1", IssuedAt: time.Now()}, token.Config{
		Issuer: strconv.Itoa(application.Id), SecretKey: application.AppSecret, AccessTokenExpiration: application.Expiration,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := applicationCache.SetExpiration(value, application, application.Expiration); err != nil {
		t.Fatal(err)
	}

	assertAllowed := func(want bool) {
		t.Helper()
		reached := false
		router := gin.New()
		router.Use(AuthHMACHandler(generator, applicationCache, applicationService))
		router.GET("/protected", func(*gin.Context) { reached = true })
		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		request.Header.Set("X-APP-TOKEN", value)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if reached != want {
			t.Fatalf("authoritative application state: reached=%v want=%v", reached, want)
		}
	}

	assertAllowed(true)
	if err := db.Model(&model.Application{}).Where("id = ?", application.Id).Update("state", false).Error; err != nil {
		t.Fatal(err)
	}
	assertAllowed(false)
	if err := db.Model(&model.Application{}).Where("id = ?", application.Id).Updates(map[string]any{"state": true, "app_secret": "new-secret"}).Error; err != nil {
		t.Fatal(err)
	}
	assertAllowed(false)
}
