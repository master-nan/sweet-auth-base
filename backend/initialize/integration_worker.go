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
		LeaseDuration:         time.Duration(worker.LeaseDuration) * time.Second,
	})
}

// ProvideIntegrationEndpointPolicy 默认只允许 HTTPS 公网地址；内部 HTTP 地址必须由服务端同时批准协议和网段。
func ProvideIntegrationEndpointPolicy(server *config.Server) (integration.EndpointPolicy, error) {
	if server == nil {
		return integration.EndpointPolicy{}, myerrors.ErrIntegrationWorkerInvalidConfig
	}
	return integration.NewEndpointPolicy(
		server.Integration.EndpointPolicy.AllowHTTP,
		server.Integration.EndpointPolicy.ApprovedPrivateCIDRs,
		nil,
	)
}

func ProvideIntegrationTransportClient(policy integration.EndpointPolicy) (*integration.HTTPTransportClient, error) {
	return integration.NewHTTPTransportClient(policy, integration.TransportClientOptions{})
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
	syncBatches repository.IntegrationSyncBatchRepository,
	provider *integration.CredentialProvider,
	transport *integration.HTTPTransportClient,
	guard *integration.InMemoryConcurrencyGuard,
	syncConsumers integration.SyncResultConsumerRegistry,
	snowflake *utils.Snowflake,
	worker integration.WorkerRunnerConfig,
) (*integration.IntegrationExecutionEngine, error) {
	return integration.NewIntegrationExecutionEngine(
		executions, systems, interfaces, credentials, syncBatches, provider, transport, guard, syncConsumers, snowflake,
		integration.ExecutionEngineOptions{WorkerID: worker.WorkerID, LeaseDuration: worker.LeaseDuration, BatchSize: worker.ClaimBatchSize},
	)
}

func ProvideIntegrationWorkerRunner(
	engine *integration.IntegrationExecutionEngine,
	worker integration.WorkerRunnerConfig,
) (*integration.IntegrationWorkerRunner, error) {
	return integration.NewIntegrationWorkerRunner(engine, worker)
}
