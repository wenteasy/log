package log

import (
	"context"
	"log/slog"
	"slices"
	"sync/atomic"
)

// LoggerKey は名前付きの Logger の名前を出すときのキー（"logger=ops" のように出る）。
const LoggerKey = "logger"

// loggerName は名前付きの印。ただの文字列と区別するために型を分けている
// （利用側が With("logger", "x") と書いただけでは名前付きにならない）。
type loggerName string

func (n loggerName) LogValue() slog.Value { return slog.StringValue(string(n)) }

// LoggerAttr は名前付きの印になる属性を返す。With で付けると、PackageLevelHandler は
// 呼び出し元のパッケージではなく、この名前でレベルを決める（設定が無ければパッケージで決める）。
//
//	ops := logger.With(log.LoggerAttr("ops"))
//
// slog.Default() を使うなら Named のほうが楽。1 件ごとの属性（logger.Info("m", log.LoggerAttr("ops"))）
// では効かない。With で付けたときだけ。
func LoggerAttr(name string) slog.Attr {
	return slog.Any(LoggerKey, loggerName(name))
}

// loggerNameOf は attrs の中の名前付きの印を探す。複数あれば後ろのものが勝つ。
func loggerNameOf(attrs []slog.Attr) (string, bool) {
	name, ok := "", false
	for _, a := range attrs {
		if n, is := a.Value.Any().(loggerName); is {
			name, ok = string(n), true
		}
	}
	return name, ok
}

// Named は名前付きの *slog.Logger を返す（Java の Logger.getLogger(name) にあたる）。
// 出力先は書くときの slog.Default() で、パッケージの変数に置いてよい
// （slog.SetDefault より先に作っても、あとから設定したハンドラへ書く）:
//
//	var ops = log.Named("ops")
//
//	ops.Info("バッチを始めます", "id", id)
//
// PackageLevelHandler の SetLoggerLevels（JSON なら "loggers"）でこの名前にレベルを決めておけば、
// パッケージの設定とは別に出せる。いつもは Warn で動かしていても、運用で必ず残したいものだけ
// Info で出す、といった使い方をする。名前は "ops.batch" のように "." で区切ると、"ops" の設定が効く。
//
// ⚠️ Named の Logger を slog.SetDefault に渡してはいけない（自分自身へ書くことになる）。
func Named(name string) *slog.Logger {
	attrs := []slog.Attr{LoggerAttr(name)}
	return slog.New(&namedHandler{ops: []func(slog.Handler) slog.Handler{
		func(h slog.Handler) slog.Handler { return h.WithAttrs(attrs) },
	}})
}

// namedHandler は書くたびに slog.Default() のハンドラへ渡す。WithAttrs / WithGroup は覚えておき、
// 渡す先のハンドラに同じ順で掛ける。slog.SetDefault で差し替わったら掛け直す。
type namedHandler struct {
	ops   []func(slog.Handler) slog.Handler
	cache atomic.Pointer[namedCache]
}

type namedCache struct {
	base *slog.Logger // 掛けたときの slog.Default()（ハンドラは == で比べられるとは限らない）
	h    slog.Handler
}

func (n *namedHandler) handler() slog.Handler {
	base := slog.Default()
	if c := n.cache.Load(); c != nil && c.base == base {
		return c.h
	}
	h := base.Handler()
	if _, loop := h.(*namedHandler); loop {
		panic("log: Named の Logger を slog.SetDefault に渡すことはできない")
	}
	for _, op := range n.ops {
		h = op(h)
	}
	n.cache.Store(&namedCache{base: base, h: h})
	return h
}

func (n *namedHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return n.handler().Enabled(ctx, l)
}

func (n *namedHandler) Handle(ctx context.Context, r slog.Record) error {
	return n.handler().Handle(ctx, r)
}

func (n *namedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return n
	}
	return &namedHandler{ops: append(slices.Clip(n.ops), func(h slog.Handler) slog.Handler {
		return h.WithAttrs(attrs)
	})}
}

func (n *namedHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return n
	}
	return &namedHandler{ops: append(slices.Clip(n.ops), func(h slog.Handler) slog.Handler {
		return h.WithGroup(name)
	})}
}
