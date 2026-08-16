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

// logAttrsUsage is the documented scope limitation, sitting beside the in-scope
// siblings above so its silence is not mistaken for the analyzer being broken.
//
// It does NOT pin the exclusion, and saying otherwise would be the test that
// blesses a limitation: adding "LogAttrs": 3 to leadingArgs leaves the whole suite
// green, because LogAttrs' trailing arguments are all slog.Attr and the walk
// consumes every one of them either way. The exclusion is unobservable by any case
// this corpus can hold, which is a finding about the clause rather than a gap in
// the case — slogkv.logattrs-exclusion-is-unobservable.
func logAttrsUsage(ctx context.Context, l *slog.Logger) {
	l.LogAttrs(ctx, slog.LevelInfo, "logattrs", slog.Int("n", 1))
}
