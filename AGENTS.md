# AGENTS.md

このリポジトリで作業するコーディングエージェント向けの指示。
**使い方は `README.md`。** ここは「触るときに守ること」だけを書く。ドキュメント・コメントは日本語、
識別子と公開 API は英語。

## これは何か

`github.com/wenteasy/log`。`log/slog` を土台にしたログの基盤で、**slog に足りないものだけを足す**
（段階 3 つ・出す関数・ハンドラ 4 つ・`RollingFileWriter`）。独立した Go モジュールで、
独立した git リポジトリ（親の `wenteasy/` はリポジトリではない）。

| ファイル | 中身 |
|---|---|
| `log.go` | 段階（`Level*`）・`ParseLevel` / `LevelName` / `ReplaceLevelName`・出す関数（`Info` / `Infof` / `InfoContext` …）・`Func` / `PrintTrace` / `PrintStackTrace` / `NoneStop` |
| `handler_simple.go` | `NewSimpleHandler`（1 件 1 行。`2006-01-02T15:04:05+09:00 [INFO  ] msg k=v`） |
| `handler_level.go` | `NewLevelHandler`（手前でレベルを絞る） |
| `handler_package.go` | `NewPackageLevelHandler`（呼び出し元のパッケージごと）・`PackageTree` |
| `handler_request.go` | `NewRequestHandler` / `WithAttr` / `WithAttrs`（ctx の属性を足す） |
| `writer.go` | `RollingFileWriter` |
| `example/` | `PackageLevelHandler` を `logging.json` で動かす例（`cd example && go run .`） |

## コマンド

```bash
go build ./...
go vet ./...
go test ./...
cd example && go run .
```

⚠️ この端末では `go test -race` が動かない（race ランタイムの読み込みに失敗する）。
並行性はコードを読んで確かめるしかない。

## ⚠️ 崩さないこと

- ⚠️ **依存は標準ライブラリだけ。** 足さないこと（`xerrors` は v0.3.0 でやめた）
- ⚠️ **go.mod の `go 1.25` より新しい API をコードで使わないこと。** `slog.NewMultiHandler`（1.26）は
  README の例に出すだけで、このパッケージの中では使っていない
- ⚠️ **出す関数の呼び出し元の位置を壊さないこと。** `put` / `putf` は**公開関数から直接**呼び、
  `output` は `runtime.Callers(callerSkip)` で段数を数えている（Callers → output → put → 公開関数 → 呼び出し元）。
  間に関数を 1 つ挟むと、記録の位置も `PackageLevelHandler` の判定も全部このパッケージになる。
  **`slog.Log` を使わないのも同じ理由**（あちらは呼んだ所＝このパッケージを記録する）。
  歯止めは `log_test.go` の `TestCallerIsTheCaller`
- ⚠️ **`Info` などは slog と同じキーと値の形、printf は `…f`。** 混ぜないこと（v0.3.0 で分けた）
- ⚠️ **大域の状態を足さないこと**（大域のレベル・大域の Logger・大域の ctx）。出口は `slog.Default()`、
  レベルは使う側の `slog.LevelVar`。v0.3.0 で `SetLevel` などを削ったのは、
  「そのハンドラを使っていないと効かない」という誤解を招いたため
- ⚠️ **ハンドラは包んだ先の `Enabled` を守ること**（`LevelHandler` / `PackageLevelHandler` は
  `next.Enabled` / `body.Enabled` も見る）。守らないと、包んだ先のレベルが黙って無視される
- ⚠️ **`PackageLevelHandler` の設定は差し替えるだけ**（`atomic.Pointer`。作った木は書き換えない）。
  `WithAttrs` / `WithGroup` で派生したハンドラも同じ設定を見る（`state` を共有）
- ⚠️ **読めないレベル名はエラーにすること**（黙って Info にしない）。`LoadJSON` は失敗したら今の設定を変えない
- ⚠️ **`RollingFileWriter` は並行に書かれる前提**（`mu`）。区切りの行は**行として**書く（次のログと繋げない）。
  `io.Writer` の約束どおり、失敗しても `n` に負の値を返さない
- ⚠️ **`SimpleHandler` は 1 件を 1 回の `Write` で書く**（派生したハンドラと `mu` を共有）。行が混ざらないように

## 使う側のこと

- このパッケージを import するのは**アプリだけ**（ライブラリは `*slog.Logger` を受け取るだけ）。README にも書いてある
- 使っているところ: `github.com/ShinteLab/ikkyoku`（`ikkyoku/log` が土台にしている。タグで引く）

## リリース

タグ（`vX.Y.Z`）を打って push すれば、モジュールプロキシから引ける。v0 のあいだは、
**互換を壊す変更はマイナーを上げる**（v0.2.0 → v0.3.0 がそれ）。README の「v0.X.0 での変更」に書く。
タグを打つのは人（エージェントは打たない・push しない）。

## コミット

Conventional Commits（`feat:` / `fix:` / `docs:` / `refactor:`）。
