package log

import (
	"context"
	"log/slog"
)

type ctxAttrsKey struct{}

// WithAttr は ctx に slog.Attr を 1 つ足す。
// NewRequestHandler で包んだハンドラが、書くときに ctx から取り出して Record に足す。
func WithAttr(ctx context.Context, attr slog.Attr) context.Context {
	return WithAttrs(ctx, attr)
}

// WithAttrs は ctx に slog.Attr をまとめて足す。親の ctx の属性は書き換えない。
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing, _ := ctx.Value(ctxAttrsKey{}).([]slog.Attr)
	merged := make([]slog.Attr, 0, len(existing)+len(attrs))
	merged = append(append(merged, existing...), attrs...)
	return context.WithValue(ctx, ctxAttrsKey{}, merged)
}

// requestHandler は ctx に載せた属性を Record に足して次へ渡す。
type requestHandler struct {
	next slog.Handler
}

// NewRequestHandler は、WithAttr / WithAttrs で ctx に載せた属性を Record に足すハンドラを返す。
// リクエスト ID のように、呼び出しの流れに付いて回る値を出すためのもの。
// ctx を受け取る書き方（InfoContext など）で書いたときだけ効く。
func NewRequestHandler(next slog.Handler) slog.Handler {
	return &requestHandler{next: next}
}

func (h *requestHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.next.Enabled(ctx, l)
}

func (h *requestHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, _ := ctx.Value(ctxAttrsKey{}).([]slog.Attr); len(attrs) > 0 {
		r.AddAttrs(attrs...)
	}
	return h.next.Handle(ctx, r)
}

func (h *requestHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &requestHandler{next: h.next.WithAttrs(attrs)}
}

func (h *requestHandler) WithGroup(name string) slog.Handler {
	return &requestHandler{next: h.next.WithGroup(name)}
}
