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

## ライブラリから import しないこと

このパッケージを使うのは**アプリ**（main と、その周りのアプリ専用のパッケージ）だけにする。
**ライブラリは `*slog.Logger` を受け取るだけ**にし、レベルの型やログの設定を独自に持たない。
`slog.SetDefault` もライブラリからは呼ばない。

そうしておけば、使う側は標準の slog だけでライブラリのログを扱える。
ライブラリごとにレベルを変えたいなら、出口を 1 つにまとめたまま、絞った Logger を配る:

```go
engineLv := new(slog.LevelVar)
engineLv.Set(slog.LevelWarn)
engine.SetLogger(slog.New(log.NewLevelHandler(sink, engineLv)))
```

（ライブラリが slog の既定の Logger に直接出しているなら、`PackageLevelHandler` でも絞れる。）

## アプリで自分の log パッケージを作るとき

`Info` などを**関数で包まないこと**。記録される位置を段数で数えているので、包むと
全部のログが包んだ関数から出たことになる（`PackageLevelHandler` も効かなくなる）。
自分のパッケージに出す関数を置きたいなら、`runtime.Callers` で呼び出し元を取り、
`slog.NewRecord` を `slog.Default().Handler().Handle` に渡す形で書く（`log.go` の `output` と同じ）。
ハンドラ・段階・`RollingFileWriter` はそのまま使ってよい。

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
