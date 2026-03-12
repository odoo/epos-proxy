package logger

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log = logrus.New()

func InitLogger() {
	dir, _ := os.UserConfigDir()
	logDir := filepath.Join(dir, "epos-proxy", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		Log.Fatalf("Failed to create log directory: %v", err)
	}

	filename := filepath.Join(logDir, "epos-proxy.log")

	Log.SetOutput(&lumberjack.Logger{
		Filename:   filename,
		MaxSize:    20, // MB
		MaxBackups: 5,  // keep last x files
		MaxAge:     5,  // days
		Compress:   false,
	})

	Log.SetReportCaller(true)

	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	Log.SetLevel(logrus.InfoLevel)

	Log.Info("Logger initialized at ", time.Now())
}
