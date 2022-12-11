package main

import (
	"os"

	"golang.org/x/exp/slog"

	"github.com/wenteasy/log"
	"github.com/wenteasy/log/example/a"
	"github.com/wenteasy/log/example/a/b"
	"github.com/wenteasy/log/example/a/c"
	"github.com/wenteasy/log/example/d"
)

//main.main is DEBUG
//time=2022-12-11T15:55:59.807+09:00 level=DEBUG msg="Debug main.main"
//time=2022-12-11T15:55:59.849+09:00 level=INFO msg="Info main.main"
//time=2022-12-11T15:55:59.849+09:00 level=WARN msg="Warn main.main"

//main.Pring is WARN
//time=2022-12-11T15:55:59.849+09:00 level=WARN msg="Warn main.Print"

//a Package is INFO
//time=2022-12-11T15:55:59.849+09:00 level=INFO msg="Info Print a"
//time=2022-12-11T15:55:59.849+09:00 level=WARN msg="Warn print a"

//b Package Print() is INFO
//time=2022-12-11T15:55:59.849+09:00 level=INFO msg="Info Print b"
//time=2022-12-11T15:55:59.849+09:00 level=WARN msg="Warn print b"

//b Package Print2() is WARN
//time=2022-12-11T16:01:02.702+09:00 level=WARN msg="Warn print2 b"

//c Package is not found.but parent package a isINFO
//time=2022-12-11T15:55:59.849+09:00 level=INFO msg="Info Print c"
//time=2022-12-11T15:55:59.849+09:00 level=WARN msg="Warn print c"

//d Package is not found.using ROOT level(WARN)
//time=2022-12-11T15:55:59.849+09:00 level=WARN msg="Warn print d"

func main() {

	h := slog.NewTextHandler(os.Stdout)
	pkgH := log.NewPackageLevelHandler(h)
	pkgH.LoadJSON("logging.json")

	slog.SetDefault(slog.New(pkgH))

	slog.Debug("Debug main.main")
	slog.Info("Info main.main")
	slog.Warn("Warn main.main")

	Print()

	a.Print()
	b.Print()
	b.Print2()
	c.Print()
	d.Print()
}

func Print() {
	slog.Debug("Debug main.Print")
	slog.Info("Info main.Print")
	slog.Warn("Warn main.Print")
}
