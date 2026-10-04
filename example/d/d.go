package d

import (
	"log/slog"

	"github.com/wenteasy/log"
)

// ops は運用で必ず残したいもの用の名前付き Logger。
// d のパッケージは root（WARN）のままでも、logging.json の "loggers" で ops を INFO にしてあるので出る。
var ops = log.Named("ops")

func Print() {
	slog.Debug("Debug Print d")
	slog.Info("Info Print d")
	slog.Warn("Warn print d")
	ops.Info("Info Print d (ops)")
}
