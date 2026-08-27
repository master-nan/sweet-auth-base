package initialize

import (
	"backend/config"
	"backend/internal/integration"
	"backend/internal/organization/hrsync"
	"backend/service"
	"strings"
	"time"
)

// ProvideOrganizationSyncConsumerRegistry 默认隐藏 HR Consumer；只有服务端明确启用并配置来源时区后才开放。
func ProvideOrganizationSyncConsumerRegistry(domain *service.OrganizationHRSyncService, server *config.Server) (*integration.StaticSyncConsumerRegistry, error) {
	if server == nil || !server.Integration.OrganizationHR.Enabled {
		return integration.NewStaticSyncConsumerRegistry(hrsync.DisabledConsumerRegistrations(domain)...)
	}
	location, err := time.LoadLocation(strings.TrimSpace(server.Integration.OrganizationHR.SourceTimezone))
	if err != nil {
		return nil, hrsync.ErrSourceContractInvalid
	}
	contract, err := hrsync.NewExplicitSourceContract(hrsync.OrganizationHRSourceSystemCode, location)
	if err != nil {
		return nil, err
	}
	registrations, err := hrsync.EnabledConsumerRegistrations(domain, contract)
	if err != nil {
		return nil, err
	}
	return integration.NewStaticSyncConsumerRegistry(registrations...)
}
