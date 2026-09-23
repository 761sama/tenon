package database

import (
	"errors"
	"path/filepath"
	"testing"

	"gopkg.761sama.com/tenon/conf"
)

type testUser struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:64"`
}

// 初始化 sqlite 测试库。
// 入参: file (数据库文件名)
func setupSQLite(t *testing.T, file string) {
	t.Helper()
	cfg := conf.DatabaseConfig{
		Type:   "sqlite3",
		Master: conf.DBNodeConfig{DBFile: filepath.Join(t.TempDir(), file)},
	}
	if err := Init(cfg, false); err != nil {
		t.Fatalf("failed to init sqlite: %s", err)
	}
	t.Cleanup(Close)
}

// 验证空数据库类型拒绝初始化。
func TestInitEmptyType(t *testing.T) {
	if err := Init(conf.DatabaseConfig{}, false); err == nil {
		t.Fatal("empty database type should be rejected")
	}
}

// 验证 sqlite 初始化、RegisterModels 自动迁移与基本读写。
func TestInitWithSQLite(t *testing.T) {
	RegisterModels(new(testUser))
	setupSQLite(t, "test.db")
	if !IsAvailable() || GetDB() == nil {
		t.Fatal("db should be initialized")
	}
	u := testUser{Name: "tenon"}
	if err := GetDB().Create(&u).Error; err != nil {
		t.Fatalf("failed to create: %s", err)
	}
	var got testUser
	if err := GetDB().First(&got, u.ID).Error; err != nil {
		t.Fatalf("failed to query: %s", err)
	}
	if got.Name != "tenon" {
		t.Fatalf("unexpected name: %s", got.Name)
	}
}

// 验证事务封装。
func TestTransaction(t *testing.T) {
	setupSQLite(t, "tx.db")
	err := Transaction(func(tx Tx) error {
		return tx.Create(&testUser{Name: "in-tx"}).Error
	})
	if err != nil {
		t.Fatalf("transaction failed: %s", err)
	}
	var count int64
	GetDB().Model(&testUser{}).Where("name = ?", "in-tx").Count(&count)
	if count != 1 {
		t.Fatalf("unexpected count: %d", count)
	}
}

// 验证数据库已初始化后注册模型会立即迁移。
func TestRegisterModelsAfterInit(t *testing.T) {
	setupSQLite(t, "late.db")
	type lateModel struct {
		ID uint `gorm:"primaryKey"`
	}
	RegisterModels(new(lateModel))
	if !GetDB().Migrator().HasTable(&lateModel{}) {
		t.Fatal("late registered model should be migrated immediately")
	}
}

// 验证 MySQL 初始化、自动迁移、读写与事务（连接本地测试库）。
func TestMySQL(t *testing.T) {
	cfg := conf.DatabaseConfig{
		Type: "mysql",
		Master: conf.DBNodeConfig{
			Host:     "127.0.0.1",
			Port:     3306,
			User:     "testuser",
			Password: "test123456",
			Name:     "testdb",
		},
	}
	if err := Init(cfg, false); err != nil {
		t.Skipf("local mysql unavailable, skip: %s", err)
	}
	defer Close()
	if Default().Type() != "mysql" {
		t.Fatalf("unexpected db type: %s", Default().Type())
	}
	type mysqlProbe struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"size:64"`
	}
	if err := AutoMigrate(&mysqlProbe{}); err != nil {
		t.Fatalf("failed to migrate: %s", err)
	}
	defer GetDB().Migrator().DropTable(&mysqlProbe{})
	p := mysqlProbe{Name: "mysql-ok"}
	if err := GetDB().Create(&p).Error; err != nil {
		t.Fatalf("failed to create: %s", err)
	}
	var got mysqlProbe
	if err := GetDB().First(&got, p.ID).Error; err != nil {
		t.Fatalf("failed to query: %s", err)
	}
	if got.Name != "mysql-ok" {
		t.Fatalf("unexpected name: %s", got.Name)
	}
	err := Transaction(func(tx Tx) error {
		return tx.Create(&mysqlProbe{Name: "mysql-tx"}).Error
	})
	if err != nil {
		t.Fatalf("transaction failed: %s", err)
	}
	var count int64
	GetDB().Model(&mysqlProbe{}).Where("name = ?", "mysql-tx").Count(&count)
	if count != 1 {
		t.Fatalf("unexpected count: %d", count)
	}
}

// 验证多实例：默认实例与命名实例（独立 sqlite 文件）数据隔离。
func TestMultiInstance(t *testing.T) {
	setupSQLite(t, "first.db")
	dir := t.TempDir()
	cfg := conf.DatabaseConfig{
		Type:   "sqlite3",
		Master: conf.DBNodeConfig{DBFile: filepath.Join(dir, "second.db")},
	}
	if err := InitNamed("second", cfg, false); err != nil {
		t.Fatalf("failed to init named instance: %s", err)
	}
	second := Named("second")
	if second == nil || !second.IsAvailable() {
		t.Fatal("named instance should be available")
	}
	if err := second.AutoMigrate(&testUser{}); err != nil {
		t.Fatalf("failed to migrate on named instance: %s", err)
	}
	if err := GetDB().Create(&testUser{Name: "in-default"}).Error; err != nil {
		t.Fatalf("default create failed: %s", err)
	}
	var defaultCount, secondCount int64
	GetDB().Model(&testUser{}).Where("name = ?", "in-default").Count(&defaultCount)
	second.GetDB().Model(&testUser{}).Where("name = ?", "in-default").Count(&secondCount)
	if defaultCount != 1 || secondCount != 0 {
		t.Fatalf("instances should be isolated: %d, %d", defaultCount, secondCount)
	}
	if Named("not-exists") != nil {
		t.Fatal("unknown instance should return nil")
	}
}

// 验证未初始化实例的操作返回 ErrDBUnavailable。
func TestUnavailable(t *testing.T) {
	CloseAll()
	if err := Default().Transaction(func(tx Tx) error { return nil }); !errors.Is(err, ErrDBUnavailable) {
		t.Fatalf("expected ErrDBUnavailable, got: %v", err)
	}
	var empty *Instance
	if err := empty.Transaction(func(tx Tx) error { return nil }); !errors.Is(err, ErrDBUnavailable) {
		t.Fatalf("nil instance should return ErrDBUnavailable, got: %v", err)
	}
}

// 验证连接池配置：sqlite 默认单连接，可显式覆盖。
func TestPoolConfigSQLite(t *testing.T) {
	setupSQLite(t, "pool.db")
	sqlDB, err := GetDB().DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %s", err)
	}
	if sqlDB.Stats().MaxOpenConnections != 1 {
		t.Fatalf("sqlite should default to single connection: %d", sqlDB.Stats().MaxOpenConnections)
	}
	CloseAll()
	cfg := conf.DatabaseConfig{
		Type:   "sqlite3",
		Master: conf.DBNodeConfig{DBFile: filepath.Join(t.TempDir(), "pool5.db"), MaxOpenConns: 5},
	}
	if err := Init(cfg, false); err != nil {
		t.Fatalf("failed to init sqlite: %s", err)
	}
	sqlDB, _ = GetDB().DB()
	if sqlDB.Stats().MaxOpenConnections != 5 {
		t.Fatalf("explicit MaxOpenConns should be respected: %d", sqlDB.Stats().MaxOpenConnections)
	}
}

// 验证 MySQL 连接池：默认 50，可显式覆盖。
func TestPoolConfigMySQL(t *testing.T) {
	cfg := conf.DatabaseConfig{
		Type: "mysql",
		Master: conf.DBNodeConfig{
			Host: "127.0.0.1", Port: 3306, User: "testuser", Password: "test123456", Name: "testdb",
		},
	}
	if err := Init(cfg, false); err != nil {
		t.Skipf("local mysql unavailable, skip: %s", err)
	}
	sqlDB, err := GetDB().DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %s", err)
	}
	if sqlDB.Stats().MaxOpenConnections != 50 {
		t.Fatalf("mysql should default to 50 connections: %d", sqlDB.Stats().MaxOpenConnections)
	}
	CloseAll()
	cfg.Master.MaxOpenConns = 20
	cfg.Master.MaxIdleConns = 5
	if err := Init(cfg, false); err != nil {
		t.Fatalf("failed to reinit mysql: %s", err)
	}
	defer CloseAll()
	sqlDB, _ = GetDB().DB()
	if sqlDB.Stats().MaxOpenConnections != 20 {
		t.Fatalf("explicit MaxOpenConns should be respected: %d", sqlDB.Stats().MaxOpenConnections)
	}
}

// 验证列名引用兼容。
func TestColumnName(t *testing.T) {
	if got := ColumnName("user"); got != "`user`" {
		t.Fatalf("unexpected column name: %s", got)
	}
}
