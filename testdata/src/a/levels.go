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

// oddNonConstKey is odd AND carries a non-constant key in position 0. The
// odd-pairs report is the only one, because the key positions of an unpairable
// call are not knowable; removing the return after that report fails here.
func oddNonConstKey() {
	slog.Info("odd-nonconst", nonConstKey) // want `odd number of key/value arguments`
}
