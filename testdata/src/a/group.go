package a

import "log/slog"

// groupUsage exercises slog.Group, which carries loose pairs behind its own key
// exactly as the leveled entrypoints do — and which is the cheapest way to hide a
// dynamic key inside an Attr the outer call steps over.
func groupUsage() {
	slog.Info("ok", slog.Group("g", "k", 1))
	slog.Info("group-odd", slog.Group("g", "k"))                // want `odd number of key/value arguments`
	slog.Info("group-key", slog.Group("g", nonConstKey, 1))     // want `key must be a constant string`
	slog.Info("group-own-key", slog.Group(nonConstKey, "k", 1)) // want `key must be a constant string`
}

// attrKeyUsage writes the key as an Attr constructor's declared parameter. The
// record log/slog emits is byte-identical to the loose-pair spelling, so the same
// rule applies to it.
func attrKeyUsage() {
	slog.Info("ok", slog.String("k", "v"), slog.Int("n", 1))
	slog.Info("any-key", slog.Any(nonConstKey, 1))         // want `key must be a constant string`
	slog.Info("string-key", slog.String(nonConstKey, "v")) // want `key must be a constant string`
}

// valueMethodUsage calls the zero-argument slog.Value methods that share every
// Attr constructor's name. They are methods, not constructors, and they have no
// key argument to index.
func valueMethodUsage(v slog.Value) {
	_ = v.String()
	_ = v.Any()
	_ = v.Group()
	_ = v.Bool()
}
