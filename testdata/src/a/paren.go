package a

import "log/slog"

// parenUsage parenthesises the callee, which changes nothing about which
// entrypoint the call site names.
func parenUsage(l *slog.Logger) {
	(slog.Info)("paren-ok", "k", 1)
	(slog.Info)("paren-odd", "k")                   // want `odd number of key/value arguments`
	(slog.Warn)("paren-key", nonConstKey, 1)        // want `key must be a constant string`
	(l.Info)("paren-recv-odd", "k")                 // want `odd number of key/value arguments`
	((*slog.Logger).Info)(l, "paren-expr-odd", "k") // want `odd number of key/value arguments`
}
