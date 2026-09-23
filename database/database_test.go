package database

import (
	"testing"

	"gopkg.761sama.com/tenon/conf"
)

type testUser struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:64"`
}

// 验证 sqlite 初始化、RegisterModels 自动迁移与基本读写。
func TestInitWithSQLite(t *testing.T) {
	cfg := conf.Default()
	cfg.DataDir = t.TempDir()
	cfg.Database.Type = "sqlite3"
	cfg.Database.Master.DBFile = "test.db"
	RegisterModels(new(testUser))
	Init(cfg)
	defer Close()
	if GetDB() == nil {
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
	cfg := conf.Default()
	cfg.DataDir = t.TempDir()
	cfg.Database.Type = "sqlite3"
	cfg.Database.Master.DBFile = "tx.db"
	RegisterModels(new(testUser))
	Init(cfg)
	defer Close()
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
	cfg := conf.Default()
	cfg.DataDir = t.TempDir()
	cfg.Database.Type = "sqlite3"
	cfg.Database.Master.DBFile = "late.db"
	Init(cfg)
	defer Close()
	type lateModel struct {
		ID uint `gorm:"primaryKey"`
	}
	RegisterModels(new(lateModel))
	if !GetDB().Migrator().HasTable(&lateModel{}) {
		t.Fatal("late registered model should be migrated immediately")
	}
}

// 验证列名引用兼容。
func TestColumnName(t *testing.T) {
	if got := ColumnName("user"); got != "`user`" {
		t.Fatalf("unexpected column name: %s", got)
	}
}
