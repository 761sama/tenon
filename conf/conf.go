package conf

import "time"

// HTTPConfig 为 Web 服务配置，tenon.WebServer 仅需此配置。
type HTTPConfig struct {
	Debug          bool          // 调试模式：gin 调试模式、详细错误返回
	Address        string        // 监听地址，默认 0.0.0.0
	Port           int           // HTTP 端口，-1 表示禁用 HTTP
	HTTPSPort      int           // HTTPS 端口，-1 表示禁用 HTTPS
	CertFile       string        // TLS 证书文件路径（相对路径基于工作目录解析）
	KeyFile        string        // TLS 私钥文件路径（相对路径基于工作目录解析）
	TrustedProxies []string      // 可信反向代理 IP/IP 段(CIDR)，为空表示不信任任何代理
	AllowOrigins   []string      // CORS 允许的来源列表，["*"] 表示允许所有来源
	AllowMethods   []string      // CORS 允许的请求方法列表
	AllowHeaders   []string      // CORS 允许的请求头列表
	ReadTimeout    time.Duration // 读超时，0 表示使用默认值
	WriteTimeout   time.Duration // 写超时，0 表示使用默认值
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
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
	}
}

// LogConfig 为日志配置（供 tenon.InitLog 显式初始化使用）。
type LogConfig struct {
	Enable     bool   // 是否写入日志文件（关闭时仅输出到标准输出）
	Name       string // 日志文件名（相对路径基于工作目录解析）
	MaxSize    int    // 单个日志文件最大大小 (MB)
	MaxBackups int    // 保留的备份文件数
	MaxAge     int    // 日志文件保留天数
	Compress   bool   // 是否压缩历史日志文件
}

// DatabaseConfig 为数据库组配置（供 tenon.DB.Init 显式初始化使用），Type 为空字符串时拒绝初始化。
type DatabaseConfig struct {
	Type     string         // 数据库类型：sqlite3 / mysql
	Master   DBNodeConfig   // 主库配置
	Replicas []DBNodeConfig // 从库配置（读写分离，可选）
}

// DBNodeConfig 为单个数据库节点配置。
type DBNodeConfig struct {
	Host        string // 数据库主机
	Port        int    // 数据库端口，0 表示使用默认端口
	User        string // 数据库用户
	Password    string // 数据库密码
	Name        string // 数据库名称
	DBFile      string // SQLite 文件路径（相对路径基于工作目录解析）
	SSLMode     string // SSL 模式（MySQL 暂不使用，预留给 Postgres 系）
	TablePrefix string // 表前缀（仅主库生效）
}

// RedisConfig 为 Redis 配置（供 tenon.Redis.Init 显式初始化使用）。
type RedisConfig struct {
	Host     string // Redis 主机
	Port     int    // Redis 端口
	Password string // Redis 密码
	DB       int    // Redis 编号
}
