package a

import "golang.org/x/exp/slog"

func Print() {
	slog.Debug("Debug Print a")
	slog.Info("Info Print a")
	slog.Warn("Warn print a")
}
