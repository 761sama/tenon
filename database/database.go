package database

import (
	"errors"
	"fmt"
	"sync"

	"gorm.io/gorm"
	log "github.com/sirupsen/logrus"

	"gopkg.761sama.com/tenon/bootstrap"
	"gopkg.761sama.com/tenon/conf"
)

// ErrDBUnavailable 数据库不可用哨兵错误：实例未初始化时由各操作返回。
var ErrDBUnavailable = errors.New("database unavailable")

// TransactionFunc 事务函数类型。
type TransactionFunc func(tx *gorm.DB) error

// Tx 事务连接类型（无需引用 gorm 包）。
type Tx = *gorm.DB

// DefaultName 默认实例名称。
const DefaultName = "default"

// Instance 数据库实例：持有独立的 gorm 连接，操作方法挂在实例上。
type Instance struct {
	db     *gorm.DB
	dbType string
}

var (
	mu               sync.RWMutex
	instances        = map[string]*Instance{} // 实例注册表（名称 -> 实例）
	registeredModels []any                    // 已注册待迁移的模型（全局共享，实例初始化时迁移）
)

// 显式初始化默认数据库实例。
// 入参: cfg (数据库配置，Type 为空时返回错误), debug (是否打印 SQL 日志)
// 出参: 连接或迁移失败时返回错误
func Init(cfg conf.DatabaseConfig, debug bool) error {
	return InitNamed(DefaultName, cfg, debug)
}

// 显式初始化命名数据库实例（多实例），同名重复初始化会覆盖旧实例（旧连接随之关闭）。
// 入参: name (实例名称), cfg (数据库配置), debug (是否打印 SQL 日志)
// 出参: 连接或迁移失败时返回错误
func InitNamed(name string, cfg conf.DatabaseConfig, debug bool) error {
	if cfg.Type == "" {
		return fmt.Errorf("database type is empty")
	}
	gormDB, err := connect(cfg, debug)
	if err != nil {
		return err
	}
	ins := &Instance{db: gormDB, dbType: cfg.Type}
	mu.Lock()
	if old, ok := instances[name]; ok {
		old.close()
	}
	instances[name] = ins
	models := registeredModels
	mu.Unlock()
	if len(models) > 0 {
		if err := ins.migrate(models...); err != nil {
			return fmt.Errorf("failed to auto migrate database: %w", err)
		}
	}
	bootstrap.RegisterRelease("database", CloseAll)
	return nil
}

// 获取命名实例，不存在时返回 nil。
// 入参: name (实例名称)
// 出参: 数据库实例
func Named(name string) *Instance {
	mu.RLock()
	defer mu.RUnlock()
	return instances[name]
}

// 获取默认实例，未初始化时返回持 nil 连接的空实例（各操作返回 ErrDBUnavailable）。
// 出参: 默认数据库实例
func Default() *Instance {
	if ins := Named(DefaultName); ins != nil {
		return ins
	}
	return &Instance{}
}

// 注册模型（全局共享）：各实例初始化时自动迁移；已初始化的实例立即迁移。
// 入参: models (模型指针列表，如 new(User))
func RegisterModels(models ...any) {
	mu.Lock()
	registeredModels = append(registeredModels, models...)
	var initialized []*Instance
	for _, ins := range instances {
		initialized = append(initialized, ins)
	}
	mu.Unlock()
	for _, ins := range initialized {
		ins.migrate(models...)
	}
}

// 默认实例是否可用。
// 出参: 默认实例是否已初始化
func IsAvailable() bool {
	return Default().IsAvailable()
}

// 默认实例的 gorm 连接，未初始化时返回 nil。
// 出参: gorm 数据库连接
func GetDB() *gorm.DB {
	return Default().GetDB()
}

// 在默认实例上执行事务。
// 入参: fn (事务函数，参数为事务连接)
// 出参: 事务执行错误
func Transaction(fn TransactionFunc) error {
	return Default().Transaction(fn)
}

// 获取默认实例连接（支持事务）：tx 不为 nil 时使用事务，否则使用默认连接。
// 入参: tx (事务连接，可为 nil)
// 出参: 实际使用的数据库连接
func GetDBWithTx(tx *gorm.DB) *gorm.DB {
	return Default().GetDBWithTx(tx)
}

// 在默认实例上自动迁移（带表引擎选项）。
// 入参: dst (模型指针列表)
// 出参: 迁移错误
func AutoMigrate(dst ...any) error {
	return Default().AutoMigrate(dst...)
}

// 默认实例的列名引用兼容处理。
// 入参: name (列名)
// 出参: 带引用符的列名
func ColumnName(name string) string {
	return Default().ColumnName(name)
}

// 关闭全部数据库实例连接。
func CloseAll() {
	mu.Lock()
	defer mu.Unlock()
	for _, ins := range instances {
		ins.close()
	}
	instances = map[string]*Instance{}
	log.Info("all database connections closed")
}

// 关闭全部数据库实例连接（CloseAll 的别名）。
func Close() {
	CloseAll()
}

// ==================== 实例方法 ====================

// 实例是否可用。
// 出参: 数据库是否已初始化
func (i *Instance) IsAvailable() bool {
	return i != nil && i.db != nil
}

// 实例的 gorm 连接，未初始化时返回 nil。
// 出参: gorm 数据库连接
func (i *Instance) GetDB() *gorm.DB {
	if i == nil {
		return nil
	}
	return i.db
}

// 实例数据库类型（sqlite3/mysql/...）。
// 出参: 数据库类型
func (i *Instance) Type() string {
	return i.dbType
}

// 在实例上执行事务。
// 入参: fn (事务函数，参数为事务连接)
// 出参: 事务执行错误
func (i *Instance) Transaction(fn TransactionFunc) error {
	if i == nil || i.db == nil {
		return ErrDBUnavailable
	}
	return i.db.Transaction(fn)
}

// 获取实例连接（支持事务）：tx 不为 nil 时使用事务，否则使用实例连接。
// 入参: tx (事务连接，可为 nil)
// 出参: 实际使用的数据库连接
func (i *Instance) GetDBWithTx(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return i.db
}

// 在实例上自动迁移（带表引擎选项），仅迁移本实例，不影响全局注册表。
// 入参: dst (模型指针列表)
// 出参: 迁移错误
func (i *Instance) AutoMigrate(dst ...any) error {
	return i.migrate(dst...)
}

func (i *Instance) migrate(dst ...any) error {
	if i == nil || i.db == nil {
		return ErrDBUnavailable
	}
	if i.dbType == "mysql" {
		return i.db.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4").AutoMigrate(dst...)
	}
	return i.db.AutoMigrate(dst...)
}

// 实例的列名引用兼容处理。
// 入参: name (列名)
// 出参: 带引用符的列名
func (i *Instance) ColumnName(name string) string {
	if i.dbType == "postgres" || i.dbType == "kingbase" {
		return fmt.Sprintf(`"%s"`, name)
	}
	return fmt.Sprintf("`%s`", name)
}

// 关闭实例连接（不操作时注册表）。
func (i *Instance) Close() {
	i.close()
}

func (i *Instance) close() {
	if i.db == nil {
		return
	}
	sqlDB, err := i.db.DB()
	if err != nil {
		log.Errorf("failed to get database instance: %s", err.Error())
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Errorf("failed to close database: %s", err.Error())
	}
	i.db = nil
}
