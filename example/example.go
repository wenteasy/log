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

	lv := log.Level()
	lv.Set(log.LevelTrace)

	h := log.NewSimpleHandler(os.Stdout, lv)
	logger := slog.New(h)
	log.SetDefault(logger)

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
