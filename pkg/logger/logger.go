package logger

// import logrus
import (
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func New() *Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	})

	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	return &Logger{log}
}

// TODO: write to websockets

// debug
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
}

// info
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
}

// warn
func (l *Logger) Warn(args ...interface{}) {
	l.Logger.Warn(args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warnf(format, args...)
}

// error
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Errorf(format, args...)
}

// fatal
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatalf(format, args...)
}
