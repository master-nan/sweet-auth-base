package initialize

import (
	"backend/config"
	"backend/internal/integration"
	"errors"
	"testing"
	"time"
)

func TestProvideIntegrationWorkerRunnerConfigDefaultsDisabled(t *testing.T) {
	value, err := ProvideIntegrationWorkerRunnerConfig(&config.Server{})
	if err != nil {
		t.Fatalf("provide defaults: %v", err)
	}
	if value.Enabled || value.WorkerID != "integration-disabled" || value.PollInterval != 5*time.Second || value.ClaimBatchSize != 8 || value.InstanceConcurrency != 4 || value.LeaseDuration != 3*time.Minute {
		t.Fatalf("unexpected default worker config: %+v", value)
	}
}

func TestProvideIntegrationWorkerRunnerConfigUsesConfiguredWorkerIdentity(t *testing.T) {
	value, err := ProvideIntegrationWorkerRunnerConfig(&config.Server{Integration: config.Integration{Worker: config.IntegrationWorker{
		Enabled: true, WorkerID: "runtime-instance-7", PollInterval: 6, ClaimBatchSize: 2, InstanceConcurrency: 2, LeaseRecoveryInterval: 20, ShutdownTimeout: 8, LeaseDuration: 200,
	}}})
	if err != nil {
		t.Fatalf("provide configured worker: %v", err)
	}
	if !value.Enabled || value.WorkerID != "runtime-instance-7" || value.PollInterval != 6*time.Second || value.LeaseDuration != 200*time.Second {
		t.Fatalf("unexpected configured worker: %+v", value)
	}
}

func TestProvideIntegrationWorkerRunnerConfigRejectsMissingEnabledWorkerIdentity(t *testing.T) {
	_, err := ProvideIntegrationWorkerRunnerConfig(&config.Server{Integration: config.Integration{Worker: config.IntegrationWorker{Enabled: true}}})
	if err == nil {
		t.Fatal("expected enabled worker without worker_id to fail")
	}
}

func TestProvideIntegrationTransportClientUsesStrictDefaults(t *testing.T) {
	policy, err := ProvideIntegrationEndpointPolicy(&config.Server{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := ProvideIntegrationTransportClient(policy)
	if err != nil || client == nil {
		t.Fatalf("provide default transport client: client=%v err=%v", client, err)
	}
}

func TestProvideIntegrationTransportClientRejectsInvalidApprovedCIDR(t *testing.T) {
	_, err := ProvideIntegrationEndpointPolicy(&config.Server{Integration: config.Integration{
		EndpointPolicy: config.IntegrationEndpointPolicy{AllowHTTP: true, ApprovedPrivateCIDRs: []string{"not-a-cidr"}},
	}})
	var transportErr *integration.TransportError
	if !errors.As(err, &transportErr) || transportErr.Category() != integration.TransportErrorInvalidConfig {
		t.Fatalf("invalid CIDR error=%v", err)
	}
}
