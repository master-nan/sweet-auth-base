package initialize

import (
	"backend/config"
	myerrors "backend/internal/errors"
	"errors"
	"testing"
	"time"
)

func TestProvideIntegrationSyncRunnerConfigDefaultsDisabled(t *testing.T) {
	value, err := ProvideIntegrationSyncRunnerConfig(&config.Server{})
	if err != nil {
		t.Fatal(err)
	}
	if value.Enabled || value.RunnerID != "sync-disabled" || value.PollInterval != 10*time.Second || value.ScheduleBatchSize != 8 || value.CoordinateBatchSize != 16 {
		t.Fatalf("defaults=%+v", value)
	}
}

func TestProvideIntegrationSyncRunnerConfigRejectsUnsafeConfig(t *testing.T) {
	_, err := ProvideIntegrationSyncRunnerConfig(&config.Server{Integration: config.Integration{SyncRunner: config.IntegrationSyncRunner{Enabled: true}}})
	if !errors.Is(err, myerrors.ErrSyncRunnerInvalidConfig) {
		t.Fatalf("missing runner id error=%v", err)
	}
	_, err = ProvideIntegrationSyncRunnerConfig(&config.Server{Integration: config.Integration{SyncRunner: config.IntegrationSyncRunner{Enabled: true, RunnerID: "sync", PollInterval: -1}}})
	if !errors.Is(err, myerrors.ErrSyncRunnerInvalidConfig) {
		t.Fatalf("negative poll error=%v", err)
	}
}
