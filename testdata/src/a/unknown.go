package a

import "log/slog"

// unknownUsage passes an argument whose static type is an empty interface. It may
// hold a key, a value or a slog.Attr, and log/slog dispatches on the dynamic type,
// so the position after it is not known — but the keys before it still are.
func unknownUsage(extra any) {
	slog.Info("attr-then-unknown", slog.Int("n", 1), extra)
	slog.Info("lone-unknown", extra)
	slog.Info("key-before-unknown", nonConstKey, 1, extra) // want `key must be a constant string`
}

// unknownThenKnown recovers the walk instead of abandoning it. Only a string makes
// log/slog consume TWO arguments, so an argument that cannot be a string consumes
// exactly one whichever branch it takes — whether `extra` was a key taking the 1 as
// its value, or was consumed alone leaving the 1 a bad key, the argument after the
// 1 stands in key position either way and is judged.
//
// The second call is the other side of that boundary: every argument after `extra`
// could be a string, so no position is ever recovered and nothing is reported. It
// is what fails if the recovery ever forgets that a string is string-shaped.
func unknownThenKnown(extra any) {
	slog.Info("resume-after-unknown", extra, 1, nonConstKey, 2) // want `key must be a constant string`
	slog.Info("no-resume", extra, "k", nonConstKey, 2)
}
