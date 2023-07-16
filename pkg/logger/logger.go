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
	log := logrus.New()
	// format := &logrus.TextFormatter{}
	var format logrus.TextFormatter

	// set logs output to file if GUI is enabled
	if os.Getenv("GUI") != "0" {
		file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
			format.DisableColors = true
			format.DisableTimestamp = false
		} else {
			log.Fatal(err)
		}
	} else {
		format.ForceColors = true
		format.DisableTimestamp = true
	}

	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	log.SetFormatter(&format)

	return &Logger{log, guiCh}
}

func (l *Logger) Close() error {
	if file, ok := l.Out.(*os.File); ok {
		return file.Close()
	}
	return nil
}

// TODO: write to websockets
func (l *Logger) Ship(t level, args ...interface{}) {
	if l.guiCh != nil {
		msg := fmt.Sprintf("%s: ", t)
		msg += fmt.Sprint(args...)
		l.guiCh <- gui.IncomingData{Log: msg}
	}
}

func (l *Logger) Shipf(t level, format string, args ...interface{}) {
	if l.guiCh != nil {
		msg := fmt.Sprintf("%s: ", t)
		msg += fmt.Sprintf(format, args...)
		l.guiCh <- gui.IncomingData{Log: msg}
	}
}

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
