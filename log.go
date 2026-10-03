// Package log は log/slog を土台にしたログの基盤。
//
// slog の代わりになるものではなく、slog に足りないものだけを足す:
//
//   - 段階を 3 つ（LevelTrace / LevelNotice / LevelEmergency）と、その名前の読み書き
//     （ParseLevel / LevelName / ReplaceLevelName）
//   - パッケージ関数（Info / Infof / InfoContext など）。出力先は slog.Default()
//   - ハンドラ: NewSimpleHandler（1 行の素朴な書式）/ NewLevelHandler（レベルで絞る）/
//     NewPackageLevelHandler（呼び出し元のパッケージごとに絞る）/ NewRequestHandler（ctx の属性を足す）
//   - 日付で切り替わるファイル（RollingFileWriter）
//
// # ライブラリから使わないこと
//
// このパッケージを import するのはアプリ（main と、その周りのアプリ専用のパッケージ）だけにする。
// ライブラリは *slog.Logger を受け取るだけにし、レベルの型やログの設定を独自に持たない。
// そうしておけば、使う側は標準の slog だけでライブラリのログを扱える
// （レベルを変えたいなら、NewLevelHandler で絞った Logger を渡せばよい）。
//
// # Info と Infof
//
// Info などは slog と同じ「メッセージ + キーと値」の形で、Infof などが printf の形。
// slog.Info と log.Info を同じ書き方で呼べるようにしてある（Info に書式を渡しても整形されない）。
package log

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// slog の段階（Debug = -4 / Info = 0 / Warn = 4 / Error = 8）の間に 3 つ足す。
const (
	LevelTrace     slog.Level = -8
	LevelDebug                = slog.LevelDebug
	LevelInfo                 = slog.LevelInfo
	LevelNotice    slog.Level = 2
	LevelWarn                 = slog.LevelWarn
	LevelError                = slog.LevelError
	LevelEmergency slog.Level = 32
)

// ParseLevel は "TRACE" / "debug" / "Notice" などの名前をレベルにする（大文字小文字は問わない）。
// slog の書き方（"INFO+2" / "WARN-1" など）も読める。読めない名前はエラーにする
// （黙って Info に倒すと、綴りを間違えたことに気づけない）。
func ParseLevel(s string) (slog.Level, error) {
	t := strings.TrimSpace(s)
	switch strings.ToUpper(t) {
	case "TRACE":
		return LevelTrace, nil
	case "NOTICE":
		return LevelNotice, nil
	case "EMERGENCY", "EMERG":
		return LevelEmergency, nil
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(t)); err != nil {
		return 0, fmt.Errorf("unknown log level %q", s)
	}
	return l, nil
}

// LevelName はレベルの名前を返す。段階の間の値は、それより下で一番近い段階の名前になる
// （slog の "INFO+1" のような書き方はしない）。
func LevelName(l slog.Level) string {
	switch {
	case l < LevelDebug:
		return "TRACE"
	case l < LevelInfo:
		return "DEBUG"
	case l < LevelNotice:
		return "INFO"
	case l < LevelWarn:
		return "NOTICE"
	case l < LevelError:
		return "WARN"
	case l < LevelEmergency:
		return "ERROR"
	default:
		return "EMERGENCY"
	}
}

// ReplaceLevelName は slog.HandlerOptions.ReplaceAttr に渡す関数。
// slog の TextHandler / JSONHandler は足した段階を "DEBUG-4" / "INFO+2" / "ERROR+24" と書くので、
// それを TRACE / NOTICE / EMERGENCY に直す。
//
//	slog.NewTextHandler(w, &slog.HandlerOptions{Level: lv, ReplaceAttr: log.ReplaceLevelName})
func ReplaceLevelName(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.LevelKey {
		if l, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(LevelName(l))
		}
	}
	return a
}

// Enabled は slog.Default() がそのレベルを出すかを返す。
// 引数を組み立てるのが重いときに、先に確かめるためのもの。
func Enabled(l slog.Level) bool {
	return slog.Default().Enabled(context.Background(), l)
}

// ---- パッケージ関数（キーと値の形） -----------------------------------------

func Trace(msg string, args ...any)     { put(nil, LevelTrace, msg, args) }
func Debug(msg string, args ...any)     { put(nil, LevelDebug, msg, args) }
func Info(msg string, args ...any)      { put(nil, LevelInfo, msg, args) }
func Notice(msg string, args ...any)    { put(nil, LevelNotice, msg, args) }
func Warn(msg string, args ...any)      { put(nil, LevelWarn, msg, args) }
func Error(msg string, args ...any)     { put(nil, LevelError, msg, args) }
func Emergency(msg string, args ...any) { put(nil, LevelEmergency, msg, args) }

func TraceContext(ctx context.Context, msg string, args ...any)  { put(ctx, LevelTrace, msg, args) }
func DebugContext(ctx context.Context, msg string, args ...any)  { put(ctx, LevelDebug, msg, args) }
func InfoContext(ctx context.Context, msg string, args ...any)   { put(ctx, LevelInfo, msg, args) }
func NoticeContext(ctx context.Context, msg string, args ...any) { put(ctx, LevelNotice, msg, args) }
func WarnContext(ctx context.Context, msg string, args ...any)   { put(ctx, LevelWarn, msg, args) }
func ErrorContext(ctx context.Context, msg string, args ...any)  { put(ctx, LevelError, msg, args) }
func EmergencyContext(ctx context.Context, msg string, args ...any) {
	put(ctx, LevelEmergency, msg, args)
}

// ---- パッケージ関数（printf の形） -------------------------------------------

func Tracef(format string, args ...any)     { putf(LevelTrace, format, args) }
func Debugf(format string, args ...any)     { putf(LevelDebug, format, args) }
func Infof(format string, args ...any)      { putf(LevelInfo, format, args) }
func Noticef(format string, args ...any)    { putf(LevelNotice, format, args) }
func Warnf(format string, args ...any)      { putf(LevelWarn, format, args) }
func Errorf(format string, args ...any)     { putf(LevelError, format, args) }
func Emergencyf(format string, args ...any) { putf(LevelEmergency, format, args) }

// put と putf は公開関数から**直接**呼ぶこと。呼び出し元の位置を段数で数えている
// （runtime.Callers → output → put → 公開関数 → 呼び出し元）。間に関数を挟むと、
// 記録される位置（と PackageLevelHandler が見るパッケージ）がずれる。
const callerSkip = 4

func put(ctx context.Context, l slog.Level, msg string, args []any) {
	output(ctx, callerSkip, l, msg, "", nil, args)
}

func putf(l slog.Level, format string, args []any) {
	output(nil, callerSkip, l, "", format, args, nil)
}

// output は slog.Default() へ 1 件書く。slog.Log を使わないのは、あちらだと記録される
// 呼び出し元がこのパッケージになるため（PackageLevelHandler が全部このパッケージと見る）。
// format が空でなければ printf の形で、出さないレベルでは fmt.Sprintf を回さない。
func output(ctx context.Context, skip int, l slog.Level, msg, format string, fargs, attrs []any) {
	if ctx == nil {
		ctx = context.Background()
	}
	logger := slog.Default()
	if !logger.Enabled(ctx, l) {
		return
	}
	if format != "" {
		msg = fmt.Sprintf(format, fargs...)
	}
	var pcs [1]uintptr
	runtime.Callers(skip, pcs[:])
	r := slog.NewRecord(time.Now(), l, msg, pcs[0])
	r.Add(attrs...)
	_ = logger.Handler().Handle(ctx, r)
}

// ---- 関数の出入り --------------------------------------------------------------

// Func は関数の入口を Trace で出し、渡した名前をそのまま返す。PrintTrace と組で使う:
//
//	defer log.PrintTrace(log.Func("Name", arg1, arg2))
func Func(name string, args ...any) string {
	if len(args) > 0 && Enabled(LevelTrace) {
		put(nil, LevelTrace, name+" Start", []any{"args", arguments(args)})
	} else {
		put(nil, LevelTrace, name+" Start", nil)
	}
	return name
}

// PrintTrace は関数の出口を Trace で出す。使い方は Func。
func PrintTrace(name string) {
	put(nil, LevelTrace, name+" End", nil)
}

func arguments(args []any) string {
	var b strings.Builder
	for i, a := range args {
		if i != 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "%v", a)
	}
	return b.String()
}

// PrintStackTrace は err を "%+v" で Error に出す（スタックを持つエラーならスタックも出る）。
func PrintStackTrace(err error) {
	if err == nil {
		return
	}
	putf(LevelError, "%+v", []any{err})
}

// ---- panic ---------------------------------------------------------------------

// NoneStop は panic を拾って Emergency で記録し、**そこで止める**（panic を外へ伝えない）。
// goroutine の入口などで、1 つの失敗でプロセスごと落ちないようにするためのもの。
//
//	defer log.NoneStop()
//
// 記録の位置は panic を起こした関数（runtime の中の関数は飛ばす）。スタックも一緒に出す。
func NoneStop() {
	if r := recover(); r != nil {
		emergency(r)
	}
}

func emergency(r any) {
	logger := slog.Default()
	ctx := context.Background()
	if !logger.Enabled(ctx, LevelEmergency) {
		return
	}
	// runtime.Callers → emergency → NoneStop の 3 つを飛ばし、そこから runtime の
	// 関数（gopanic / panicmem / sigpanic など。段数は panic の種類で変わる）を飛ばす。
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	var pc uintptr
	var file string
	var line int
	frames := runtime.CallersFrames(pcs[:n])
	for {
		f, more := frames.Next()
		if !strings.HasPrefix(f.Function, "runtime.") {
			pc, file, line = f.PC, f.File, f.Line
			break
		}
		if !more {
			break
		}
	}
	rec := slog.NewRecord(time.Now(), LevelEmergency, fmt.Sprintf("panic: %+v", r), pc)
	rec.Add("file", file, "line", line, "stack", string(debug.Stack()))
	_ = logger.Handler().Handle(ctx, rec)
}
