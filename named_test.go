package log_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wenteasy/log"
)

// slog.SetDefault より先に作った（パッケージの変数に置いた）Logger も、あとの設定へ書く。
var ops = log.Named("ops")

func newNamedHandler(buf *strings.Builder) *log.PackageLevelHandler {
	h := log.NewPackageLevelHandler(log.NewSimpleHandler(buf, traceLevel()), slog.LevelWarn)
	h.SetLoggerLevels(map[string]slog.Level{"ops": slog.LevelInfo})
	return h
}

func TestNamedUsesLoggerLevel(t *testing.T) {
	var buf strings.Builder
	useDefault(t, newNamedHandler(&buf))

	slog.Info("plain info") // root は Warn
	ops.Info("ops info")    // ops は Info
	ops.Debug("ops debug")

	out := buf.String()
	if strings.Contains(out, "plain info") || strings.Contains(out, "ops debug") {
		t.Errorf("unexpected output:\n%s", out)
	}
	if !strings.Contains(out, "ops info logger=ops") {
		t.Errorf("named logger should use its own level and show its name:\n%s", out)
	}
}

// 名前の設定はパッケージの設定より強い（上げる方向にも効く）。
func TestNamedOverridesPackage(t *testing.T) {
	var buf strings.Builder
	h := newNamedHandler(&buf)
	h.SetLevels(slog.LevelWarn, map[string]slog.Level{"github.com/wenteasy/log_test": log.LevelTrace})
	h.SetLoggerLevels(map[string]slog.Level{"ops": slog.LevelError})
	useDefault(t, h)

	slog.Debug("plain debug")
	ops.Warn("ops warn")

	out := buf.String()
	if !strings.Contains(out, "plain debug") || strings.Contains(out, "ops warn") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

// 設定の無い名前は、名前が無いときと同じくパッケージで決める。
func TestNamedFallsBackToPackage(t *testing.T) {
	var buf strings.Builder
	h := newNamedHandler(&buf)
	h.SetLevels(slog.LevelWarn, map[string]slog.Level{"github.com/wenteasy/log_test": slog.LevelDebug})
	useDefault(t, h)

	log.Named("other").Debug("other debug")
	log.Named("other").Log(t.Context(), log.LevelTrace, "other trace")

	out := buf.String()
	if !strings.Contains(out, "other debug") || strings.Contains(out, "other trace") {
		t.Errorf("unconfigured name should use the package level:\n%s", out)
	}
}

func TestNamedHierarchy(t *testing.T) {
	var buf strings.Builder
	h := newNamedHandler(&buf)
	h.SetLoggerLevels(map[string]slog.Level{"ops": slog.LevelInfo, "ops.noisy": slog.LevelError})
	useDefault(t, h)

	log.Named("ops.batch").Info("batch info")
	log.Named("ops.noisy").Warn("noisy warn")

	out := buf.String()
	if !strings.Contains(out, "batch info") || strings.Contains(out, "noisy warn") {
		t.Errorf("deepest match should win:\n%s", out)
	}
	if l, ok := h.LoggerLevel("ops.batch.x"); !ok || l != slog.LevelInfo {
		t.Errorf("LoggerLevel(ops.batch.x) = %v, %v", l, ok)
	}
	if _, ok := h.LoggerLevel("opsx"); ok {
		t.Error("LoggerLevel(opsx) should not match")
	}
}

// ただの文字列の "logger" では名前付きにならない。
func TestNamedNeedsLoggerAttr(t *testing.T) {
	var buf strings.Builder
	h := newNamedHandler(&buf)
	logger := slog.New(h)

	logger.With("logger", "ops").Info("string key")
	logger.With(log.LoggerAttr("ops")).Info("attr")

	out := buf.String()
	if strings.Contains(out, "string key") || !strings.Contains(out, "attr logger=ops") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

// 派生（With / WithGroup）しても名前は残り、slog.SetDefault の差し替えにも付いていく。
func TestNamedDerivedFollowsDefault(t *testing.T) {
	var first, second strings.Builder
	useDefault(t, newNamedHandler(&first))

	job := ops.With("job", "j1").WithGroup("g")
	job.Info("one", "k", 1)
	useDefault(t, log.NewRequestHandler(newNamedHandler(&second))) // 間に別のハンドラを挟んでも効く
	job.Info("two", "k", 2)

	if !strings.Contains(first.String(), "one logger=ops job=j1 g.k=1") {
		t.Errorf("first: %q", first.String())
	}
	if !strings.Contains(second.String(), "two logger=ops job=j1 g.k=2") {
		t.Errorf("second: %q", second.String())
	}
}

// 派生したハンドラにも、あとから変えた名前の設定が効く。SetLevels は名前の設定を消さない。
func TestNamedFollowsSetLoggerLevels(t *testing.T) {
	var buf strings.Builder
	h := newNamedHandler(&buf)
	useDefault(t, h)

	ops.Info("before")
	h.SetLoggerLevels(nil)
	ops.Info("cleared")
	h.SetLoggerLevels(map[string]slog.Level{"ops": slog.LevelInfo})
	h.SetLevels(slog.LevelError, nil)
	ops.Info("after SetLevels")

	out := buf.String()
	if !strings.Contains(out, "before") || strings.Contains(out, "cleared") || !strings.Contains(out, "after SetLevels") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestNamedLoadJSON(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	os.WriteFile(good, []byte(`{"root":"warn","loggers":[{"name":"ops","level":"INFO"}]}`), 0o644)
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte(`{"root":"warn","loggers":[{"name":"ops","level":"VERBOSE"}]}`), 0o644)

	h := log.NewPackageLevelHandler(slog.DiscardHandler, slog.LevelInfo)
	if err := h.LoadJSON(good); err != nil {
		t.Fatal(err)
	}
	if l, ok := h.LoggerLevel("ops"); !ok || l != slog.LevelInfo {
		t.Errorf("LoggerLevel(ops) = %v, %v", l, ok)
	}
	if err := h.LoadJSON(bad); err == nil {
		t.Error("unknown level should be an error")
	}
	if l, ok := h.LoggerLevel("ops"); !ok || l != slog.LevelInfo {
		t.Errorf("failed LoadJSON should keep the current levels: %v, %v", l, ok)
	}
}

func TestNamedAsDefaultPanics(t *testing.T) {
	useDefault(t, log.Named("loop").Handler())
	defer func() {
		if recover() == nil {
			t.Error("should panic")
		}
	}()
	slog.Info("x")
}
