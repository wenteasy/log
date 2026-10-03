package log_test

import (
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/wenteasy/log"
)

// useDefault は slog.Default() を h に差し替え、テストの終わりに戻す。
func useDefault(t *testing.T, h slog.Handler) {
	t.Helper()
	old := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(old) })
}

func traceLevel() *slog.LevelVar {
	var lv slog.LevelVar
	lv.Set(log.LevelTrace)
	return &lv
}

func TestPackageFunctions(t *testing.T) {
	var buf strings.Builder
	useDefault(t, log.NewSimpleHandler(&buf, traceLevel()))

	log.Trace("trace", "k", 1)
	log.Debug("debug", "k", 2)
	log.Info("info", "k", 3)
	log.Notice("notice")
	log.Warn("warn")
	log.Error("error")
	log.Emergency("emergency")
	log.Infof("infof %s %d", "msg", 4)

	out := buf.String()
	for _, e := range []string{
		"[TRACE ] trace k=1",
		"[DEBUG ] debug k=2",
		"[INFO  ] info k=3",
		"[NOTICE] notice",
		"[WARN  ] warn",
		"[ERROR ] error",
		"[EMERG ] emergency",
		"[INFO  ] infof msg 4",
	} {
		if !strings.Contains(out, e) {
			t.Errorf("missing %q in output:\n%s", e, out)
		}
	}
}

// Info は slog と同じくキーと値の形で、書式は解釈しない（printf は Infof）。
func TestInfoIsNotPrintf(t *testing.T) {
	var buf strings.Builder
	useDefault(t, log.NewSimpleHandler(&buf, nil))

	log.Info("a %s", "b")
	if out := buf.String(); !strings.Contains(out, "a %s !BADKEY=b") {
		t.Errorf("Info should behave like slog.Info: %q", out)
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf strings.Builder
	var lv slog.LevelVar
	lv.Set(log.LevelWarn)
	useDefault(t, log.NewSimpleHandler(&buf, &lv))

	log.Debug("should not appear")
	log.Infof("should not appear %d", 1)
	log.Warn("should appear")

	out := buf.String()
	if strings.Contains(out, "should not appear") {
		t.Errorf("low-level messages should be filtered:\n%s", out)
	}
	if !strings.Contains(out, "should appear") {
		t.Errorf("warn should appear:\n%s", out)
	}
}

// 記録される位置が、このパッケージの中ではなく呼んだ側になること。
// ずれると AddSource の位置も PackageLevelHandler の判定も全部このパッケージになる。
func TestCallerIsTheCaller(t *testing.T) {
	var buf strings.Builder
	useDefault(t, slog.NewTextHandler(&buf, &slog.HandlerOptions{AddSource: true, Level: log.LevelTrace}))

	calls := map[string]func(){
		"Info":            func() { log.Info("x") },
		"Infof":           func() { log.Infof("x %d", 1) },
		"Func":            func() { log.Func("F", 1) },
		"PrintTrace":      func() { log.PrintTrace("F") },
		"PrintStackTrace": func() { log.PrintStackTrace(errors.New("e")) },
	}
	for name, call := range calls {
		buf.Reset()
		call()
		if out := buf.String(); !strings.Contains(out, "log_test.go") {
			t.Errorf("%s: source should be the caller: %q", name, out)
		}
	}
}

func TestParseLevel(t *testing.T) {
	for _, c := range []struct {
		in   string
		want slog.Level
	}{
		{"TRACE", log.LevelTrace},
		{"debug", slog.LevelDebug},
		{"Info", slog.LevelInfo},
		{"NOTICE", log.LevelNotice},
		{"warn", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"EMERGENCY", log.LevelEmergency},
		{"emerg", log.LevelEmergency},
		{" INFO+1 ", slog.LevelInfo + 1},
	} {
		got, err := log.ParseLevel(c.in)
		if err != nil || got != c.want {
			t.Errorf("ParseLevel(%q) = %v, %v; want %v", c.in, got, err, c.want)
		}
	}
	if _, err := log.ParseLevel("WARNING"); err == nil {
		t.Error("unknown level should be an error")
	}
}

func TestReplaceLevelName(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:       log.LevelTrace,
		ReplaceAttr: log.ReplaceLevelName,
	}))
	logger.Log(t.Context(), log.LevelTrace, "t")
	logger.Log(t.Context(), log.LevelNotice, "n")
	logger.Log(t.Context(), log.LevelEmergency, "e")

	out := buf.String()
	for _, e := range []string{"level=TRACE", "level=NOTICE", "level=EMERGENCY"} {
		if !strings.Contains(out, e) {
			t.Errorf("missing %q in output:\n%s", e, out)
		}
	}
}

func panicky() {
	defer log.NoneStop()
	panic("boom")
}

func nilDeref() {
	defer log.NoneStop()
	var p *struct{ v int }
	_ = p.v
}

// NoneStop は panic を止め、起こした関数の位置で記録する（runtime の中ではなく）。
func TestNoneStop(t *testing.T) {
	var buf strings.Builder
	useDefault(t, slog.NewTextHandler(&buf, &slog.HandlerOptions{AddSource: true}))

	for name, f := range map[string]func(){"panic": panicky, "nil": nilDeref} {
		buf.Reset()
		f()
		out := buf.String()
		if !strings.Contains(out, "panic: ") {
			t.Errorf("%s: panic should be logged: %q", name, out)
		}
		if !strings.Contains(out, "log_test.go") {
			t.Errorf("%s: source should be the panicking function: %q", name, out)
		}
	}
}
