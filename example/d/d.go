package d

import "golang.org/x/exp/slog"

func Print() {
	slog.Debug("Debug Print d")
	slog.Info("Info Print d")
	slog.Warn("Warn print d")
}
