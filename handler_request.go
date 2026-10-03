package log

import (
	"context"
	"log/slog"
)

type ctxAttrsKey struct{}

// WithAttr は ctx に slog.Attr を追加する。
// requestHandler が Handle 時に ctx から全属性を取り出して Record に付与する。
func WithAttr(ctx context.Context, attr slog.Attr) context.Context {
	attrs := ctxAttrs(ctx)
	attrs = append(attrs, attr)
	return context.WithValue(ctx, ctxAttrsKey{}, attrs)
}

// WithAttrs は複数の slog.Attr をまとめて ctx に追加する。
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing := ctxAttrs(ctx)
	existing = append(existing, attrs...)
	return context.WithValue(ctx, ctxAttrsKey{}, existing)
}

func ctxAttrs(ctx context.Context) []slog.Attr {
	if v, ok := ctx.Value(ctxAttrsKey{}).([]slog.Attr); ok {
		copied := make([]slog.Attr, len(v), len(v)+4)
		copy(copied, v)
		return copied
	}
	return make([]slog.Attr, 0, 4)
}

// requestHandler は ctx に載せた属性を Record に付与して次の Handler に委譲するデコレータ。
type requestHandler struct {
	next slog.Handler
}

func NewRequestHandler(next slog.Handler) slog.Handler {
	return &requestHandler{next: next}
}

func (h *requestHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.next.Enabled(ctx, l)
}

func (h *requestHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs := ctxAttrs(ctx); len(attrs) > 0 {
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
