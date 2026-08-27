/**
 * @Author: Nan
 * @Date: 2024/10/11 11:35
 */

package config

type Server struct {
	Name        string      `mapstructure:"name"`
	Version     string      `mapstructure:"version"`
	Port        int         `mapstructure:"port"`
	DBS         DBS         `mapstructure:"dbs"`
	Redis       Redis       `mapstructure:"redis"`
	Session     Session     `mapstructure:"session"`
	Security    Security    `mapstructure:"security"`
	Audit       Audit       `mapstructure:"audit"`
	Integration Integration `mapstructure:"integration"`
	WorkerID    int64       `mapstructure:"worker_id"`
	Conf        Conf        `mapstructure:"conf"`
	ALiYun      ALiYun      `mapstructure:"aliyun"`
	Upload      Upload      `mapstructure:"upload"`
}

type DBS struct {
	Primary DB `mapstructure:"primary"`
}

type DB struct {
	Host     string      `mapstructure:"host"`
	Port     int         `mapstructure:"port"`
	Name     string      `mapstructure:"name"`
	User     string      `mapstructure:"user"`
	Password string      `mapstructure:"password"`
	Prefix   string      `mapstructure:"prefix"`
	TLS      PostgresTLS `mapstructure:"tls"`
}

type PostgresTLS struct {
	Mode       string `mapstructure:"mode"`
	RootCAFile string `mapstructure:"root_ca_file"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
}

type Redis struct {
	Host            string   `mapstructure:"host"`
	Port            int      `mapstructure:"port"`
	DB              int      `mapstructure:"db"`
	Password        string   `mapstructure:"password"`
	PoolSize        int      `mapstructure:"pool_size"`
	MinIdleConns    int      `mapstructure:"min_idle_conns"`
	MaxIdleConns    int      `mapstructure:"max_idle_conns"`
	ConnMaxIdleTime int      `mapstructure:"conn_max_idle_time"`
	TLS             RedisTLS `mapstructure:"tls"`
}

type RedisTLS struct {
	Enabled    bool   `mapstructure:"enabled"`
	ServerName string `mapstructure:"server_name"`
	CAFile     string `mapstructure:"ca_file"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
}

type Session struct {
	Secret string `mapstructure:"secret"`
}

type Security struct {
	EnforceCasbinPolicyCoverage bool     `mapstructure:"enforce_casbin_policy_coverage"`
	CORSAllowedOrigins          []string `mapstructure:"cors_allowed_origins"`
	CORSAllowCredentials        bool     `mapstructure:"cors_allow_credentials"`
}

type Audit struct {
	AccessLogRetentionDays int `mapstructure:"access_log_retention_days"`
}

// Integration 仅保存集成运行时的服务端配置，不由普通请求修改。
type Integration struct {
	Worker         IntegrationWorker         `mapstructure:"worker"`
	SyncRunner     IntegrationSyncRunner     `mapstructure:"sync_runner"`
	EndpointPolicy IntegrationEndpointPolicy `mapstructure:"endpoint_policy"`
	OrganizationHR IntegrationOrganizationHR `mapstructure:"organization_hr"`
}

// IntegrationEndpointPolicy 只允许服务端显式批准内部 HTTP 地址。
// 默认值仍拒绝 HTTP 和私网地址。
type IntegrationEndpointPolicy struct {
	AllowHTTP            bool     `mapstructure:"allow_http"`
	ApprovedPrivateCIDRs []string `mapstructure:"approved_private_cidrs"`
}

// IntegrationOrganizationHR 控制内置 HR Consumer 是否可被同步任务引用。
// 来源时区用于解析无 UTC offset 的 changeTime。
type IntegrationOrganizationHR struct {
	Enabled        bool   `mapstructure:"enabled"`
	SourceTimezone string `mapstructure:"source_timezone"`
}

// IntegrationWorker 的时间字段单位均为秒，避免配置文件中使用无单位 duration。
type IntegrationWorker struct {
	Enabled               bool   `mapstructure:"enabled"`
	WorkerID              string `mapstructure:"worker_id"`
	PollInterval          int    `mapstructure:"poll_interval"`
	ClaimBatchSize        int    `mapstructure:"claim_batch_size"`
	InstanceConcurrency   int    `mapstructure:"instance_concurrency"`
	LeaseRecoveryInterval int    `mapstructure:"lease_recovery_interval"`
	ShutdownTimeout       int    `mapstructure:"shutdown_timeout"`
	LeaseDuration         int    `mapstructure:"lease_duration"`
}

// IntegrationSyncRunner 的时间字段单位为秒；默认关闭并且只接受服务端配置。
type IntegrationSyncRunner struct {
	Enabled             bool   `mapstructure:"enabled"`
	RunnerID            string `mapstructure:"runner_id"`
	PollInterval        int    `mapstructure:"poll_interval"`
	ScheduleBatchSize   int    `mapstructure:"schedule_batch_size"`
	CoordinateBatchSize int    `mapstructure:"coordinate_batch_size"`
	ShutdownTimeout     int    `mapstructure:"shutdown_timeout"`
}

type Conf struct {
	Salt   string `mapstructure:"salt"`
	Enable bool   `mapstructure:"enable"` //是否开启定时任务
}

type ALiYun struct {
	SMS SMS `mapstructure:"sms"`
}

type Upload struct {
	Driver              string    `mapstructure:"driver"`                    // 存储驱动: "local" 或 "oss"，默认 "local"
	Dir                 string    `mapstructure:"dir"`                       // 本地存储目录（支持绝对路径，如 /data/uploads）
	BaseURL             string    `mapstructure:"base_url"`                  // 文件访问URL前缀
	MaxSize             int64     `mapstructure:"max_size"`                  // 最大文件大小（MB）
	ChunkSize           int64     `mapstructure:"chunk_size"`                // 分片大小（MB），默认 5
	AllowedExtensions   []string  `mapstructure:"allowed_extensions"`        // 允许上传的扩展名，如 .png/.pdf
	AllowedMimeTypes    []string  `mapstructure:"allowed_mime_types"`        // 允许上传的 MIME 类型
	InlinePreviewMimes  []string  `mapstructure:"inline_preview_mime_types"` // 允许浏览器内联预览的 MIME 类型
	PublicPreview       bool      `mapstructure:"public_preview"`            // 是否允许通过 /files/:uuid 公开预览
	AccessTTLMinutes    int64     `mapstructure:"access_ttl_minutes"`        // 文件签名访问默认有效期（分钟）
	MaxAccessTTLMinutes int64     `mapstructure:"max_access_ttl_minutes"`    // 文件签名访问最大有效期（分钟）
	ChunkTTLHours       int       `mapstructure:"chunk_ttl_hours"`           // 未完成分片暂存保留时间（小时）
	ChunkCleanupMinutes int       `mapstructure:"chunk_cleanup_minutes"`     // 分片暂存清理周期（分钟）
	OSS                 OSSConfig `mapstructure:"oss"`                       // 阿里云 OSS 配置
}

type OSSConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	BucketName      string `mapstructure:"bucket_name"`
	BaseURL         string `mapstructure:"base_url"`  // CDN/自定义域名
	BasePath        string `mapstructure:"base_path"` // OSS路径前缀，默认 "uploads/"
}

type SMS struct {
	AccessKeyId         string `mapstructure:"access_key_id"`
	AccessKeySecret     string `mapstructure:"access_key_secret"`
	SendIntervalSeconds int    `mapstructure:"send_interval_seconds"`
}
