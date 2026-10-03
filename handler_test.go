package log_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/wenteasy/log"
)

func TestSimpleHandlerEmitsAttrs(t *testing.T) {
	var buf strings.Builder
	var lv slog.LevelVar
	lv.Set(log.LevelTrace)

	logger := slog.New(log.NewSimpleHandler(&buf, &lv))
	logger.Info("hello", "key", "val", slog.Int("n", 3))

	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Fatalf("message missing: %q", out)
	}
	if !strings.Contains(out, "key=val") {
		t.Errorf("attr key=val missing: %q", out)
	}
	if !strings.Contains(out, "n=3") {
		t.Errorf("attr n=3 missing: %q", out)
	}
}

func TestSimpleHandlerWithAttrs(t *testing.T) {
	var buf strings.Builder
	var lv slog.LevelVar
	lv.Set(log.LevelTrace)

	h := log.NewSimpleHandler(&buf, &lv)
	logger := slog.New(h.WithAttrs([]slog.Attr{slog.String("service", "test")}))
	logger.Info("msg")

	out := buf.String()
	if !strings.Contains(out, "service=test") {
		t.Errorf("WithAttrs attr missing: %q", out)
	}
}

func TestParseSlogLevel(t *testing.T) {

	vals := []struct {
		value string
		ans   slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"TRACE", log.LevelTrace},
		{"NOTICE", log.LevelNotice},
		{"EMERGENCY", log.LevelEmergency},
		{"info", slog.LevelInfo},
		{"debug", slog.LevelDebug},
	}

	for _, elm := range vals {
		got := log.ParseSlogLevel(elm.value)
		if got != elm.ans {
			t.Errorf("ParseSlogLevel(%s) = %v, want %v", elm.value, got, elm.ans)
		}
	}
}

func TestRequestHandlerInjectsCtxAttrs(t *testing.T) {
	var buf strings.Builder
	var lv slog.LevelVar
	lv.Set(log.LevelTrace)

	h := log.NewRequestHandler(log.NewSimpleHandler(&buf, &lv))
	logger := slog.New(h)

	ctx := log.WithAttrs(context.Background(),
		slog.String("request_id", "req_abc"),
		slog.String("user", "usr_123"),
	)
	logger.InfoContext(ctx, "job started")

	out := buf.String()
	if !strings.Contains(out, "request_id=req_abc") {
		t.Errorf("request_id attr missing: %q", out)
	}
	if !strings.Contains(out, "user=usr_123") {
		t.Errorf("user attr missing: %q", out)
	}
}

func TestRequestHandlerNoCtxAttrs(t *testing.T) {
	var buf strings.Builder
	var lv slog.LevelVar
	lv.Set(log.LevelTrace)

	h := log.NewRequestHandler(log.NewSimpleHandler(&buf, &lv))
	logger := slog.New(h)

	logger.InfoContext(context.Background(), "plain")

	out := buf.String()
	if strings.Contains(out, "request_id=") || strings.Contains(out, "user=") {
		t.Errorf("unexpected attrs in output: %q", out)
	}
}

func TestWithAttrSingle(t *testing.T) {
	var buf strings.Builder
	var lv slog.LevelVar
	lv.Set(log.LevelTrace)

	h := log.NewRequestHandler(log.NewSimpleHandler(&buf, &lv))
	logger := slog.New(h)

	ctx := log.WithAttr(context.Background(), slog.String("trace_id", "t_001"))
	ctx = log.WithAttr(ctx, slog.String("span_id", "s_002"))
	logger.DebugContext(ctx, "span")

	out := buf.String()
	if !strings.Contains(out, "trace_id=t_001") {
		t.Errorf("trace_id missing: %q", out)
	}
	if !strings.Contains(out, "span_id=s_002") {
		t.Errorf("span_id missing: %q", out)
	}
}

func TestPackageTree(t *testing.T) {

	tree := log.NewPackageTree[string]("ROOT")

	tree.Add("a/b", "a/b")
	tree.Add("a/b/c.MethodA", "a/b/c.MethodA")
	tree.Add("a/b/d", "a/b/d")
	tree.Add("a/b/e.MethodA", "a/b/e.MethodA")
	tree.Add("f", "f")
	tree.Add("f/g.MethodB", "f/g.MethodB")
	tree.Add("h/i.MethodA", "h/i.MethodA")

	vals := []struct {
		value string
		ans   string
	}{
		{"a/b/c.MethodA", "a/b/c.MethodA"},
		{"a/b/d", "a/b/d"},
		{"f", "f"},
		{"i", "ROOT"},
		{"a/b/c", "a/b"},
		{"a/b/e", "a/b"},
		{"i", "ROOT"},
	}

	for _, elm := range vals {
		got := tree.Search(elm.value)
		if got != elm.ans {
			t.Errorf("Search(%s) = %v, want %v", elm.value, got, elm.ans)
		}
	}
}
