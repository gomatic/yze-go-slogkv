package a

import . "log/slog"

// dotImportUsage calls a dot-imported leveled slog function; it resolves to
// log/slog and is checked exactly like a qualified call.
func dotImportUsage() {
	Info("ok", "k", 1)
	Info("odd", "keyonly") // want `odd number of key/value arguments`
}

// methodValueUsage binds a leveled function to a plain variable; the call site
// no longer names a slog entrypoint and is out of scope by design.
func methodValueUsage() {
	f := Info
	f("m", "keyonly")
}
