package log_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wenteasy/log"
	"github.com/wenteasy/log/example/a"
)

func TestSimpleHandlerEmitsAttrs(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(log.NewSimpleHandler(&buf, traceLevel()))
	logger.Info("hello", "key", "val", slog.Int("n", 3))

	out := buf.String()
	for _, e := range []string{"hello", "key=val", "n=3"} {
		if !strings.Contains(out, e) {
			t.Errorf("missing %q: %q", e, out)
		}
	}
}

func TestSimpleHandlerWithAttrs(t *testing.T) {
	var buf strings.Builder
	h := log.NewSimpleHandler(&buf, traceLevel())
	logger := slog.New(h.WithAttrs([]slog.Attr{slog.String("service", "test")}))
	logger.Info("msg")

	if out := buf.String(); !strings.Contains(out, "msg service=test") {
		t.Errorf("WithAttrs attr missing: %q", out)
	}
}

func TestSimpleHandlerGroups(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(log.NewSimpleHandler(&buf, nil))
	logger.With("outer", 0).WithGroup("g").Info("m", "k", 1, slog.Group("h", "a", 2), slog.Group("empty"))

	out := buf.String()
	for _, e := range []string{" outer=0", " g.k=1", " g.h.a=2"} {
		if !strings.Contains(out, e) {
			t.Errorf("missing %q: %q", e, out)
		}
	}
	if strings.Contains(out, "empty") {
		t.Errorf("empty group should be dropped: %q", out)
	}
}

func TestLevelHandler(t *testing.T) {
	var buf strings.Builder
	var lv slog.LevelVar
	lv.Set(slog.LevelWarn)
	logger := slog.New(log.NewLevelHandler(log.NewSimpleHandler(&buf, traceLevel()), &lv))

	logger.Info("hidden")
	logger.Warn("shown")
	lv.Set(slog.LevelDebug)
	logger.Debug("after")

	out := buf.String()
	if strings.Contains(out, "hidden") || !strings.Contains(out, "shown") || !strings.Contains(out, "after") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

// 下げる方向には効かない（next が出さないものは出ない）。
func TestLevelHandlerRespectsNext(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(log.NewLevelHandler(log.NewSimpleHandler(&buf, nil), log.LevelTrace))
	logger.Debug("hidden")
	if buf.Len() != 0 {
		t.Errorf("next's level should still apply: %q", buf.String())
	}
}

func TestPackageLevelHandler(t *testing.T) {
	var buf strings.Builder
	h := log.NewPackageLevelHandler(log.NewSimpleHandler(&buf, traceLevel()), slog.LevelWarn)
	h.SetLevels(slog.LevelWarn, map[string]slog.Level{
		"github.com/wenteasy/log_test":      slog.LevelDebug,
		"github.com/wenteasy/log/example/a": slog.LevelError,
	})
	useDefault(t, h)

	log.Debug("test debug") // このパッケージは Debug まで
	log.Trace("test trace")
	slog.Debug("slog debug") // slog を直接呼んでも同じ
	a.Print()                // a は Error まで（Warn も出ない）

	out := buf.String()
	for _, e := range []string{"test debug", "slog debug"} {
		if !strings.Contains(out, e) {
			t.Errorf("missing %q:\n%s", e, out)
		}
	}
	for _, e := range []string{"test trace", "Print a", "print a"} {
		if strings.Contains(out, e) {
			t.Errorf("unexpected %q:\n%s", e, out)
		}
	}
}

// 位置を持たない Record は root のレベルで判定する（Info に決め打ちしない）。
func TestPackageLevelHandlerNoPC(t *testing.T) {
	var buf strings.Builder
	h := log.NewPackageLevelHandler(log.NewSimpleHandler(&buf, traceLevel()), slog.LevelError)
	ctx := context.Background()
	h.Handle(ctx, slog.NewRecord(time.Now(), slog.LevelWarn, "warn", 0))
	h.Handle(ctx, slog.NewRecord(time.Now(), slog.LevelError, "error", 0))

	out := buf.String()
	if strings.Contains(out, "warn") || !strings.Contains(out, "error") {
		t.Errorf("records without PC should use root:\n%s", out)
	}
}

func TestPackageLevelHandlerRespectsBody(t *testing.T) {
	var buf strings.Builder
	h := log.NewPackageLevelHandler(log.NewSimpleHandler(&buf, nil), log.LevelTrace)
	slog.New(h).Debug("hidden")
	if buf.Len() != 0 {
		t.Errorf("body's level should still apply: %q", buf.String())
	}
}

// 派生したハンドラにも、あとから変えた設定が効く。
func TestPackageLevelHandlerDerivedSharesLevels(t *testing.T) {
	var buf strings.Builder
	h := log.NewPackageLevelHandler(log.NewSimpleHandler(&buf, traceLevel()), slog.LevelWarn)
	logger := slog.New(h).With("k", 1)
	h.SetLevels(slog.LevelDebug, nil)
	logger.Debug("shown")
	if !strings.Contains(buf.String(), "shown") {
		t.Errorf("derived handler should follow SetLevels: %q", buf.String())
	}
}

func TestPackageLevelHandlerLoadJSON(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	os.WriteFile(good, []byte(`{"root":"warn","packages":[{"name":"github.com/x/y","level":"TRACE"}]}`), 0o644)
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte(`{"root":"INFO","packages":[{"name":"github.com/x/y","level":"VERBOSE"}]}`), 0o644)

	h := log.NewPackageLevelHandler(slog.DiscardHandler, slog.LevelInfo)
	if err := h.LoadJSON(good); err != nil {
		t.Fatal(err)
	}
	if got := h.Level("github.com/x/y.F"); got != log.LevelTrace {
		t.Errorf("Level(x/y) = %v", got)
	}
	if got := h.Level("github.com/x/z"); got != slog.LevelWarn {
		t.Errorf("Level(x/z) = %v", got)
	}
	if err := h.LoadJSON(bad); err == nil {
		t.Error("unknown level should be an error")
	}
	if got := h.Level("github.com/x/y"); got != log.LevelTrace {
		t.Errorf("failed LoadJSON should keep the current levels: %v", got)
	}
}

func TestRequestHandlerInjectsCtxAttrs(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(log.NewRequestHandler(log.NewSimpleHandler(&buf, traceLevel())))

	ctx := log.WithAttrs(context.Background(),
		slog.String("request_id", "req_abc"),
		slog.String("user", "usr_123"),
	)
	ctx = log.WithAttr(ctx, slog.String("span_id", "s_002"))
	logger.InfoContext(ctx, "job started")
	logger.InfoContext(context.Background(), "plain")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %q", lines)
	}
	for _, e := range []string{"request_id=req_abc", "user=usr_123", "span_id=s_002"} {
		if !strings.Contains(lines[0], e) {
			t.Errorf("missing %q: %q", e, lines[0])
		}
	}
	if strings.Contains(lines[1], "=") {
		t.Errorf("unexpected attrs: %q", lines[1])
	}
}

func TestPackageTree(t *testing.T) {
	tree := log.NewPackageTree[string]("ROOT")
	tree.Add("a/b", "a/b")
	tree.Add("a/b/c.MethodA", "a/b/c.MethodA")
	tree.Add("a/b/d", "a/b/d")
	tree.Add("f", "f")
	tree.Add("github.com/x/y", "x/y")

	for _, c := range []struct{ in, want string }{
		{"a/b/c.MethodA", "a/b/c.MethodA"},
		{"a/b/d", "a/b/d"},
		{"f", "f"},
		{"i", "ROOT"},
		{"a/b/c", "a/b"},
		{"a/b/e", "a/b"},
		{"github.com/x/y.(*T).M", "x/y"},
		{"github.com/x/yy", "ROOT"},
	} {
		if got := tree.Search(c.in); got != c.want {
			t.Errorf("Search(%s) = %v, want %v", c.in, got, c.want)
		}
	}
}
