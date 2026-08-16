package a

import "log/slog"

// unknownUsage passes an argument whose static type is an empty interface. It may
// hold a key, a value or a slog.Attr, and log/slog dispatches on the dynamic type,
// so nothing after it can be judged — but the keys before it still are.
func unknownUsage(extra any) {
	slog.Info("attr-then-unknown", slog.Int("n", 1), extra)
	slog.Info("lone-unknown", extra)
	slog.Info("key-before-unknown", nonConstKey, 1, extra) // want `key must be a constant string`
}
