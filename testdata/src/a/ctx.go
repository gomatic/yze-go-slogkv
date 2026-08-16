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

// logAttrsUsage carries no loose pairs — every trailing argument is a slog.Attr —
// and its keys are judged all the same, because an Attr WRITTEN in an argument
// list is written there whatever entrypoint the list belongs to. While LogAttrs
// was excluded these three lines were silent and emitted the same record as the
// reported Info spelling, which is the whole of the rule going free at the
// allocation-free entrypoint slog's own documentation recommends.
//
// The constant-key line beside them is the other side of the boundary, and the
// spread is out of scope for the reason every spread is.
func logAttrsUsage(ctx context.Context, l *slog.Logger, attrs []slog.Attr) {
	l.LogAttrs(ctx, slog.LevelInfo, "logattrs", slog.Int("n", 1))
	l.LogAttrs(ctx, slog.LevelInfo, "logattrs-key", slog.String(nonConstKey, "v"))       // want `key must be a constant string`
	slog.LogAttrs(ctx, slog.LevelInfo, "logattrs-fn", slog.Any(nonConstKey, 1))          // want `key must be a constant string`
	slog.LogAttrs(ctx, slog.LevelInfo, "logattrs-lit", slog.Attr{nonConstKey, slog.IntValue(1)}) // want `key must be a constant string`
	l.LogAttrs(ctx, slog.LevelInfo, "logattrs-spread", attrs...)
}
