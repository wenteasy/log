package log

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"golang.org/x/xerrors"
)

// PackageLevelHandler は PackageTree を使い、呼び出し元パッケージごとに
// slog のログレベルを制御する slog.Handler デコレータ。
// Enabled は常に true を返し、Handle 内で Record.PC からパッケージ名を解決して
// フィルタする。
type PackageLevelHandler struct {
	ParseLevelFunc func(string) slog.Level
	body           slog.Handler
	tree           *PackageTree[slog.Level]
}

func NewPackageLevelHandler(h slog.Handler) *PackageLevelHandler {
	var p PackageLevelHandler
	p.body = h
	p.tree = NewPackageTree[slog.Level](slog.LevelInfo)
	p.ParseLevelFunc = ParseSlogLevel
	return &p
}

func (h *PackageLevelHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *PackageLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	pkg := pcToPackageName(r.PC)
	if pkg == "" {
		if r.Level < slog.LevelInfo {
			return nil
		}
	} else {
		if r.Level < h.tree.Search(pkg) {
			return nil
		}
	}
	return h.body.Handle(ctx, r)
}

func (h *PackageLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PackageLevelHandler{
		ParseLevelFunc: h.ParseLevelFunc,
		body:           h.body.WithAttrs(attrs),
		tree:           h.tree,
	}
}

func (h *PackageLevelHandler) WithGroup(name string) slog.Handler {
	return &PackageLevelHandler{
		ParseLevelFunc: h.ParseLevelFunc,
		body:           h.body.WithGroup(name),
		tree:           h.tree,
	}
}

type settings struct {
	Root     string       `json:"root"`
	Packages []pkgSetting `json:"packages"`
}

type pkgSetting struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

func (h *PackageLevelHandler) LoadJSON(n string) error {

	b, err := os.ReadFile(n)
	if err != nil {
		return xerrors.Errorf("os.ReadFile() error: %w", err)
	}

	var s settings
	err = json.Unmarshal(b, &s)
	if err != nil {
		return xerrors.Errorf("json.Unmarshal() error: %w", err)
	}

	err = h.setPackages(&s)
	if err != nil {
		return xerrors.Errorf("PackageLevelHandler setPackages() error: %w", err)
	}

	return nil
}

func (h *PackageLevelHandler) setPackages(s *settings) error {
	root := h.ParseLevelFunc(s.Root)
	h.tree = NewPackageTree[slog.Level](root)
	for _, elm := range s.Packages {
		h.tree.Add(elm.Name, h.ParseLevelFunc(elm.Level))
	}
	return nil
}

func ParseSlogLevel(v string) slog.Level {
	nv := strings.ToUpper(v)
	switch nv {
	case "TRACE":
		return LevelTrace
	case "DEBUG":
		return slog.LevelDebug
	case "NOTICE":
		return LevelNotice
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	case "EMERGENCY", "EMERG":
		return LevelEmergency
	default:
		return slog.LevelInfo
	}
}

func pcToPackageName(pc uintptr) string {
	if pc == 0 {
		return ""
	}
	return runtime.FuncForPC(pc).Name()
}

// PackageTree はパッケージパスをキーとする汎用 trie。
// パス区切り（"/" と "."）で分割し、最も深くマッチしたノードの値を返す。
//
//	pt := NewPackageTree[string]("ROOT")
//	pt.Add("github.com/wenteasy/log", "Log package")
//	pt.Search("github.com/wenteasy/log.Handler") // -> "Log package"
//	pt.Search("github.com/wenteasy/sync")        // -> "ROOT"
type PackageTree[T any] struct {
	root *tree[T]
}

type tree[T any] struct {
	value    T
	exists   bool
	children map[string]*tree[T]
}

func NewPackageTree[T any](v T) *PackageTree[T] {
	var pt PackageTree[T]
	tree := newTree[T]()
	tree.setValue(v)

	pt.root = tree
	return &pt
}

func (t *PackageTree[T]) GoString() string {
	return t.root.GoString()
}

func newTree[T any]() *tree[T] {
	var t tree[T]
	t.children = make(map[string]*tree[T])
	t.exists = false
	return &t
}

func (t *tree[T]) setValue(v T) {
	t.value = v
	t.exists = true
}

func (t *tree[T]) GoString() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("RootValue:%v\n", t.value))
	t.writeChildren(&builder, 0)
	return builder.String()
}

func (t *tree[T]) writeChildren(w *strings.Builder, d int) {

	space := strings.Repeat(" ", (d+1)*2)

	for key, v := range t.children {
		w.WriteString(fmt.Sprintf("%s%s=%v\n", space, key, v.value))
		v.writeChildren(w, d+1)
	}
}

func (t *tree[T]) add(name string) *tree[T] {
	ct, ok := t.children[name]
	if !ok {
		ct = newTree[T]()
		t.children[name] = ct
	}
	return ct
}

func (t *PackageTree[T]) Add(name string, v T) {
	s := t.parse(name)
	target := t.root
	for idx, child := range s {
		target = target.add(child)
		if idx+1 == len(s) {
			target.setValue(v)
		}
	}
}

func (t *PackageTree[T]) parse(name string) []string {
	str := strings.ReplaceAll(name, "/", ".")
	s := strings.Split(str, ".")
	return s
}

func (t *PackageTree[T]) Search(name string) T {
	names := t.parse(name)
	return t.search(names)
}

func (t *PackageTree[T]) search(names []string) T {

	tree := t.root
	v := tree.value

	ok := false
	for _, name := range names {
		tree, ok = tree.children[name]
		if !ok {
			break
		}

		if tree.exists {
			v = tree.value
		}
	}
	return v
}
