package log

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/exp/slog"
	"golang.org/x/xerrors"
)

func ParseSlogLevel(v string) slog.Level {
	nv := strings.ToUpper(v)
	l := slog.InfoLevel
	switch nv {
	case "WARN":
		l = slog.WarnLevel
	case "INFO":
		l = slog.InfoLevel
	case "DEBUG":
		l = slog.DebugLevel
	case "ERROR":
		l = slog.ErrorLevel
	}
	return l
}

func getCallerPackageName(depth int) string {
	pc, _, _, _ := runtime.Caller(depth)

	funcName := runtime.FuncForPC(pc).Name()
	lastDot := strings.LastIndexByte(funcName, '.')

	return funcName[:lastDot]
}

//
// golang.org/x/exp/slog Handler
//
type PackageLevelHandler struct {
	ParseLevelFunc func(string) slog.Level
	body           slog.Handler
	tree           *PackageTree[slog.Level]
}

func NewPackageLevelHandler(h slog.Handler) *PackageLevelHandler {
	var p PackageLevelHandler
	p.body = h
	p.tree = NewPackageTree[slog.Level](slog.InfoLevel)
	p.ParseLevelFunc = ParseSlogLevel
	return &p
}

type settings struct {
	Root     string       `"json":"root"`
	Packages []pkgSetting `"json":"packages"`
}

type pkgSetting struct {
	Name  string `"json":"name"`
	Level string `"json":"level"`
}

func (h *PackageLevelHandler) LoadJSON(n string) error {

	b, err := os.ReadFile(n)
	if err != nil {
		return xerrors.Errorf("os.ReadFile() error: %w", err)
	}

	var s settings
	err = json.Unmarshal(b, &s)
	if err != nil {
		return xerrors.Errorf("json.Marshal() error: %w", err)
	}

	err = h.setPackages(&s)
	if err != nil {
		return xerrors.Errorf("PackageLevelHandler setPackages() error: %w", err)
	}

	return nil
}

func (h *PackageLevelHandler) LoadYAML(n string) error {
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

func (h *PackageLevelHandler) Enabled(l slog.Level) bool {

	pkg := getCallerPackageName(2)
	if pkg == "" {
		return l >= slog.InfoLevel
	}

	v := h.tree.Search(pkg)
	return v >= l
}

func (h *PackageLevelHandler) Handle(r slog.Record) error {
	return h.body.Handle(r)
}

func (h *PackageLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.body.WithAttrs(attrs)
}

func (h *PackageLevelHandler) WithGroup(name string) slog.Handler {
	return h.body.WithGroup(name)
}

//
// (e.g.
//    pt := NewPackageTree[string]("ROOT")
//    pt.Add("github.com","GitHub")
//    pt.Add("github.com/westeasy","WestEasy")
//    pt.Add("github.com/westeasy/log","WestEasy Log package")
//    pt.Add("github.com/westeasy/strings","WestEasy Strings package")
//
//    v := pt.Search("github.com/westeasy/log.PackageLevelHandler")
//    ret -> "WestEasy Log package"
//    v := pt.Search("github.com/westeasy/sync")
//    ret -> "WestEasy"
//    v := pt.Search("github.com/shizuokago/blog")
//    ret -> "ROOT"
//
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
