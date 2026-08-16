package a

import (
	"context"
	"log/slog"
)

func ctxAndMsg() (context.Context, string) { return context.Background(), "multi" }

func logAll() (context.Context, slog.Level, string) {
	return context.Background(), slog.LevelInfo, "multi"
}

func keyAndValue() (string, any) { return "k", 1 }

// multiValueUsage uses the f(g()) form, whose ONE syntactic argument carries every
// parameter. No argument position names a key, so nothing here can be judged —
// and indexing past the leading count would panic.
func multiValueUsage() {
	slog.InfoContext(ctxAndMsg())
	slog.Log(logAll())
	slog.Info("multi-attr", slog.Any(keyAndValue()))
}
