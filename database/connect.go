package database

import (
	"database/sql"
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
)

// 按配置建立数据库连接（含主从）。
// 入参: cfg (数据库配置), debug (是否打印 SQL 日志)
// 出参: gorm 数据库连接与错误
func connect(cfg conf.DatabaseConfig, debug bool) (*gorm.DB, error) {
	logLevel := logger.Silent
	if debug {
		logLevel = logger.Info
	}
	masterDialector, err := buildDialector(cfg.Type, cfg.Master)
	if err != nil {
		return nil, fmt.Errorf("failed to build master database dialector: %w", err)
	}
	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: cfg.Master.TablePrefix,
		},
		Logger: logger.Default.LogMode(logLevel),
	}
	masterDB, err := gorm.Open(masterDialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect master database: %w", err)
	}
	// 配置主库连接池：优先使用配置值，未配置时按数据库类型取默认值
	if sqlDB, err := masterDB.DB(); err != nil {
		return nil, fmt.Errorf("failed to get master database instance for pool config: %w", err)
	} else {
		applyPool(sqlDB, cfg.Type, cfg.Master)
	}
	if len(cfg.Replicas) > 0 {
		var replicaDialectors []gorm.Dialector
		for _, replica := range cfg.Replicas {
			d, err := buildDialector(cfg.Type, replica)
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

// 应用连接池配置：配置值优先，未配置时按数据库类型取默认值。
// SQLite 为单写库，默认限制单连接避免 database is locked。
// 入参: sqlDB (底层连接池), dbType (数据库类型), node (节点配置)
func applyPool(sqlDB *sql.DB, dbType string, node conf.DBNodeConfig) {
	switch dbType {
	case "sqlite3", "sqlite":
		maxOpen := node.MaxOpenConns
		if maxOpen <= 0 {
			maxOpen = 1
		}
		sqlDB.SetMaxOpenConns(maxOpen)
		if node.MaxIdleConns > 0 {
			sqlDB.SetMaxIdleConns(node.MaxIdleConns)
		}
	case "mysql":
		maxOpen := node.MaxOpenConns
		if maxOpen <= 0 {
			maxOpen = 50
		}
		maxIdle := node.MaxIdleConns
		if maxIdle <= 0 {
			maxIdle = 10
		}
		lifetime := node.ConnMaxLifetime
		if lifetime <= 0 {
			lifetime = time.Hour
		}
		idleTime := node.ConnMaxIdleTime
		if idleTime <= 0 {
			idleTime = 10 * time.Minute
		}
		sqlDB.SetMaxOpenConns(maxOpen)
		sqlDB.SetMaxIdleConns(maxIdle)
		sqlDB.SetConnMaxLifetime(lifetime)
		sqlDB.SetConnMaxIdleTime(idleTime)
	default:
		if node.MaxOpenConns > 0 {
			sqlDB.SetMaxOpenConns(node.MaxOpenConns)
		}
		if node.MaxIdleConns > 0 {
			sqlDB.SetMaxIdleConns(node.MaxIdleConns)
		}
		if node.ConnMaxLifetime > 0 {
			sqlDB.SetConnMaxLifetime(node.ConnMaxLifetime)
		}
		if node.ConnMaxIdleTime > 0 {
			sqlDB.SetConnMaxIdleTime(node.ConnMaxIdleTime)
		}
	}
}

// 构建单个节点的数据库 dialector。
// 入参: dbType (数据库类型), node (节点配置)
// 出参: dialector 与错误
func buildDialector(dbType string, node conf.DBNodeConfig) (gorm.Dialector, error) {
	switch dbType {
	case "sqlite3", "sqlite":
		if !(strings.HasSuffix(node.DBFile, ".db") && len(node.DBFile) > 3) {
			return nil, fmt.Errorf("database file name must end with .db")
		}
		dir := filepath.Dir(node.DBFile)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
		return sqlite.Open(fmt.Sprintf("%s?_journal=WAL&_vacuum=incremental", node.DBFile)), nil
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
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
