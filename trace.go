package log

import (
	"fmt"
	"strings"
)

// Trace() Implementation is done with Logger struct

func (l *Logger) TS(msg string) string {
	l.write(gTraceConfig.lv, "%s %s %s", gTraceConfig.prefix, msg, gTraceConfig.tsSuffix)
	return fmt.Sprintf("%s %s %s", gTraceConfig.prefix, msg, gTraceConfig.traceSuffix)
}

func (l *Logger) Trace(msg string) {
	l.write(gTraceConfig.lv, msg)
}

var gTraceConfig *traceConfig

func init() {
	gTraceConfig = defaultTrace()
}

func defaultTrace() *traceConfig {
	var tc traceConfig
	tc.set(DEBUG, strings.Repeat("=", 10), "Start", "End")
	return &tc
}

type traceConfig struct {
	prefix      string
	tsSuffix    string
	traceSuffix string
	lv          Priority
}

func (tc *traceConfig) set(lv Priority, p, s, e string) {
	tc.lv = lv
	tc.prefix = p
	tc.tsSuffix = s
	tc.traceSuffix = e
}

func SetTrace(lv Priority, p, s, e string) {
	gTraceConfig.set(lv, p, s, e)
}
