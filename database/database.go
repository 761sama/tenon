package database

import (
	"fmt"
	"sync"

	"gorm.io/gorm"
	log "github.com/sirupsen/logrus"

	"gopkg.761sama.com/tenon/bootstrap"
	"gopkg.761sama.com/tenon/conf"
)

// TransactionFunc 事务函数类型。
type TransactionFunc func(tx *gorm.DB) error

// Tx 事务连接类型（无需引用 gorm 包）。
type Tx = *gorm.DB

var (
	mu               sync.RWMutex
	db               *gorm.DB
	registeredModels []any  // 已注册待迁移的模型
	dbType           string // 当前数据库类型
)

// 显式初始化数据库：按配置连接主库与从库，自动迁移已注册模型；初始化成功后注册停机释放。
// 入参: cfg (数据库配置，Type 为空时返回错误), debug (是否打印 SQL 日志)
// 出参: 连接或迁移失败时返回错误
func Init(cfg conf.DatabaseConfig, debug bool) error {
	if cfg.Type == "" {
		return fmt.Errorf("database type is empty")
	}
	gormDB, err := connect(cfg, debug)
	if err != nil {
		return err
	}
	mu.Lock()
	db = gormDB
	dbType = cfg.Type
	models := registeredModels
	mu.Unlock()
	if len(models) > 0 {
		if err := autoMigrate(gormDB, cfg.Type, models...); err != nil {
			return fmt.Errorf("failed to auto migrate database: %w", err)
		}
	}
	bootstrap.RegisterRelease("database", Close)
	return nil
}

// 注册模型，模型将在数据库初始化时自动迁移；数据库已初始化时立即迁移。
// 入参: models (模型指针列表，如 new(User))
func RegisterModels(models ...any) {
	mu.Lock()
	defer mu.Unlock()
	registeredModels = append(registeredModels, models...)
	if db != nil {
		if err := autoMigrate(db, dbType, models...); err != nil {
			log.Errorf("failed to auto migrate registered models: %s", err.Error())
		}
	}
}

// 数据库是否可用。
// 出参: 数据库是否已初始化
func IsAvailable() bool {
	mu.RLock()
	defer mu.RUnlock()
	return db != nil
}

// 外部调用 DB，未初始化时返回 nil。
// 出参: gorm 数据库连接
func GetDB() *gorm.DB {
	mu.RLock()
	defer mu.RUnlock()
	return db
}

// 执行事务。
// 入参: fn (事务函数，参数为事务连接)
// 出参: 事务执行错误
func Transaction(fn TransactionFunc) error {
	return GetDB().Transaction(fn)
}

// 获取数据库连接（支持事务）：tx 不为 nil 时使用事务，否则使用默认连接。
// 入参: tx (事务连接，可为 nil)
// 出参: 实际使用的数据库连接
func GetDBWithTx(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return GetDB()
}

// 自动迁移（带表引擎选项）。
// 入参: dst (模型指针列表)
// 出参: 迁移错误
func AutoMigrate(dst ...any) error {
	mu.RLock()
	defer mu.RUnlock()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	return autoMigrate(db, dbType, dst...)
}

func autoMigrate(gormDB *gorm.DB, typ string, dst ...any) error {
	if typ == "mysql" {
		return gormDB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4").AutoMigrate(dst...)
	}
	return gormDB.AutoMigrate(dst...)
}

// 关闭数据库连接。
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if db == nil {
		return
	}
	log.Info("closing database connection")
	sqlDB, err := db.DB()
	if err != nil {
		log.Errorf("failed to get database instance: %s", err.Error())
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Errorf("failed to close database: %s", err.Error())
	}
	db = nil
}
