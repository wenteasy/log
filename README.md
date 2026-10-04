# log

`log/slog` を土台にしたログの基盤。slog の代わりではなく、slog に足りないものだけを足す。

| 足すもの | 中身 |
|---|---|
| 段階 | `LevelTrace`（-8）・`LevelNotice`（2）・`LevelEmergency`（32）。名前の読み書きは `ParseLevel` / `LevelName` / `ReplaceLevelName` |
| パッケージ関数 | `Info` / `Infof` / `InfoContext` など（Trace〜Emergency の 7 段階）。出力先は `slog.Default()` |
| ハンドラ | `NewSimpleHandler`（1 行の素朴な書式）・`NewLevelHandler`（レベルで絞る）・`NewPackageLevelHandler`（呼び出し元のパッケージごとに絞る）・`NewRequestHandler`（ctx に載せた属性を足す） |
| ファイル | `RollingFileWriter`（日付などでファイルを切り替える `io.Writer`） |
| 名前付きの Logger | `Named`（Java の Logger 名にあたる。`PackageLevelHandler` がパッケージではなく名前でレベルを決める） |
| その他 | `Func` / `PrintTrace`（関数の出入り）・`PrintStackTrace`・`NoneStop`（panic を記録して止める） |

## ライブラリから import しないこと

このパッケージを使うのは**アプリ**（main と、その周りのアプリ専用のパッケージ）だけにする。
**ライブラリは `*slog.Logger` を受け取るだけ**にし、レベルの型やログの設定を独自に持たない。
`slog.SetDefault` もライブラリからは呼ばない。

そうしておけば、使う側は標準の slog だけでライブラリのログを扱える。
ライブラリごとにレベルを変えたいなら、出口を 1 つにまとめたまま、絞った Logger を配る:

```go
sink := slog.NewMultiHandler(fileHandler, stderrHandler) // 出口（Go 1.26〜）

engineLv := new(slog.LevelVar)
engineLv.Set(slog.LevelWarn)
engine.SetLogger(slog.New(log.NewLevelHandler(sink, engineLv)))
```

`slog.NewMultiHandler` は「1 件を複数の出口へ配る」もの（ファイルは Debug 以上、標準エラーは Info 以上、など）。
ライブラリごとのレベルは `NewLevelHandler` の役目で、両方を組み合わせて使う。

## 書き方

`Info` などは **slog と同じ「メッセージ + キーと値」**、`Infof` などが **printf**。

```go
log.Info("保存しました", "path", p, "bytes", n)
log.Infof("保存しました: %s (%d bytes)", p, n)
```

⚠️ `Info` に書式を渡しても整形されない（`slog.Info` と同じ）。

記録される位置（`AddSource` や `PackageLevelHandler` が見るもの）は、このパッケージの中ではなく**呼んだ側**になる。

## 組み立ての例

```go
w, err := log.NewRollingFileWriter(dir, log.Day) // dir/app_20260102.log
if err != nil { ... }
w.SetPrefix("app")
defer w.Close()

var lv slog.LevelVar // あとから変えられる
lv.Set(log.LevelDebug)

h := log.NewSimpleHandler(w, &lv)
// slog の TextHandler / JSONHandler を使うなら、段階の名前を直す:
//   slog.NewTextHandler(w, &slog.HandlerOptions{Level: &lv, ReplaceAttr: log.ReplaceLevelName})
slog.SetDefault(slog.New(h))
```

### パッケージごとのレベル

```go
body := log.NewSimpleHandler(os.Stderr, log.LevelTrace) // 出口は低くしておく
h := log.NewPackageLevelHandler(body, log.LevelInfo)
if err := h.LoadJSON("logging.json"); err != nil { ... }
slog.SetDefault(slog.New(h))
```

```json
{
  "root": "WARN",
  "packages": [
    {"name": "github.com/x/y", "level": "DEBUG"},
    {"name": "main.main", "level": "TRACE"}
  ]
}
```

- `name` はパッケージのパスか、関数まで含めた名前。一番深く当たったものが効く
- ⚠️ 下げる方向には効かない。出口（body）が出さないレベルは、パッケージのレベルを下げても出ない
- 読めないレベル名はエラー（黙って Info にしない）。失敗したときは今の設定のまま
- `SetLevels` でコードから設定してもよい。実行中に変えてよい

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

## 以前の版からの変更

- `Info` などを printf からキーと値の形に変えた（printf は `Infof` など）
- 記録される位置を呼んだ側にした（以前は全部このパッケージの中になり、`PackageLevelHandler` が効かなかった）
- 大域の `SetDefault` / `Default` / `SetLevel` / `GetLevel` / `Level` / `SetContext` を削った。
  `slog.SetDefault` / `slog.Default` を直接使い、レベルは自分の `slog.LevelVar` を持つ
- `NewSimpleHandler` は `slog.Leveler` を取る（`*slog.LevelVar` はそのまま渡せる）。グループを出すようにした
- `NewPackageLevelHandler(h, root)` は root のレベルを取る。位置の無い Record も root で判定する
  （以前は Info に決め打ち）。出口（body）のレベルも守る。設定の差し替えを並行に安全にした
- `ParseSlogLevel` を `ParseLevel`（エラーを返す）に変えた
- `RollingFileWriter`: `None` で panic する不具合・区切りの文言の取り違え・区切りの行が次の行と繋がる不具合を直した。
  並行に書いてよくなった。ディレクトリを作る。`SetPrefix` / `Path` / `DisableMarkers` を足した
- `golang.org/x/xerrors` をやめた
