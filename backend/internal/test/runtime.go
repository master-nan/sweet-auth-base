package testutil

import (
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const (
	PostgreSQLDSNEnv     = "SWEET_TEST_POSTGRES_DSN"
	RequirePostgreSQLEnv = "SWEET_REQUIRE_POSTGRES_TESTS"
)

// ConfigureGinTestMode 由包级 TestMain 调用，统一设置 Gin 测试模式。
func ConfigureGinTestMode() {
	gin.SetMode(gin.TestMode)
}

// PostgreSQLDSN 返回 PostgreSQL 专项测试连接串。
// 本地未配置时跳过；CI 明确要求专项测试时，缺少连接串直接失败。
func PostgreSQLDSN(t testing.TB) string {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv(PostgreSQLDSNEnv))
	if dsn != "" {
		return dsn
	}
	message := "PostgreSQL 专项测试未执行：请设置 " + PostgreSQLDSNEnv
	if PostgreSQLTestsRequired() {
		t.Fatal(message)
	}
	t.Skip(message)
	return ""
}

func PostgreSQLTestsRequired() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(RequirePostgreSQLEnv)))
	return value == "1" || value == "true" || value == "yes"
}
