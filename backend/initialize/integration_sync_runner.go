package initialize

import (
	"backend/config"
	myerrors "backend/internal/errors"
	"backend/internal/integration"
	"backend/service"
	"strings"
	"time"
)

func ProvideIntegrationSyncRunnerConfig(server *config.Server) (integration.SyncRunnerConfig, error) {
	if server == nil {
		return integration.SyncRunnerConfig{}, myerrors.ErrSyncRunnerInvalidConfig
	}
	value := server.Integration.SyncRunner
	return integration.NewSyncRunnerConfig(integration.SyncRunnerConfig{
		Enabled: value.Enabled, RunnerID: strings.TrimSpace(value.RunnerID), PollInterval: time.Duration(value.PollInterval) * time.Second,
		ScheduleBatchSize: value.ScheduleBatchSize, CoordinateBatchSize: value.CoordinateBatchSize, ShutdownTimeout: time.Duration(value.ShutdownTimeout) * time.Second,
	})
}

func ProvideSyncBusinessResultProvider() integration.SyncBusinessResultProvider {
	return integration.NewPersistedSyncBusinessResultProvider()
}

func ProvideIntegrationSyncRunner(coordinator *service.IntegrationSyncCoordinator, config integration.SyncRunnerConfig) (*integration.IntegrationSyncRunner, error) {
	return integration.NewIntegrationSyncRunner(coordinator, config)
}
