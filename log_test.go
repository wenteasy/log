package log_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/wenteasy/log"
)

func TestSetLevel(t *testing.T) {
	log.SetLevel(slog.LevelDebug)
	if log.GetLevel() != slog.LevelDebug {
		t.Errorf("expected Debug, got %v", log.GetLevel())
	}
	log.SetLevel(slog.LevelInfo)
}

func TestPackageFunctions(t *testing.T) {

	var buf strings.Builder
	lv := log.Level()
	lv.Set(log.LevelTrace)

	h := log.NewSimpleHandler(&buf, lv)
	logger := slog.New(h)
	log.SetDefault(logger)
	defer log.SetDefault(slog.Default())

	log.Trace("trace %s", "msg")
	log.Debug("debug %s", "msg")
	log.Info("info %s", "msg")
	log.Notice("notice %s", "msg")
	log.Warn("warn %s", "msg")
	log.Error("error %s", "msg")

	out := buf.String()

	expects := []string{
		"[TRACE ] trace msg",
		"[DEBUG ] debug msg",
		"[INFO  ] info msg",
		"[NOTICE] notice msg",
		"[WARN  ] warn msg",
		"[ERROR ] error msg",
	}

	for _, e := range expects {
		if !strings.Contains(out, e) {
			t.Errorf("missing %q in output:\n%s", e, out)
		}
	}
}

func TestLevelFiltering(t *testing.T) {

	var buf strings.Builder
	lv := log.Level()
	lv.Set(slog.LevelWarn)

	h := log.NewSimpleHandler(&buf, lv)
	logger := slog.New(h)
	log.SetDefault(logger)
	defer func() {
		log.SetDefault(slog.Default())
		lv.Set(slog.LevelInfo)
	}()

	log.Debug("should not appear")
	log.Info("should not appear")
	log.Warn("should appear")
	log.Error("should appear")

	out := buf.String()
	if strings.Contains(out, "should not appear") {
		t.Errorf("low-level messages should be filtered:\n%s", out)
	}
	if !strings.Contains(out, "should appear") {
		t.Errorf("warn/error messages should appear:\n%s", out)
	}
}
