package initialize

import (
	"backend/internal/integration"
	"backend/internal/organization/hrsync"
	"backend/service"
)

// ProvideOrganizationSyncConsumerRegistry 固定生产 code/version，但在 HR P0
// 契约关闭前保持 disabled，因此配置中心无法引用或启用这些 Consumer。
func ProvideOrganizationSyncConsumerRegistry(domain *service.OrganizationHRSyncService) (*integration.StaticSyncConsumerRegistry, error) {
	return integration.NewStaticSyncConsumerRegistry(hrsync.DisabledConsumerRegistrations(domain)...)
}
