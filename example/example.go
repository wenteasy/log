// example はパッケージごとのレベル（PackageLevelHandler）を試すもの。
// example ディレクトリで `go run .` すると、logging.json の設定で絞った結果が出る。
// d は名前付きの Logger（log.Named("ops")）も使っていて、パッケージのレベルとは別に出る。
package main

import (
	"log/slog"
	"os"

	"github.com/wenteasy/log"
	"github.com/wenteasy/log/example/a"
	"github.com/wenteasy/log/example/a/b"
	"github.com/wenteasy/log/example/a/c"
	"github.com/wenteasy/log/example/d"
)

func main() {
	// 出口は一番低いレベルにしておき、絞り込みは PackageLevelHandler に任せる
	// （出口が出さないレベルは、パッケージのレベルを下げても出ない）。
	body := log.NewSimpleHandler(os.Stdout, log.LevelTrace)
	h := log.NewPackageLevelHandler(body, log.LevelInfo)
	if err := h.LoadJSON("logging.json"); err != nil {
		log.Error("logging.json を読めません", "err", err)
	}
	slog.SetDefault(slog.New(h))

	log.Trace("Trace main.main")
	log.Debug("Debug main.main")
	log.Info("Info main.main")
	log.Notice("Notice main.main")
	log.Warn("Warn main.main")

	a.Print()
	b.Print()
	b.Print2()
	c.Print()
	d.Print()
}
