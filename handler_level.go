package log

import (
	"context"
	"log/slog"
)

// levelHandler は next の手前でレベルを絞る slog.Handler。
type levelHandler struct {
	next  slog.Handler
	level slog.Leveler
}

// NewLevelHandler は next の手前に level の絞り込みを被せたハンドラを返す。
// level に *slog.LevelVar を渡せば、あとから変えられる。
//
// ライブラリごとにレベルを変えたいときに使う。出口（next）は 1 つにまとめたまま、
// ライブラリごとに別の Logger を作って渡す:
//
//	sink := slog.NewMultiHandler(fileHandler, stderrHandler)
//	engineLv := new(slog.LevelVar)
//	engineLv.Set(slog.LevelWarn)
//	engine.SetLogger(slog.New(log.NewLevelHandler(sink, engineLv)))
//
// ⚠️ 下げる方向には効かない。next 自身が出さないレベル（next.Enabled が false）は、
// level をそれより下げても出ない。
func NewLevelHandler(next slog.Handler, level slog.Leveler) slog.Handler {
	if level == nil {
		level = LevelInfo
	}
	return &levelHandler{next: next, level: level}
}

func (h *levelHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return l >= h.level.Level() && h.next.Enabled(ctx, l)
}

func (h *levelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.next.Handle(ctx, r)
}

func (h *levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelHandler{next: h.next.WithAttrs(attrs), level: h.level}
}

func (h *levelHandler) WithGroup(name string) slog.Handler {
	return &levelHandler{next: h.next.WithGroup(name), level: h.level}
}
