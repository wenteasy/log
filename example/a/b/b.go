package b

import "golang.org/x/exp/slog"

func Print() {
	slog.Debug("Debug Print b")
	slog.Info("Info Print b")
	slog.Warn("Warn print b")
}

func Print2() {
	slog.Debug("Debug Print2 b")
	slog.Info("Info Print2 b")
	slog.Warn("Warn print2 b")
}
