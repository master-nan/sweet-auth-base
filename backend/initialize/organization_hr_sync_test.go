package initialize

import (
	"backend/config"
	"backend/service"
	"testing"
)

func TestProvideOrganizationSyncConsumerRegistryDefaultsDisabled(t *testing.T) {
	registry, err := ProvideOrganizationSyncConsumerRegistry(service.NewOrganizationHRSyncService(nil, nil), &config.Server{})
	if err != nil {
		t.Fatal(err)
	}
	if metadata := registry.ListMetadata(); len(metadata) != 0 {
		t.Fatalf("disabled HR consumers=%d", len(metadata))
	}
}

func TestProvideOrganizationSyncConsumerRegistryUsesConfiguredSourceTimezone(t *testing.T) {
	registry, err := ProvideOrganizationSyncConsumerRegistry(service.NewOrganizationHRSyncService(nil, nil), &config.Server{Integration: config.Integration{
		OrganizationHR: config.IntegrationOrganizationHR{Enabled: true, SourceTimezone: "Asia/Shanghai"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if metadata := registry.ListMetadata(); len(metadata) != 7 {
		t.Fatalf("enabled HR consumers=%d", len(metadata))
	}
}

func TestProvideOrganizationSyncConsumerRegistryRejectsInvalidSourceTimezone(t *testing.T) {
	_, err := ProvideOrganizationSyncConsumerRegistry(service.NewOrganizationHRSyncService(nil, nil), &config.Server{Integration: config.Integration{
		OrganizationHR: config.IntegrationOrganizationHR{Enabled: true, SourceTimezone: "invalid/timezone"},
	}})
	if err == nil {
		t.Fatal("expected invalid source timezone to fail")
	}
}
