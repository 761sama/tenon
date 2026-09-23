package conf

import "time"

// HTTPConfig 为 Web 服务配置，tenon.WebServer 仅需此配置。
type HTTPConfig struct {
	Debug          bool          `json:"debug"`           // 调试模式：gin 调试模式、详细错误返回
	Address        string        `json:"address"`         // 监听地址，默认 0.0.0.0
	Port           int           `json:"port"`            // HTTP 端口，-1 表示禁用 HTTP
	HTTPSPort      int           `json:"https_port"`      // HTTPS 端口，-1 表示禁用 HTTPS
	CertFile       string        `json:"cert_file"`       // TLS 证书文件路径（相对路径基于工作目录解析）
	KeyFile        string        `json:"key_file"`        // TLS 私钥文件路径（相对路径基于工作目录解析）
	EnableQUIC     bool          `json:"enable_quic"`     // 是否启用 QUIC (HTTP/3)，与 HTTPS 同地址监听并自动附加 Alt-Svc 响应头
	TrustedProxies []string      `json:"trusted_proxies"` // 可信反向代理 IP/IP 段(CIDR)，为空表示不信任任何代理
	AllowOrigins   []string      `json:"allow_origins"`   // CORS 允许的来源列表，["*"] 表示允许所有来源
	AllowMethods   []string      `json:"allow_methods"`   // CORS 允许的请求方法列表
	AllowHeaders   []string      `json:"allow_headers"`   // CORS 允许的请求头列表
	MaxBodySize    int64         `json:"max_body_size"`   // 请求体最大字节数，0 使用默认值 32MB，负数表示不限制
	ReadHeaderTimeout time.Duration `json:"read_header_timeout"` // 读取请求头超时（防 Slowloris），0 使用默认值 10s
	ReadTimeout    time.Duration `json:"read_timeout"`    // 读超时（纳秒），0 使用默认值 30s
	WriteTimeout   time.Duration `json:"write_timeout"`   // 写超时（纳秒），0 使用默认值 30s
	IdleTimeout    time.Duration `json:"idle_timeout"`    // Keep-Alive 空闲连接超时，0 使用默认值 120s
}

// 构造默认 HTTP 配置，使用方可在此基础上按需修改。
// 出参: 填充了合理默认值的 HTTPConfig
func DefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Debug:          false,
		Address:        "0.0.0.0",
		Port:           8080,
		HTTPSPort:      -1,
		TrustedProxies: []string{"127.0.0.1"},
		AllowOrigins:   []string{"*"},
		AllowMethods:   []string{"*"},
		AllowHeaders:   []string{"*"},
		MaxBodySize:       32 << 20,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// LogConfig 为日志配置（供 tenon.InitLog 显式初始化使用）。
type LogConfig struct {
	Enable     bool   `json:"enable"`      // 是否写入日志文件（关闭时仅输出到标准输出）
	Name       string `json:"name"`        // 日志文件名（相对路径基于工作目录解析）
	MaxSize    int    `json:"max_size"`    // 单个日志文件最大大小 (MB)
	MaxBackups int    `json:"max_backups"` // 保留的备份文件数
	MaxAge     int    `json:"max_age"`     // 日志文件保留天数
	Compress   bool   `json:"compress"`    // 是否压缩历史日志文件
}

// DatabaseConfig 为数据库组配置（供 tenon.DB.Init 显式初始化使用），Type 为空字符串时拒绝初始化。
type DatabaseConfig struct {
	Type     string         `json:"type"`     // 数据库类型：sqlite3 / mysql
	Master   DBNodeConfig   `json:"master"`   // 主库配置
	Replicas []DBNodeConfig `json:"replicas"` // 从库配置（读写分离，可选）
}

// DBNodeConfig 为单个数据库节点配置。
type DBNodeConfig struct {
	Host            string        `json:"host"`              // 数据库主机
	Port            int           `json:"port"`              // 数据库端口，0 表示使用默认端口
	User            string        `json:"user"`              // 数据库用户
	Password        string        `json:"password"`          // 数据库密码
	Name            string        `json:"name"`              // 数据库名称
	DBFile          string        `json:"db_file"`           // SQLite 文件路径（相对路径基于工作目录解析）
	SSLMode         string        `json:"ssl_mode"`          // SSL 模式（MySQL 暂不使用，预留给 Postgres 系）
	TablePrefix     string        `json:"table_prefix"`      // 表前缀（仅主库生效）
	MaxOpenConns    int           `json:"max_open_conns"`    // 最大打开连接数，0 使用类型默认值（mysql 50 / sqlite 1）
	MaxIdleConns    int           `json:"max_idle_conns"`    // 最大空闲连接数，0 使用类型默认值（mysql 10）
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"` // 连接最大存活时间（纳秒），0 使用类型默认值（mysql 1h）
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`// 连接最大空闲时间（纳秒），0 使用类型默认值（mysql 10m）
}

// RedisConfig 为 Redis 配置（供 tenon.Redis.Init 显式初始化使用）。
type RedisConfig struct {
	Host     string `json:"host"`     // Redis 主机
	Port     int    `json:"port"`     // Redis 端口
	Password string `json:"password"` // Redis 密码
	DB       int    `json:"db"`       // Redis 编号
}
