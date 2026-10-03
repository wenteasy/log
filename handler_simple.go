package log

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// simpleHandler は標準の log パッケージに近い、1 件 1 行の素朴な書式で書く slog.Handler。
//
//	2006-01-02T15:04:05+09:00 [INFO  ] メッセージ key=value group.key=value
type simpleHandler struct {
	mu     *sync.Mutex // 派生したハンドラと共有する（同じ w へ行が混ざらないように）
	w      io.Writer
	level  slog.Leveler
	prefix string // WithGroup で付いたグループ（"a.b."）
	pre    string // WithAttrs で付いた属性を書式にしたもの
}

// NewSimpleHandler は w へ 1 件 1 行で書くハンドラを返す。
// level を下回るものは出さない。level に *slog.LevelVar を渡せば、あとから変えられる。
// nil なら Info。
func NewSimpleHandler(w io.Writer, level slog.Leveler) slog.Handler {
	if level == nil {
		level = LevelInfo
	}
	return &simpleHandler{mu: &sync.Mutex{}, w: w, level: level}
}

func (h *simpleHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *simpleHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	if !r.Time.IsZero() {
		b.WriteString(r.Time.Format(time.RFC3339))
		b.WriteByte(' ')
	}
	b.WriteByte('[')
	name := LevelName(r.Level)
	if r.Level >= LevelEmergency {
		name = "EMERG" // 幅を 6 文字に揃えるため
	}
	b.WriteString(name)
	b.WriteString(strings.Repeat(" ", max(0, len("NOTICE")-len(name))))
	b.WriteString("] ")
	b.WriteString(r.Message)
	b.WriteString(h.pre)
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&b, h.prefix, a)
		return true
	})
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, b.String())
	return err
}

// writeAttr は " key=value" を書く。グループは "group.key=value" に開く。
func writeAttr(b *strings.Builder, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return
		}
		p := prefix
		if a.Key != "" {
			p += a.Key + "."
		}
		for _, ga := range attrs {
			writeAttr(b, p, ga)
		}
		return
	}
	b.WriteByte(' ')
	b.WriteString(prefix)
	b.WriteString(a.Key)
	b.WriteByte('=')
	b.WriteString(a.Value.String())
}

func (h *simpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	var b strings.Builder
	b.WriteString(h.pre)
	for _, a := range attrs {
		writeAttr(&b, h.prefix, a)
	}
	nh := *h
	nh.pre = b.String()
	return &nh
}

func (h *simpleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	nh := *h
	nh.prefix = h.prefix + name + "."
	return &nh
}
