package controller

import (
	testutil "backend/internal/test"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	testutil.ConfigureGinTestMode()
	os.Exit(m.Run())
}
