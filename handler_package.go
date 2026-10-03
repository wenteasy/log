package log

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

// PackageLevelHandler は、ログを出した関数のパッケージごとにレベルを変える slog.Handler。
// Record.PC から関数名（"github.com/x/y.(*T).Method" など）を引き、PackageTree で
// 一番深く当たった設定のレベルを使う。どれにも当たらなければ root のレベル。
//
// 呼び出し元が分かるのは、slog を直接呼んだときと、このパッケージの Info などを通したとき。
// 位置を持たない Record（PC が 0）は root のレベルで判定する。
//
// ⚠️ 下げる方向には効かない。body 自身が出さないレベル（body.Enabled が false）は、
// パッケージのレベルをそれより下げても出ない。細かく出したいなら body は低いレベルで作る。
//
// 設定（SetLevels / LoadJSON）は WithAttrs / WithGroup で派生したハンドラにも効く。
// 実行中に変えてよい。
type PackageLevelHandler struct {
	body  slog.Handler
	state *packageLevelState
}

type packageLevelState struct {
	levels atomic.Pointer[packageLevels]
	names  sync.Map // uintptr（PC）→ string（関数名）
}

// packageLevels は 1 回ぶんの設定。作ったあとは書き換えない（差し替えるだけ）。
type packageLevels struct {
	root slog.Level
	tree *PackageTree[slog.Level]
	min  slog.Level // root と全パッケージの中で一番低いレベル（Enabled の足切り）
}

// NewPackageLevelHandler は body の手前にパッケージごとの絞り込みを被せる。
// 設定するまでは、どのパッケージも root のレベル。
func NewPackageLevelHandler(body slog.Handler, root slog.Level) *PackageLevelHandler {
	h := &PackageLevelHandler{body: body, state: &packageLevelState{}}
	h.SetLevels(root, nil)
	return h
}

// SetLevels は設定を丸ごと差し替える。pkgs のキーはパッケージのパス
// （"github.com/x/y"）か、関数まで含めた名前（"github.com/x/y.Func" / "main.main"）。
func (h *PackageLevelHandler) SetLevels(root slog.Level, pkgs map[string]slog.Level) {
	lv := &packageLevels{root: root, tree: NewPackageTree(root), min: root}
	for name, l := range pkgs {
		lv.tree.Add(name, l)
		lv.min = min(lv.min, l)
	}
	h.state.levels.Store(lv)
}

// LevelsFile は LoadJSON が読む JSON の形。
//
//	{
//	  "root": "WARN",
//	  "packages": [
//	    {"name": "github.com/x/y", "level": "DEBUG"},
//	    {"name": "main.main", "level": "TRACE"}
//	  ]
//	}
//
// レベルの書き方は ParseLevel。
type LevelsFile struct {
	Root     string `json:"root"`
	Packages []struct {
		Name  string `json:"name"`
		Level string `json:"level"`
	} `json:"packages"`
}

// LoadJSON はファイルから設定を読み、丸ごと差し替える。
// 読めないレベルが 1 つでもあればエラーにして、今の設定は変えない。
func (h *PackageLevelHandler) LoadJSON(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("os.ReadFile() error: %w", err)
	}
	var f LevelsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("json.Unmarshal() error: %w", err)
	}
	root := LevelInfo
	if f.Root != "" {
		if root, err = ParseLevel(f.Root); err != nil {
			return fmt.Errorf("root: %w", err)
		}
	}
	pkgs := make(map[string]slog.Level, len(f.Packages))
	for _, p := range f.Packages {
		l, err := ParseLevel(p.Level)
		if err != nil {
			return fmt.Errorf("packages %q: %w", p.Name, err)
		}
		pkgs[p.Name] = l
	}
	h.SetLevels(root, pkgs)
	return nil
}

// Level は関数名（またはパッケージのパス）に効くレベルを返す。設定の確かめ用。
func (h *PackageLevelHandler) Level(name string) slog.Level {
	return h.state.levels.Load().tree.Search(name)
}

func (h *PackageLevelHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return l >= h.state.levels.Load().min && h.body.Enabled(ctx, l)
}

func (h *PackageLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	lv := h.state.levels.Load()
	need := lv.root
	if name := h.state.funcName(r.PC); name != "" {
		need = lv.tree.Search(name)
	}
	if r.Level < need {
		return nil
	}
	return h.body.Handle(ctx, r)
}

func (h *PackageLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PackageLevelHandler{body: h.body.WithAttrs(attrs), state: h.state}
}

func (h *PackageLevelHandler) WithGroup(name string) slog.Handler {
	return &PackageLevelHandler{body: h.body.WithGroup(name), state: h.state}
}

// funcName は PC の関数名を返す（インライン展開された関数も、元の関数の名前で返す）。
// 同じ PC は何度も来るので覚えておく。
func (s *packageLevelState) funcName(pc uintptr) string {
	if pc == 0 {
		return ""
	}
	if v, ok := s.names.Load(pc); ok {
		return v.(string)
	}
	f, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	s.names.Store(pc, f.Function)
	return f.Function
}

// PackageTree はパッケージのパスをキーにした trie。
// "/" と "." で区切り、一番深く当たったノードの値を返す。
//
//	pt := NewPackageTree[string]("ROOT")
//	pt.Add("github.com/wenteasy/log", "Log package")
//	pt.Search("github.com/wenteasy/log.Handler") // -> "Log package"
//	pt.Search("github.com/wenteasy/sync")        // -> "ROOT"
//
// 並行に Add と Search を呼んではいけない（PackageLevelHandler は作り直して差し替えている）。
type PackageTree[T any] struct {
	root *tree[T]
}

type tree[T any] struct {
	value    T
	exists   bool
	children map[string]*tree[T]
}

func NewPackageTree[T any](v T) *PackageTree[T] {
	t := newTree[T]()
	t.setValue(v)
	return &PackageTree[T]{root: t}
}

func (t *PackageTree[T]) GoString() string {
	return t.root.GoString()
}

func newTree[T any]() *tree[T] {
	return &tree[T]{children: make(map[string]*tree[T])}
}

func (t *tree[T]) setValue(v T) {
	t.value = v
	t.exists = true
}

func (t *tree[T]) GoString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "RootValue:%v\n", t.value)
	t.writeChildren(&b, 0)
	return b.String()
}

func (t *tree[T]) writeChildren(w *strings.Builder, d int) {
	space := strings.Repeat(" ", (d+1)*2)
	for key, v := range t.children {
		fmt.Fprintf(w, "%s%s=%v\n", space, key, v.value)
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

// Add は name に値を置く。空の name は root の値を置き換える。
func (t *PackageTree[T]) Add(name string, v T) {
	target := t.root
	for _, seg := range splitPackage(name) {
		target = target.add(seg)
	}
	target.setValue(v)
}

// Search は name に一番深く当たった値を返す。
func (t *PackageTree[T]) Search(name string) T {
	node := t.root
	v := node.value
	for _, seg := range splitPackage(name) {
		next, ok := node.children[seg]
		if !ok {
			break
		}
		node = next
		if node.exists {
			v = node.value
		}
	}
	return v
}

func splitPackage(name string) []string {
	return strings.FieldsFunc(name, func(r rune) bool { return r == '/' || r == '.' })
}
