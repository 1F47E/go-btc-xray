package logger

// import logrus
import (
	"fmt"
	"go-btc-downloader/pkg/gui"
	"os"

	"github.com/sirupsen/logrus"
)

const logfile = "logs.log"

type level string

const (
	Debug level = "DEBUG"
	Info  level = "INFO"
	Warn  level = "WARN"
	Error level = "ERROR"
	Fatal level = "FATAL"
)

type Logger struct {
	*logrus.Logger
	guiCh chan gui.IncomingData
}

func New(guiCh chan gui.IncomingData) *Logger {

	log := initLogger()
	return &Logger{log, guiCh}
}

func initLogger() *logrus.Logger {

	log := logrus.New()

	var format logrus.TextFormatter
	if os.Getenv("GUI") == "1" {
		file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
			format.DisableColors = true
			format.DisableTimestamp = false
		} else {
			log.Fatal(err)
		}
		log.SetFormatter(&format)
	} else {
		var format logrus.TextFormatter
		format.ForceColors = true
		format.DisableTimestamp = true
		log.Out = os.Stdout
		log.SetFormatter(&format)
	}

	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	return log
}

func (l *Logger) ResetToStdout() {
	os.Setenv("GUI", "0")
	l.Logger = initLogger()
}

func (l *Logger) Close() error {
	if file, ok := l.Out.(*os.File); ok {
		return file.Close()
	}
	return nil
}

// ===== logrus wrapper

// debug
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
	l.Ship("DEBUG", args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
	l.Shipf("DEBUG", format, args...)
}

// info
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
	l.Ship(Info, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
	l.Shipf(Info, format, args...)
}

// warn
func (l *Logger) Warn(args ...interface{}) {
	l.Logger.Warn(args...)
	l.Ship(Warn, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warnf(format, args...)
	l.Shipf(Warn, format, args...)
}

// error
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
	l.Ship(Error, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Errorf(format, args...)
	l.Shipf(Error, format, args...)
}

// fatal
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
	l.Ship(Fatal, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatalf(format, args...)
	l.Shipf(Fatal, format, args...)
}

// ===== Ship logs to the chan to be displayed in the GUI

func (l *Logger) Ship(t level, args ...interface{}) {
	msg := fmt.Sprintf("%s: ", t)
	msg += fmt.Sprint(args...)
	l.ship(msg)
}

func (l *Logger) Shipf(t level, format string, args ...interface{}) {
	msg := fmt.Sprintf("%s: ", t)
	msg += fmt.Sprintf(format, args...)
	l.ship(msg)
}

// ship to gui logs chan if it's not full
func (l *Logger) ship(msg string) {
	if l.guiCh != nil && len(l.guiCh) < cap(l.guiCh) {
		l.guiCh <- gui.IncomingData{Log: msg}
	}
}
