package c

import "golang.org/x/exp/slog"

func Print() {
	slog.Debug("Debug Print c")
	slog.Info("Info Print c")
	slog.Warn("Warn print c")
}
