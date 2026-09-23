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
	registeredModels []any      // 已注册待迁移的模型
	dbType           string     // 当前数据库类型
)

func init() {
	bootstrap.RegisterInitModule("database", Init)
	bootstrap.RegisterRelease("database", Close)
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

// 初始化数据库模块（由 bootstrap 调用），Type 为空时跳过。
// 入参: cfg (总配置)
func Init(cfg conf.Config) {
	if cfg.Database.Type == "" {
		return
	}
	gormDB, err := connect(cfg)
	if err != nil {
		log.Fatalf("failed to init database: %s", err.Error())
	}
	mu.Lock()
	db = gormDB
	dbType = cfg.Database.Type
	models := registeredModels
	mu.Unlock()
	if len(models) > 0 {
		if err := autoMigrate(gormDB, cfg.Database.Type, models...); err != nil {
			log.Fatalf("failed to auto migrate database: %s", err.Error())
		}
	}
}
