package database

import (
	"gorm.io/gorm"

	"gopkg.761sama.com/tenon/conf"
)

// Ops 数据库操作门面：方法作用于显式初始化后的全局连接，
// 经根包暴露为 tenon.DB 使用。
type Ops struct{}

// 显式初始化数据库：按配置连接主库与从库，自动迁移已注册模型。
// 入参: cfg (数据库配置), debug (是否打印 SQL 日志)
// 出参: 连接或迁移失败时返回错误
func (Ops) Init(cfg conf.DatabaseConfig, debug bool) error { return Init(cfg, debug) }

// 关闭数据库连接。
func (Ops) Close() { Close() }

// 数据库是否可用。
// 出参: 数据库是否已初始化
func (Ops) IsAvailable() bool { return IsAvailable() }

// 注册模型：数据库初始化时自动迁移；已初始化后注册则立即迁移。
// 入参: models (模型指针列表，如 new(User))
func (Ops) RegisterModels(models ...any) { RegisterModels(models...) }

// 获取 gorm 数据库连接，未初始化时返回 nil。
// 出参: gorm 数据库连接
func (Ops) GetDB() *gorm.DB { return GetDB() }

// 获取数据库连接（支持事务）：tx 不为 nil 时使用事务，否则使用默认连接。
// 入参: tx (事务连接，可为 nil)
// 出参: 实际使用的数据库连接
func (Ops) GetDBWithTx(tx *gorm.DB) *gorm.DB { return GetDBWithTx(tx) }

// 执行事务。
// 入参: fn (事务函数，参数为事务连接)
// 出参: 事务执行错误
func (Ops) Transaction(fn TransactionFunc) error { return Transaction(fn) }

// 自动迁移（带表引擎选项）。
// 入参: dst (模型指针列表)
// 出参: 迁移错误
func (Ops) AutoMigrate(dst ...any) error { return AutoMigrate(dst...) }

// 处理不同数据库的列名引用兼容问题。
// 入参: name (列名)
// 出参: 带引用符的列名
func (Ops) ColumnName(name string) string { return ColumnName(name) }
