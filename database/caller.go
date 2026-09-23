package database

import (
	"gorm.io/gorm"

	"gopkg.761sama.com/tenon/conf"
)

// Ops 数据库默认实例门面：方法作用于显式初始化后的默认实例，
// 经根包暴露为 tenon.DB 使用；命名实例通过 InitNamed + Named 获取。
type Ops struct{}

// 显式初始化默认实例：按配置连接主库与从库，自动迁移已注册模型。
// 入参: cfg (数据库配置), debug (是否打印 SQL 日志)
// 出参: 连接或迁移失败时返回错误
func (Ops) Init(cfg conf.DatabaseConfig, debug bool) error { return Init(cfg, debug) }

// 显式初始化命名实例（多实例）。
// 入参: name (实例名称), cfg (数据库配置), debug (是否打印 SQL 日志)
// 出参: 连接或迁移失败时返回错误
func (Ops) InitNamed(name string, cfg conf.DatabaseConfig, debug bool) error {
	return InitNamed(name, cfg, debug)
}

// 获取命名实例，不存在时返回 nil。
// 入参: name (实例名称)
// 出参: 数据库实例
func (Ops) Named(name string) *Instance { return Named(name) }

// 关闭全部实例连接。
func (Ops) Close() { CloseAll() }

// 注册模型：各实例初始化时自动迁移；已初始化的实例立即迁移。
// 入参: models (模型指针列表，如 new(User))
func (Ops) RegisterModels(models ...any) { RegisterModels(models...) }

// 默认实例是否可用。
// 出参: 默认实例是否已初始化
func (Ops) IsAvailable() bool { return IsAvailable() }

// 默认实例的 gorm 连接，未初始化时返回 nil。
// 出参: gorm 数据库连接
func (Ops) GetDB() *gorm.DB { return GetDB() }

// 获取默认实例连接（支持事务）。
// 入参: tx (事务连接，可为 nil)
// 出参: 实际使用的数据库连接
func (Ops) GetDBWithTx(tx *gorm.DB) *gorm.DB { return GetDBWithTx(tx) }

// 在默认实例上执行事务。
// 入参: fn (事务函数，参数为事务连接)
// 出参: 事务执行错误
func (Ops) Transaction(fn TransactionFunc) error { return Transaction(fn) }

// 在默认实例上自动迁移（带表引擎选项）。
// 入参: dst (模型指针列表)
// 出参: 迁移错误
func (Ops) AutoMigrate(dst ...any) error { return AutoMigrate(dst...) }

// 默认实例的列名引用兼容处理。
// 入参: name (列名)
// 出参: 带引用符的列名
func (Ops) ColumnName(name string) string { return ColumnName(name) }
