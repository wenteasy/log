package log_test

import (
	"testing"

	"github.com/wenteasy/log"
	"golang.org/x/exp/slog"
)

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
			t.Errorf("ParseSlog(%s) got %v", elm.value, elm.ans)
		}
	}

}

func TestParseSlogLevel(t *testing.T) {

	vals := []struct {
		value string
		ans   slog.Level
	}{
		{"DEBUG", slog.DebugLevel},
		{"INFO", slog.InfoLevel},
		{"WARN", slog.WarnLevel},
		{"ERROR", slog.ErrorLevel},
		{"info", slog.InfoLevel},
		{"debug", slog.DebugLevel},
		{"warn", slog.WarnLevel},
		{"error", slog.ErrorLevel},
	}

	for _, elm := range vals {
		got := log.ParseSlogLevel(elm.value)
		if got != elm.ans {
			t.Errorf("ParseSlog(%s) got %v", elm.value, elm.ans)
		}
	}
}
