package a

import "log/slog"

// attrUsage mixes pre-built attributes with loose pairs. An Attr stands alone —
// log/slog consumes it whole — so it is stepped over and the pairs around it are
// still judged; appending an empty Attr therefore cannot silence a call.
func attrUsage() {
	slog.Info("attr-then-pairs", slog.Int("n", 1), "k", 1)
	slog.Info("pairs-then-attr", "k", 1, slog.Int("n", 1))
	slog.Info("attr-forged", nonConstKey, 1, slog.Attr{}, slog.Attr{}) // want `key must be a constant string`
	slog.Info("attr-then-odd", slog.Int("n", 1), "k")                  // want `odd number of key/value arguments`
}
