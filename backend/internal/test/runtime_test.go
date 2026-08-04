package testutil

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConfigureGinTestMode(t *testing.T) {
	ConfigureGinTestMode()
	if got := gin.Mode(); got != gin.TestMode {
		t.Fatalf("Gin mode = %q, want %q", got, gin.TestMode)
	}
}

func TestPostgreSQLTestsRequired(t *testing.T) {
	for _, item := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty", value: "", want: false},
		{name: "zero", value: "0", want: false},
		{name: "one", value: "1", want: true},
		{name: "true", value: "TRUE", want: true},
		{name: "yes", value: " yes ", want: true},
	} {
		t.Run(item.name, func(t *testing.T) {
			t.Setenv(RequirePostgreSQLEnv, item.value)
			if got := PostgreSQLTestsRequired(); got != item.want {
				t.Fatalf("PostgreSQLTestsRequired() = %t, want %t", got, item.want)
			}
		})
	}
}
