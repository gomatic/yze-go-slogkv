package a

import "log/slog"

// badKeyUsage puts an argument log/slog cannot use as a key in key position. It is
// filed under !BADKEY and consumes ONE argument, not two, so the pairs after it are
// NOT shifted: "k" is still a key and its value is still a value. Consuming two
// there reports the value as a key and calls a fully paired call unpairable, which
// is what these cases fail on.
func badKeyUsage(err error) {
	slog.Info("badkey-then-pair", 1, "k", 2)               // want `key must be a constant string`
	slog.Info("badkey-shifts", 1, "k", err, "b", 2)        // want `key must be a constant string`
	slog.Info("badkey-then-key", 1, nonConstKey, 2)        // want `key must be a constant string` `key must be a constant string`
	slog.Info("badkey-runs", 1, 2)                         // want `key must be a constant string` `key must be a constant string`
	slog.Info("badkey-after-pair", "k", 1, namedKey, "v2") // want `key must be a constant string` `odd number of key/value arguments`
}
