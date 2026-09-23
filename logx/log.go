package logx

import (
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/natefinch/lumberjack"

	"gopkg.761sama.com/tenon/bootstrap"
	"gopkg.761sama.com/tenon/conf"
	"gopkg.761sama.com/tenon/util"
)

func init() {
	formatter := log.TextFormatter{
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		TimestampFormat:           "2006-01-02 15:04:05",
		FullTimestamp:             true,
	}
	log.SetFormatter(&formatter)
	bootstrap.RegisterInitModule("log", Init)
}

// 初始化日志模块：按配置设置日志级别与文件切割输出。
// 入参: cfg (总配置)
func Init(cfg conf.Config) {
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	} else {
		log.SetLevel(log.InfoLevel)
		log.SetReportCaller(false)
	}
	if cfg.Log.Enable {
		logFile := util.ResolveDataPath(cfg.DataDir, cfg.Log.Name)
		dir := filepath.Dir(logFile)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			log.Errorf("failed to create log directory: %s", err.Error())
			return
		}
		var w io.Writer = &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    cfg.Log.MaxSize,
			MaxBackups: cfg.Log.MaxBackups,
			MaxAge:     cfg.Log.MaxAge,
			Compress:   cfg.Log.Compress,
		}
		if cfg.Debug {
			w = io.MultiWriter(os.Stdout, w)
		}
		log.SetOutput(w)
	}
	log.Infof("init logrus...")
}
