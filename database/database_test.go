package database

import (
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
	if dbType != "mysql" {
		t.Fatalf("unexpected db type: %s", dbType)
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

// 验证列名引用兼容。
func TestColumnName(t *testing.T) {
	if got := ColumnName("user"); got != "`user`" {
		t.Fatalf("unexpected column name: %s", got)
	}
}
