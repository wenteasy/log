package log

import (
	"fmt"
	"log"
)

type Priority int

const (
	DEBUG Priority = iota
	INFO
	NOTICE
	WARN
	ERROR
	CRIT
	ALERT
	EMERG
)

func (p Priority) GE(v Priority) bool {
	return p >= v
}

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

var gLogger *Logger

type Logger struct {
	normal *logger
	err    *logger
}

func init() {
	gLogger = newDefaultLogger()
}

func Get() *Logger {
	return gLogger
}

func SetLevel(lv Priority) {
	gLogger.normal.level = lv
}

func GetLevel() Priority {
	return gLogger.normal.level
}

func SetErrorLevel(lv Priority) {
	gLogger.err.level = lv
}

func GetErrorLevel() Priority {
	return gLogger.err.level
}

func newDefaultLogger() *Logger {
	l := Logger{}
	l.normal = newLogger(log.Default(), EMERG)
	l.err = newLogger(nil, EMERG)
	return &l
}

func newLogger(l *log.Logger, lv Priority) *logger {
	var rtn logger
	rtn.body = l
	rtn.level = lv
	return &rtn
}

func Set(l *log.Logger, lv Priority) {
	lgg := newLogger(l, lv)
	gLogger.normal = lgg
}

func SetError(l *log.Logger, lv Priority) {
	gLogger.err = newLogger(l, lv)
}

func (l *Logger) write(lv Priority, msg string, v ...interface{}) {

	if !l.normal.isOutput(lv) {
		return
	}

	lgg := l.normal
	if l.err.isOutput(lv) {
		lgg = l.err
	}

	lgg.printf(msg, v...)
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

type logger struct {
	body  *log.Logger
	level Priority
}

func (l *logger) isEmpty() bool {
	if l.body == nil {
		return true
	}
	return false
}

func (l *logger) isOutput(lv Priority) bool {
	if l.isEmpty() {
		return false
	}
	return lv.GE(l.level)
}

func (l *logger) String() string {
	rtn := "not been set"
	if !l.isEmpty() {
		rtn = fmt.Sprintf("Writer %T|Level=%v", l.body.Writer(), l.level)
	}
	return rtn
}

func (l *logger) printf(msg string, v ...interface{}) {
	l.body.Printf(msg, v...)
}
