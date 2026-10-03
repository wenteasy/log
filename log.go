package log

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
)

const (
	LevelTrace     slog.Level = -8
	LevelDebug                = slog.LevelDebug
	LevelInfo                 = slog.LevelInfo
	LevelNotice    slog.Level = 2
	LevelWarn                 = slog.LevelWarn
	LevelError                = slog.LevelError
	LevelEmergency slog.Level = 32
)

var gCtx context.Context
var gLogger *slog.Logger
var logLevel slog.LevelVar

func init() {
	gCtx = context.Background()
	logLevel.Set(LevelInfo)
	gLogger = slog.Default()
}

func SetDefault(l *slog.Logger) {
	gLogger = l
	slog.SetDefault(l)
}

func Default() *slog.Logger {
	return gLogger
}

func SetLevel(level slog.Level) {
	logLevel.Set(level)
}

func GetLevel() slog.Level {
	return logLevel.Level()
}

func Level() *slog.LevelVar {
	return &logLevel
}

func SetContext(ctx context.Context) {
	gCtx = ctx
}

func Enabled(lv slog.Level) bool {
	return gLogger.Enabled(gCtx, lv)
}

func Trace(msg string, args ...interface{}) {
	logf(LevelTrace, msg, args...)
}

func Debug(msg string, args ...interface{}) {
	logf(LevelDebug, msg, args...)
}

func Info(msg string, args ...interface{}) {
	logf(LevelInfo, msg, args...)
}

func Notice(msg string, args ...interface{}) {
	logf(LevelNotice, msg, args...)
}

func Warn(msg string, args ...interface{}) {
	logf(LevelWarn, msg, args...)
}

func Error(msg string, args ...interface{}) {
	logf(LevelError, msg, args...)
}

func logf(lv slog.Level, msg string, args ...interface{}) {
	put(gCtx, lv, msg, args...)
}

func put(ctx context.Context, lv slog.Level, format string, args ...interface{}) {
	if gLogger.Enabled(ctx, lv) {
		msg := fmt.Sprintf(format, args...)
		slog.Log(ctx, lv, msg)
	}
}

func TraceContext(ctx context.Context, msg string, args ...interface{}) {
	put(ctx, LevelTrace, msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...interface{}) {
	put(ctx, LevelDebug, msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...interface{}) {
	put(ctx, LevelInfo, msg, args...)
}

func NoticeContext(ctx context.Context, msg string, args ...interface{}) {
	put(ctx, LevelNotice, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...interface{}) {
	put(ctx, LevelWarn, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...interface{}) {
	put(ctx, LevelError, msg, args...)
}

// defer log.PrintTrace(log.Func("FunctionName"))
func PrintTrace(caller string) {
	if gLogger.Enabled(gCtx, LevelTrace) {
		Trace("%s %s", caller, "End")
	}
}

func Func(caller string, args ...interface{}) string {
	if gLogger.Enabled(gCtx, LevelTrace) {
		Trace("%s %s", caller, "Start")
		if len(args) > 0 {
			Trace("Arguments: %s", arguments(args...))
		}
	}
	return caller
}

func arguments(args ...interface{}) string {
	var buf strings.Builder
	for idx, a := range args {
		if idx != 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf("%v", a))
	}
	return buf.String()
}

func PrintStackTrace(err error) {
	slog.Error("Error:\n" + fmt.Sprintf("%+v", err))
}

func NoneStop() {
	if err := recover(); err != nil {
		emergency(err)
	}
}

func emergency(err interface{}) {
	_, file, line, ok := runtime.Caller(3)
	slog.Log(gCtx, LevelEmergency, "Emergency!!\n"+fmt.Sprintf("%+v", err))
	if ok {
		slog.Log(gCtx, LevelEmergency, "File:"+file)
		slog.Log(gCtx, LevelEmergency, fmt.Sprintf("Line:%d", line))
	}
}
