package log

import (
	"log"
)

type Priority int

const (
	EMERG Priority = iota
	ALERT
	CRIT
	ERROR
	WARN
	NOTICE
	INFO
	DEBUG
)

func (p Priority) String() string {
	switch p {
	case EMERG:
		return "Emergency"
	case ALERT:
		return "Alert"
	case CRIT:
		return "Critical"
	case ERROR:
		return "Error"
	case WARN:
		return "Warning"
	case NOTICE:
		return "Notice"
	case INFO:
		return "Information"
	}
	return "Debug"
}

type Logger struct {
	internal *log.Logger
	level    Priority
}

var gLogger *Logger

func init() {
	gLogger = newLogger()
}

func Get() *Logger {
	return gLogger
}

func newLogger() *Logger {
	l := Logger{}
	l.internal = nil
	l.level = EMERG
	return &l
}

func Set(l *log.Logger, lv Priority) {
	gLogger.internal = l
	gLogger.level = lv
}

func SetLogger(l *log.Logger) {
	gLogger.internal = l
}

func SetLevel(lv Priority) {
	gLogger.level = lv
}

func GetLogger() *log.Logger {
	return gLogger.internal
}

func GetLevel() Priority {
	return gLogger.level
}

func (l *Logger) write(lv Priority, msg string, v ...interface{}) {
	if lv > l.level {
		return
	}

	if l.internal == nil {
		return
	}

	if v == nil || len(v) == 0 {
		l.internal.Print(msg)
	} else {
		l.internal.Printf(msg, v...)
	}
}

func (l *Logger) Debug(msg string, v ...interface{}) {
	l.write(DEBUG, msg, v...)
}

func (l *Logger) Info(msg string, v ...interface{}) {
	l.write(INFO, msg, v...)
}

func (l *Logger) Notice(msg string, v ...interface{}) {
	l.write(NOTICE, msg, v...)
}

func (l *Logger) Warn(msg string, v ...interface{}) {
	l.write(WARN, msg, v...)
}

func (l *Logger) Error(msg string, v ...interface{}) {
	l.write(ERROR, msg, v...)
}

func (l *Logger) Crit(msg string, v ...interface{}) {
	l.write(CRIT, msg, v...)
}

func (l *Logger) Alert(msg string, v ...interface{}) {
	l.write(ALERT, msg, v...)
}

func (l *Logger) Emerg(msg string, v ...interface{}) {
	l.write(EMERG, msg, v...)
}
