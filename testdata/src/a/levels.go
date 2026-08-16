package a

import "log/slog"

// levelUsage gives every leveled entrypoint a shape that reports, so dropping any
// one of them from leadingArgs fails this package instead of passing unnoticed.
func levelUsage() {
	slog.Debug("debug-odd", "k")            // want `odd number of key/value arguments`
	slog.Info("info-odd", "k")              // want `odd number of key/value arguments`
	slog.Warn("warn-odd", "k")              // want `odd number of key/value arguments`
	slog.Error("error-odd", "k")            // want `odd number of key/value arguments`
	slog.Debug("debug-key", nonConstKey, 1) // want `key must be a constant string`
	slog.Info("info-key", nonConstKey, 1)   // want `key must be a constant string`
	slog.Warn("warn-key", nonConstKey, 1)   // want `key must be a constant string`
	slog.Error("error-key", nonConstKey, 1) // want `key must be a constant string`
}

// oddNonConstKey is odd AND its one loose argument is the package's non-constant
// key. The odd-pairs report is still the only one, and the reason is the runtime's
// rather than the analyzer's: log/slog files a trailing lone string under !BADKEY
// as a VALUE, so nonConstKey never reaches key position and reporting it as a key
// would be false. Reporting the key as well fails here.
func oddNonConstKey() {
	slog.Info("odd-nonconst", nonConstKey) // want `odd number of key/value arguments`
}
