package logger

// import logrus
import (
	"fmt"
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
	guiLogsCh chan string
}

func New(guiLogsCh chan string) *Logger {
	log := logrus.New()
	format := &logrus.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	}

	// set logs output to file if GUI is enabled
	if os.Getenv("GUI") == "1" {
		file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
			format.DisableColors = true
		} else {
			log.Fatal(err)
		}
	} else {
		format.ForceColors = true
	}

	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	log.SetFormatter(format)

	return &Logger{log, guiLogsCh}
}

func (l *Logger) Close() error {
	if file, ok := l.Out.(*os.File); ok {
		return file.Close()
	}
	return nil
}

// TODO: write to websockets

func (l *Logger) sendToGUI(t level, args ...interface{}) {
	if l.guiLogsCh != nil {
		msg := fmt.Sprintf("%s: ", t)
		msg += fmt.Sprint(args...)
		l.guiLogsCh <- msg
	}
}

func (l *Logger) sendToGUIf(t level, format string, args ...interface{}) {
	if l.guiLogsCh != nil {
		msg := fmt.Sprintf("%s: ", t)
		msg += fmt.Sprintf(format, args...)
		l.guiLogsCh <- msg
	}
}

// debug
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
	l.sendToGUI("DEBUG", args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
	l.sendToGUIf("DEBUG", format, args...)
}

// info
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
	l.sendToGUI(Info, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
	l.sendToGUIf(Info, format, args...)
}

// warn
func (l *Logger) Warn(args ...interface{}) {
	l.Logger.Warn(args...)
	l.sendToGUI(Warn, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warnf(format, args...)
	l.sendToGUIf(Warn, format, args...)
}

// error
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
	l.sendToGUI(Error, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Errorf(format, args...)
	l.sendToGUIf(Error, format, args...)
}

// fatal
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
	l.sendToGUI(Fatal, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatalf(format, args...)
	l.sendToGUIf(Fatal, format, args...)
}
