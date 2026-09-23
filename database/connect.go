package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"github.com/glebarez/sqlite"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/plugin/dbresolver"
	mysqldriver "github.com/go-sql-driver/mysql"

	"gopkg.761sama.com/tenon/conf"
	"gopkg.761sama.com/tenon/util"
)

// 按配置建立数据库连接（含主从）。
// 入参: cfg (总配置)
// 出参: gorm 数据库连接与错误
func connect(cfg conf.Config) (*gorm.DB, error) {
	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}
	masterDialector, err := buildDialector(cfg, cfg.Database.Master)
	if err != nil {
		return nil, fmt.Errorf("failed to build master database dialector: %w", err)
	}
	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: cfg.Database.Master.TablePrefix,
		},
		Logger: logger.Default.LogMode(logLevel),
	}
	masterDB, err := gorm.Open(masterDialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect master database: %w", err)
	}
	if len(cfg.Database.Replicas) > 0 {
		var replicaDialectors []gorm.Dialector
		for _, replica := range cfg.Database.Replicas {
			d, err := buildDialector(cfg, replica)
			if err != nil {
				continue
			}
			replicaDialectors = append(replicaDialectors, d)
		}
		if len(replicaDialectors) > 0 {
			if err := masterDB.Use(dbresolver.Register(dbresolver.Config{
				Replicas: replicaDialectors,
				Policy:   dbresolver.RandomPolicy{},
			})); err != nil {
				return nil, fmt.Errorf("failed to register db replicas: %w", err)
			}
		}
	}
	return masterDB, nil
}

// 构建单个节点的数据库 dialector。
// 入参: cfg (总配置), node (节点配置)
// 出参: dialector 与错误
func buildDialector(cfg conf.Config, node conf.DBNodeConfig) (gorm.Dialector, error) {
	switch cfg.Database.Type {
	case "sqlite3", "sqlite":
		if !(strings.HasSuffix(node.DBFile, ".db") && len(node.DBFile) > 3) {
			return nil, fmt.Errorf("database file name must end with .db")
		}
		dbFile := util.ResolveDataPath(cfg.DataDir, node.DBFile)
		dir := filepath.Dir(dbFile)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
		return sqlite.Open(fmt.Sprintf("%s?_journal=WAL&_vacuum=incremental", dbFile)), nil
	case "mysql":
		port := node.Port
		if port <= 0 {
			port = 3306
		}
		host := strings.TrimSpace(node.Host)
		if host == "" {
			return nil, fmt.Errorf("mysql host is required")
		}
		name := strings.TrimSpace(node.Name)
		if name == "" {
			return nil, fmt.Errorf("mysql database name is required")
		}
		mysqlConf := mysqldriver.NewConfig()
		mysqlConf.User = node.User
		mysqlConf.Passwd = node.Password
		mysqlConf.Net = "tcp"
		mysqlConf.Addr = fmt.Sprintf("%s:%d", host, port)
		mysqlConf.DBName = name
		mysqlConf.ParseTime = true
		mysqlConf.Loc = time.Local
		mysqlConf.Params = map[string]string{"charset": "utf8mb4"}
		return gormmysql.Open(mysqlConf.FormatDSN()), nil
	case "postgres":
		return nil, fmt.Errorf("postgres driver not yet implemented")
	case "kingbase":
		return nil, fmt.Errorf("kingbase driver not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}
}
