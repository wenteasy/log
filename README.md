# log

`log/slog` を土台にしたログの基盤。**slog の代わりではなく、slog に足りないものだけを足す。**
出口はいつも slog（`slog.Default()` や `*slog.Logger`）なので、このパッケージを使っていない
コードやライブラリのログも同じ出口に集まる。

```
go get github.com/wenteasy/log@latest
```

Go 1.25 以上。依存は標準ライブラリだけ。

## 何があるか

| 種類 | 中身 |
|---|---|
| 段階 | slog の 4 段階（Debug -4 / Info 0 / Warn 4 / Error 8）に `LevelTrace`（-8）・`LevelNotice`（2）・`LevelEmergency`（32）を足す。名前の読み書きは `ParseLevel` / `LevelName` / `ReplaceLevelName` |
| 出す関数 | `Info` / `Infof` / `InfoContext` など（Trace〜Emergency の 7 段階）。出口は `slog.Default()` |
| ハンドラ | `NewSimpleHandler`（1 行の素朴な書式）・`NewLevelHandler`（レベルで絞る）・`NewPackageLevelHandler`（呼び出し元のパッケージごとに絞る）・`NewRequestHandler`（ctx に載せた属性を足す） |
| ファイル | `RollingFileWriter`（日付などでファイルを切り替える `io.Writer`） |
| 名前付きの Logger | `Named`（Java の Logger 名にあたる。`PackageLevelHandler` がパッケージではなく名前でレベルを決める） |
| その他 | `Func` / `PrintTrace`（関数の出入り）・`PrintStackTrace`（エラーをスタックごと）・`NoneStop`（panic を記録して止める） |

## はじめかた

```go
package main

import (
	"log/slog"
	"os"

	"github.com/wenteasy/log"
)

func main() {
	slog.SetDefault(slog.New(log.NewSimpleHandler(os.Stderr, log.LevelDebug)))

	log.Info("起動しました", "version", "1.0.0")
	log.Debugf("設定を %d 件読みました", 3)
}
```

```
2026-10-04T10:00:00+09:00 [INFO  ] 起動しました version=1.0.0
2026-10-04T10:00:00+09:00 [DEBUG ] 設定を 3 件読みました
```

## 書き方

`Info` などは **slog と同じ「メッセージ + キーと値」**、`Infof` などが **printf**。

```go
log.Info("保存しました", "path", p, "bytes", n)
log.Infof("保存しました: %s (%d bytes)", p, n)
log.WarnContext(ctx, "遅い応答", "ms", ms)
```

- ⚠️ `Info` に書式を渡しても整形されない（`slog.Info` と同じ。`!BADKEY` になる）
- 出さないレベルでは `Infof` の `fmt.Sprintf` も回らない
- 記録される位置（`AddSource` や `PackageLevelHandler` が見るもの）は、このパッケージの中ではなく**呼んだ側**
- 引数を組み立てるのが重いときは `log.Enabled(log.LevelDebug)` で先に確かめる

## 組み立ての例

### ファイルへ（日ごと）

```go
w, err := log.NewRollingFileWriter(dir, log.Day) // dir が無ければ作る
if err != nil { ... }
w.SetPrefix("app")                                // dir/app_20261004.log
defer w.Close()

var lv slog.LevelVar // あとから変えられる
lv.Set(log.LevelDebug)

slog.SetDefault(slog.New(log.NewSimpleHandler(w, &lv)))
```

- 間隔は `Second` / `Minute` / `Hour` / `Day` / `Month` / `Year` / `None`（切り替えない。ファイル名は prefix）
- 既にあるファイルは続きから書き、区切りの行を入れる（閉じるときも 1 行）。JSON で書くなど
  区切りが邪魔なら `DisableMarkers()`
- 並行に書いてよい。古いファイルは消さない（消すのは使う側）

### slog の TextHandler / JSONHandler を使う

足した段階は、そのままだと `DEBUG-4` / `INFO+2` / `ERROR+24` と書かれる。`ReplaceLevelName` で直す。

```go
h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: &lv, ReplaceAttr: log.ReplaceLevelName})
```

### ファイルとコンソールの両方へ（Go 1.26〜）

```go
sink := slog.NewMultiHandler(
	log.NewSimpleHandler(w, log.LevelDebug),        // ファイルは Debug 以上
	log.NewSimpleHandler(os.Stderr, log.LevelInfo), // コンソールは Info 以上
)
slog.SetDefault(slog.New(sink))
```

### パッケージごとのレベル

ログを出した関数のパッケージ（`Record.PC` から引く）ごとにレベルを変える。

```go
body := log.NewSimpleHandler(os.Stderr, log.LevelTrace) // 出口は低くしておく
h := log.NewPackageLevelHandler(body, log.LevelInfo)    // 設定が無ければ Info
if err := h.LoadJSON("logging.json"); err != nil { ... }
slog.SetDefault(slog.New(h))
```

```json
{
  "root": "WARN",
  "packages": [
    {"name": "github.com/x/y", "level": "DEBUG"},
    {"name": "github.com/x/y.(*Server).Handle", "level": "TRACE"},
    {"name": "main.main", "level": "INFO"}
  ]
}
```

- `name` はパッケージのパスか、関数まで含めた名前。**一番深く当たったもの**が効く
- レベルは TRACE / DEBUG / INFO / NOTICE / WARN / ERROR / EMERGENCY（大文字小文字は問わない。`INFO+2` も可）
- 読めないレベル名はエラー（黙って Info にしない）。失敗したときは今の設定のまま
- コードからは `SetLevels(root, map[string]slog.Level{...})`。実行中に変えてよい（派生した Logger にも効く）
- ⚠️ **下げる方向には効かない。** 出口（body）が出さないレベルは、パッケージのレベルを下げても出ない
- 位置を持たない Record（PC が 0）は root で判定する

### 名前付きの Logger（Java の Logger 名）

いつもは `slog` で書き、パッケージの設定（たとえば root を WARN）で絞って運用する。
そのうえで「運用として必ず残したいもの」だけを、名前付きの Logger で書く:

```go
var ops = log.Named("ops") // パッケージの変数に置いてよい（slog.SetDefault より先でもよい）

ops.Info("バッチを始めます", "id", id) // → ... [INFO  ] バッチを始めます logger=ops id=42
```

```json
{
  "root": "WARN",
  "loggers": [
    {"name": "ops", "level": "INFO"}
  ]
}
```

- 名前に設定があれば、**呼び出し元のパッケージに関係なく**名前のレベルで決まる（上げる方向にも下げる方向にも効く）
- 名前に設定が無ければ、名前が無いときと同じくパッケージで決まる
- `"ops.batch"` のように `.` で区切ると、`"ops"` の設定が効く（一番深く当たったものが効く）
- 名前はパッケージの設定とは別の名前空間（`"main"` という名前を付けても `main` パッケージとは混ざらない）
- 出力には `logger=ops` が付く（キーは `log.LoggerKey`）
- コードからは `SetLoggerLevels`、確かめるには `LoggerLevel`
- 自分の `*slog.Logger` に名前を付けるなら `logger.With(log.LoggerAttr("ops"))`。
  ただの `With("logger", "ops")` では名前付きにならない
- ⚠️ `Named` の Logger は書くたびに `slog.Default()` へ渡すもの。`slog.SetDefault` に渡してはいけない（panic する）

動く例は `example/`（`cd example && go run .`）。

### ctx の属性を足す

```go
slog.SetDefault(slog.New(log.NewRequestHandler(body)))

ctx = log.WithAttr(ctx, slog.String("request_id", id))
log.InfoContext(ctx, "受け付けました") // ... request_id=abc
```

### 関数の出入りと panic

```go
func Handle(id string) {
	defer log.PrintTrace(log.Func("Handle", id)) // Trace で "Handle Start args=..." と "Handle End"
	defer log.NoneStop()                         // panic を Emergency で記録して止める（位置とスタックつき）
	...
}
```

## ライブラリのログを絞る

ライブラリの側は `*slog.Logger` を受け取れるようにしておく（下の「使う側のやり方」の `SetLogger`）。
ライブラリごとにレベルを変えたいなら、出口を 1 つにまとめたまま、絞った Logger を配る:

```go
engineLv := new(slog.LevelVar)
engineLv.Set(slog.LevelWarn)
engine.SetLogger(slog.New(log.NewLevelHandler(sink, engineLv)))
```

（ライブラリが slog の既定の Logger に直接出しているなら、`PackageLevelHandler` でも絞れる。）

## 使う側のやり方（自分の log パッケージを作る）

**このパッケージを直接あちこちから呼ばない。** 使うプロジェクトに自分の `log` パッケージ（`<module>/log`）を 1 つ作り、
プロジェクトのコードはすべてそこからログを出す。wenteasy/log を import するのは、その `log` パッケージの中だけにする。
エージェントに作らせるときも、この節のとおりに作らせる（セッションごとに形が変わらないように）。

```
<module>/log/
  log.go     段階（Level*）・出す関数（Info / Infof …）・（ライブラリなら）SetLogger
  setup.go   Init(Options)：出口を組んで slog.Default() を向ける。アプリの main が最初に 1 回呼ぶ
```

### 決まりごと

- ⚠️ **使う側（そのプロジェクトを使う人）に wenteasy/log の型を見せない。** レベルは `slog.Level`、出し先は
  `*slog.Logger` のまま。段階の定数は `const LevelTrace = wlog.LevelTrace` のように値を渡すだけにする。
  ライブラリでも、自分の `log` パッケージの中で import するのは構わない（公開 API に出さなければ、使う人は標準の slog だけで済む）
- ⚠️ **出す関数は自分で書く。** wenteasy/log の `Info` などを関数で包まない。記録される位置を段数で数えているので、
  包むと全部のログが包んだ関数から出たことになり、`AddSource` も `PackageLevelHandler` も効かなくなる。
  `runtime.Callers` で呼び出し元を取り、`slog.NewRecord` をハンドラに渡す（下のひな形）。`slog.Log` も使わない（呼んだ所が記録される）
- ⚠️ **形は slog と同じ「メッセージ + キーと値」（`Info`）と printf（`Infof`）の 2 つ。** 混ぜない
- ⚠️ **出口を決める（`Init`・`slog.SetDefault`）のはアプリの main だけ。** ライブラリのコードからは呼ばない
- ⚠️ **ログが書けなくてもアプリは動かす。** `Init` はファイルを作れなくても出せる出口だけで組み、理由を error で返す
  （アプリは画面などで伝える）。返す `io.Closer` は nil にしない
- ⚠️ **Windows の GUI の exe（`-H windowsgui`。Wails の `wails3 build` など）には標準エラー出力の行き先が無い。**
  配る exe はファイルに書く。開発中（`wails3 dev`・`go run`）はコンソールに出す
- **Logger を引数で配らない**（Service に `*slog.Logger` のフィールドを足さない）。呼び出し元のパッケージは記録の位置から分かり、
  レベルは `PackageLevelHandler` でパッケージごとに絞れる

### アプリだけか、ライブラリとしても使われるか

| | アプリだけ（例: ikkyoku） | ライブラリとしても使われる（例: aibow） |
|---|---|---|
| 出し先 | `slog.Default()` だけ | `SetLogger(*slog.Logger)` で決めた Logger。決めなければ `slog.Default()` |
| 使う人がすること | — | 何もしなくてよい（自分の `slog.SetDefault` の出口に出る）。このライブラリのログだけ別にしたいときは `SetLogger` |
| `Init` | main が呼ぶ | アプリの main が呼ぶ。ライブラリとして使う人は呼ばない |

ライブラリとしても使われるなら `SetLogger` を持たせる（ログの出し先・レベルを、使う人が自分のログと分けられるように）。
そのため、出す関数は `slog.Default()` 決め打ちにせず、`Logger()` を通す。

### ひな形: log.go

```go
// Package log は <名前> のログ。<名前> のコードはここの関数でログを出す（標準の log・slog・wenteasy/log を直接使わない）。
package log

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	wlog "github.com/wenteasy/log"
)

// 値は wenteasy/log と同じ。型は slog.Level なので、使う人は wenteasy/log を知らなくてよい。
const (
	LevelTrace  = wlog.LevelTrace
	LevelDebug  = wlog.LevelDebug
	LevelInfo   = wlog.LevelInfo
	LevelNotice = wlog.LevelNotice
	LevelWarn   = wlog.LevelWarn
	LevelError  = wlog.LevelError
)

// ---- ライブラリとしても使われるときだけ（アプリだけなら消して、Logger() を slog.Default() にする） ----

var logger atomic.Pointer[slog.Logger]

// SetLogger は出し先を決める。nil で slog.Default() に戻る。
func SetLogger(l *slog.Logger) { logger.Store(l) }

// Logger は今の出し先（SetLogger で決めていなければ slog.Default()）。
func Logger() *slog.Logger {
	if l := logger.Load(); l != nil {
		return l
	}
	return slog.Default()
}

// ----

func Enabled(level slog.Level) bool { return Logger().Enabled(context.Background(), level) }

func Trace(msg string, args ...any)  { output(LevelTrace, msg, "", nil, args) }
func Debug(msg string, args ...any)  { output(LevelDebug, msg, "", nil, args) }
func Info(msg string, args ...any)   { output(LevelInfo, msg, "", nil, args) }
func Notice(msg string, args ...any) { output(LevelNotice, msg, "", nil, args) }
func Warn(msg string, args ...any)   { output(LevelWarn, msg, "", nil, args) }
func Error(msg string, args ...any)  { output(LevelError, msg, "", nil, args) }

func Tracef(format string, args ...any)  { output(LevelTrace, "", format, args, nil) }
func Debugf(format string, args ...any)  { output(LevelDebug, "", format, args, nil) }
func Infof(format string, args ...any)   { output(LevelInfo, "", format, args, nil) }
func Noticef(format string, args ...any) { output(LevelNotice, "", format, args, nil) }
func Warnf(format string, args ...any)   { output(LevelWarn, "", format, args, nil) }
func Errorf(format string, args ...any)  { output(LevelError, "", format, args, nil) }

// runtime.Callers → output → 公開関数 → 呼び出し元。⚠️ output は公開関数から直接呼ぶ（間に関数を挟まない）。
const callerSkip = 3

func output(level slog.Level, msg, format string, fargs, attrs []any) {
	l := Logger()
	ctx := context.Background()
	if !l.Enabled(ctx, level) {
		return // 出さないレベルなら Sprintf もしない
	}
	if format != "" {
		msg = fmt.Sprintf(format, fargs...)
	}
	var pcs [1]uintptr
	runtime.Callers(callerSkip, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(attrs...)
	_ = l.Handler().Handle(ctx, r)
}
```

### ひな形: setup.go

```go
package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	wlog "github.com/wenteasy/log"
)

type Options struct {
	Dir     string // ログファイルのフォルダ。空ならファイルに書かない（無ければ作る）
	Console bool   // 標準エラー出力にも出す（開発中だけ立てる。GUI の exe では見えない）
}

const FilePrefix = "<名前>" // <名前>_20261014.log

// Init は出口を組み、slog.Default() をそこへ向ける。アプリの main の最初に 1 回だけ呼ぶ。
// ファイルを作れなくても止めない（出せる出口で組み、理由を返す）。返す io.Closer は nil にしない。
func Init(opts Options) (io.Closer, error) {
	var sinks []slog.Handler
	var closer io.Closer = nopCloser{}
	var err error
	if opts.Dir != "" {
		var w *wlog.RollingFileWriter
		if w, err = wlog.NewRollingFileWriter(opts.Dir, wlog.Day); err == nil {
			w.SetPrefix(FilePrefix)
			sinks = append(sinks, wlog.NewSimpleHandler(w, LevelInfo))
			closer = w
		} else {
			err = fmt.Errorf("ログのフォルダ %s を作れないので、ログはファイルに残りません: %w", opts.Dir, err)
		}
	}
	if opts.Console || len(sinks) == 0 {
		sinks = append(sinks, wlog.NewSimpleHandler(os.Stderr, LevelInfo))
	}
	slog.SetDefault(slog.New(slog.NewMultiHandler(sinks...))) // NewMultiHandler は Go 1.26〜
	return closer, err
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }
```

パッケージごとのレベルにするなら、`sinks` をまとめたものを `NewPackageLevelHandler` で包む（出口は低いレベルで作り、
絞るのはパッケージのレベルに任せる。ikkyoku の `setup.go`）。古いファイルを消す・異常終了（panic）を残す
（`debug.SetCrashOutput`）も ikkyoku の `log` にある。

### main（Wails の例）

```go
func main() {
	logs, logErr := startLog() // 最初に組む（設定の読み込みのログから残すため）
	defer logs.Close()
	// logErr は止めずに、画面のお知らせなどで伝える
	...
	app := application.New(application.Options{
		Logger: slog.Default(), // ⚠️ Wails 自身のログも同じ出口へ（下の説明）
		...
	})
}

// 開発中はコンソール、配る exe はファイル（GUI の exe には標準エラー出力の行き先が無い）。
// isDev は -tags production の有無で見分ける（wails3 build は production を付け、wails3 dev は付けない）。
func startLog() (io.Closer, error) {
	if isDev {
		return log.Init(log.Options{Console: true})
	}
	dir, err := os.UserConfigDir() // Windows なら %APPDATA%
	if err != nil {
		c, _ := log.Init(log.Options{})
		return c, err
	}
	return log.Init(log.Options{Dir: filepath.Join(dir, "<名前>", "logs")})
}
```

⚠️ **Wails v3 には `application.Options.Logger` に `slog.Default()` を渡す。** 渡さないと Wails は `slog.Default()` を使わず、
開発中は自分の書式で標準エラー出力に出し、**配る exe（`production`）では自分のログを捨てる**（v3.0.0-beta.26 の `application.go`）。
WebView2 の行（`[WebView2] …`）は Wails の中で標準の `log.Printf` で書かれるが、`slog.SetDefault` のあとは標準の `log` も
slog の出口に流れるので、こちらは渡さなくても入る（ikkyoku は標準の `log` の出力を名前付きの Logger へ向け直している）。

**置き場所はアプリで決める。** 上の例は `%APPDATA%\<名前>\logs`（aibow）。ikkyoku と binder は一時フォルダ
（`os.TempDir()`）に置いている（配る exe の置き場所に溜めない・何日も前の記録は見ない・アプリ内のログビューアで見る）。
一時フォルダは OS の掃除で消えることがあるので、何日分を残したいか、どこで見るかで決める。
レベルなどの**設定のファイル**は、消えては困るので一時フォルダに置かない（ikkyoku は実行ファイルの隣）。
試しに起動するとき（テストやエージェントの確認）に普段のログと混ざらないよう、場所は環境変数（`<名前>_LOG_DIR` など）で変えられるようにしておくとよい。

### テスト（入れておく歯止め）

- **記録される位置が呼んだ側になること**（自分の `log` を通してもずれない）。`AddSource: true` のハンドラか、
  `runtime.FuncForPC(r.PC)` でテスト関数の名前になっているかを見る
- `Init(Options{Dir: t.TempDir()})` でファイルに書かれること（Windows では `Close` してから一時フォルダを消す）
- ファイルを `Dir` に渡すなど、フォルダを作れなくても止まらず error を返すこと
- テストで出し先を差し替えたら `t.Cleanup` で戻す（`slog.SetDefault(old)` / `SetLogger(nil)`）。出し先は大域なので `t.Parallel` と混ぜない

### よくある間違い

| 間違い | 何が起きるか | 直し方 |
|---|---|---|
| `func Info(f string, a ...any) { wlog.Infof(f, a...) }` と包む | 位置が全部自分の `log` になる。`PackageLevelHandler` が効かない | 上のひな形の `output` |
| `var Info = wlog.Infof` と変数で指す | 位置は正しいが、出し先が `slog.Default()` 決め打ちになり `SetLogger` を持てない | アプリだけで `SetLogger` が要らないときに限る（binder の形）。新しく作るならひな形に揃える |
| 公開関数の引数・戻り値に `wlog.…` の型を使う | 使う人が wenteasy/log を知る必要が出る | `slog.Level` / `*slog.Logger` にする |
| ライブラリのコードで `slog.SetDefault` を呼ぶ | 使う人のログの出口を勝手に変える | 出口を決めるのはアプリの main だけ |
| GUI の exe で標準エラー出力だけに出す | 配った exe のログがどこにも残らない | 配る exe は `Init(Options{Dir: …})` |
| Wails の `Options.Logger` を渡さない | 配る exe で Wails 自身のログ（エラーを含む）が捨てられる | `Logger: slog.Default()` |
| `Info("%d 件", n)` と printf の書式を渡す | 整形されず `!BADKEY` になる | `Infof` を使う |

### 使っているところ

| プロジェクト | 形 |
|---|---|
| `github.com/ShinteLab/ikkyoku`（`ikkyoku/log`） | アプリだけ。`Init(Options)`（一時フォルダのファイル・コンソール・パッケージごとのレベル・全体のレベル `Root`・名前付きの Logger を別のファイルへ振り分け・古いファイルの片付け）と `CatchCrash`。このひな形より先に進んでいる（`log/TODO.md`） |
| `github.com/secondarykey/aibow`（`aibow/log`） | ライブラリとしても使われる。ひな形 + `SetLogger`。`wails3 dev` はコンソール、exe は `%APPDATA%\aibow\logs` |
| binder（`binder/log`） | アプリだけ。出す関数を変数で指す形（`Info` なども printf の形。ひな形とは違う。`log/TODO.md`）。一時フォルダのファイル |

## v0.3.1 での変更（v0.3.0 から）

互換は壊していない（足しただけ）。

- 名前付きの Logger（`Named` / `LoggerAttr` / `LoggerKey`）を足した。`PackageLevelHandler` が
  呼び出し元のパッケージではなく名前でレベルを決める（`SetLoggerLevels` / `LoggerLevel`、`LoadJSON` の `"loggers"`）
- README に使い方を足した

## v0.3.0 での変更（v0.2.0 から）

- `Info` などを printf からキーと値の形に変えた（printf は `Infof` など）
- 記録される位置を呼んだ側にした（以前は全部このパッケージの中になり、`PackageLevelHandler` が効かなかった）
- 大域の `SetDefault` / `Default` / `SetLevel` / `GetLevel` / `Level` / `SetContext` を削った。
  `slog.SetDefault` / `slog.Default` を直接使い、レベルは自分の `slog.LevelVar` を持つ
- `NewSimpleHandler` は `slog.Leveler` を取る（`*slog.LevelVar` はそのまま渡せる）。グループを出すようにした
- `NewPackageLevelHandler(h, root)` は root のレベルを取る。位置の無い Record も root で判定する
  （以前は Info に決め打ち）。出口（body）のレベルも守る。設定の差し替えを並行に安全にした
- `ParseSlogLevel` を `ParseLevel`（エラーを返す）に変えた
- `NewLevelHandler` / `ReplaceLevelName` / `LevelName` / `Emergency` などを足した
- `RollingFileWriter`: `None` で panic する不具合・区切りの文言の取り違え・区切りの行が次の行と繋がる不具合を直した。
  並行に書いてよくなった。ディレクトリを作る。`SetPrefix` / `Path` / `DisableMarkers` を足した
- `golang.org/x/xerrors` をやめた
