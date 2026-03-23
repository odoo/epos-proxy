package logger

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log = logrus.New()

var logDir string

func InitLogger() {
	dir, err := os.UserConfigDir()
	if err != nil {
		Log.Fatalf("Failed to get user config dir: %v", err)
	}
	logDir = filepath.Join(dir, "epos-proxy", "logs")
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

// Wrappers
func Info(args ...interface{})                  { Log.Info(args...) }
func Infof(format string, args ...interface{})  { Log.Infof(format, args...) }
func Warn(args ...interface{})                  { Log.Warn(args...) }
func Warnf(format string, args ...interface{})  { Log.Warnf(format, args...) }
func Error(args ...interface{})                 { Log.Error(args...) }
func Errorf(format string, args ...interface{}) { Log.Errorf(format, args...) }
func Fatal(args ...interface{})                 { Log.Fatal(args...) }
func Fatalf(format string, args ...interface{}) { Log.Fatalf(format, args...) }
func Debug(args ...interface{})                 { Log.Debug(args...) }
func Debugf(format string, args ...interface{}) { Log.Debugf(format, args...) }

func LogDirectory() string { return logDir }
