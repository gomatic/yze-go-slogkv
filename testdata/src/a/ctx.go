package a

import (
	"context"
	"log/slog"
)

// contextUsage exercises the Context variants and Log, whose loose pairs sit
// behind a fixed prefix the analyzer accounts for.
func contextUsage(ctx context.Context, l *slog.Logger) {
	slog.InfoContext(ctx, "ctx-ok", "k", 1)
	slog.InfoContext(ctx, "ctx-odd", "k")                           // want `odd number of key/value arguments`
	slog.WarnContext(ctx, "ctx-key", nonConstKey, 1)                // want `key must be a constant string`
	slog.DebugContext(ctx, "ctx-debug-odd", "k")                    // want `odd number of key/value arguments`
	slog.ErrorContext(ctx, "ctx-error-odd", "k")                    // want `odd number of key/value arguments`
	l.InfoContext(ctx, "ctx-recv-odd", "k")                         // want `odd number of key/value arguments`
	slog.Log(ctx, slog.LevelInfo, "log-odd", "k")                   // want `odd number of key/value arguments`
	slog.Log(ctx, slog.LevelInfo, "log-key", nonConstKey, 1)        // want `key must be a constant string`
	(*slog.Logger).Log(l, ctx, slog.LevelInfo, "log-expr-odd", "k") // want `odd number of key/value arguments`
}

// logAttrsUsage is the documented scope limitation, pinned beside the in-scope
// siblings above: LogAttrs takes only slog.Attr values, so it has no loose pairs
// to judge and stays silent where the sibling shape reports.
func logAttrsUsage(ctx context.Context, l *slog.Logger) {
	l.LogAttrs(ctx, slog.LevelInfo, "logattrs", slog.Int("n", 1))
}
