package initialize

import (
	"backend/config"
	myerrors "backend/internal/errors"
	"backend/internal/integration"
	"backend/internal/utils"
	"backend/repository"
	"strings"
	"time"
)

// ProvideIntegrationWorkerRunnerConfig 将配置文件中的秒级配置转换为受控运行配置。
// Worker 默认关闭；启用时必须提供稳定实例标识，避免多实例租约日志混淆。
func ProvideIntegrationWorkerRunnerConfig(server *config.Server) (integration.WorkerRunnerConfig, error) {
	if server == nil {
		return integration.WorkerRunnerConfig{}, myerrors.ErrIntegrationWorkerInvalidConfig
	}
	worker := server.Integration.Worker
	return integration.NewWorkerRunnerConfig(integration.WorkerRunnerConfig{
		Enabled:               worker.Enabled,
		WorkerID:              strings.TrimSpace(worker.WorkerID),
		PollInterval:          time.Duration(worker.PollInterval) * time.Second,
		ClaimBatchSize:        worker.ClaimBatchSize,
		InstanceConcurrency:   worker.InstanceConcurrency,
		LeaseRecoveryInterval: time.Duration(worker.LeaseRecoveryInterval) * time.Second,
		ShutdownTimeout:       time.Duration(worker.ShutdownTimeout) * time.Second,
	})
}

// ProvideIntegrationTransportClient 使用默认 HTTPS-only 端点策略；放宽网络边界不属于 Worker 生命周期职责。
func ProvideIntegrationTransportClient() (*integration.HTTPTransportClient, error) {
	return integration.NewHTTPTransportClient(integration.DefaultEndpointPolicy(), integration.TransportClientOptions{})
}

// ProvideIntegrationConcurrencyGuard 将 Runner 实例配额同步传入 Engine 的三级进程内保护。
// 数据库租约仍是多实例唯一领取的唯一依据。
func ProvideIntegrationConcurrencyGuard(worker integration.WorkerRunnerConfig) (*integration.InMemoryConcurrencyGuard, error) {
	return integration.NewInMemoryConcurrencyGuard(worker.InstanceConcurrency, worker.InstanceConcurrency, worker.InstanceConcurrency)
}

func ProvideIntegrationExecutionEngine(
	executions repository.IntegrationExecutionRepository,
	systems repository.ExternalSystemRepository,
	interfaces repository.InterfaceDefinitionRepository,
	credentials repository.CredentialRepository,
	provider *integration.CredentialProvider,
	transport *integration.HTTPTransportClient,
	guard *integration.InMemoryConcurrencyGuard,
	snowflake *utils.Snowflake,
	worker integration.WorkerRunnerConfig,
) (*integration.IntegrationExecutionEngine, error) {
	return integration.NewIntegrationExecutionEngine(
		executions, systems, interfaces, credentials, provider, transport, guard, snowflake,
		integration.ExecutionEngineOptions{WorkerID: worker.WorkerID, BatchSize: worker.ClaimBatchSize},
	)
}

func ProvideIntegrationWorkerRunner(
	engine *integration.IntegrationExecutionEngine,
	worker integration.WorkerRunnerConfig,
) (*integration.IntegrationWorkerRunner, error) {
	return integration.NewIntegrationWorkerRunner(engine, worker)
}
