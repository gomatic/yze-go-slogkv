package a

import "log/slog"

// nonConstKey is a non-constant string used as a slog key (must be flagged).
var nonConstKey = "dynamic"

func usage(err error) {
	slog.Info("ok", "key", 1)
	slog.Info("odd", "key")          // want `odd number of key/value arguments`
	slog.Warn("bad", nonConstKey, 1) // want `key must be a constant string`
	slog.Error("attrs", slog.Int("n", 1))
	slog.Debug("none")
	slog.Info("intkey", 123, 1) // want `key must be a constant string`
	slog.Info("witherr", "k", err)
}

func loggerUsage() {
	l := slog.Default()
	l.Info("ok", "k", 1)
	l.Info("odd", "k") // want `odd number of key/value arguments`
}

// notSlog has an Info method that is not slog's; calls to it must NOT be flagged.
type notSlog struct{}

func (notSlog) Info(string, ...any) {}

func nonSlogCall() {
	var n notSlog
	n.Info("m", "k")
}

// hasField has a func-typed field named Info; a call through it resolves to a
// *types.Var (not a *types.Func) and must NOT be flagged.
type hasField struct {
	Info func(string, ...any)
}

func fieldCall() {
	h := hasField{Info: func(string, ...any) {}}
	h.Info("m", "k")
}

func plainCall() {
	f := func(string) {}
	f("x")
}
