package logx

import (
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/natefinch/lumberjack"

	"gopkg.761sama.com/tenon/conf"
)

func init() {
	formatter := log.TextFormatter{
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		TimestampFormat:           "2006-01-02 15:04:05",
		FullTimestamp:             true,
	}
	log.SetFormatter(&formatter)
}

// 显式初始化日志模块：设置日志级别与文件切割输出；不调用时默认输出到标准错误。
// 入参: cfg (日志配置), debug (是否调试模式)
// 出参: 日志目录创建失败时返回错误
func Init(cfg conf.LogConfig, debug bool) error {
	if debug {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	} else {
		log.SetLevel(log.InfoLevel)
		log.SetReportCaller(false)
	}
	if cfg.Enable {
		dir := filepath.Dir(cfg.Name)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
		var w io.Writer = &lumberjack.Logger{
			Filename:   cfg.Name,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		if debug {
			w = io.MultiWriter(os.Stdout, w)
		}
		log.SetOutput(w)
	}
	log.Infof("init logrus...")
	return nil
}
