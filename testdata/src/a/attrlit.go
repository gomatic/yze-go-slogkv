package a

import "log/slog"

// attrLiteralUsage writes the key as an Attr composite literal's Key field. It
// emits the record the constructor emits and is one token shorter, so it is held
// to the same rule; the keyed and unkeyed spellings are both here because a Key:
// element and a first positional element are matched by different code.
func attrLiteralUsage() {
	slog.Info("lit-ok", slog.Attr{Key: "k", Value: slog.IntValue(1)})
	slog.Info("lit-keyed", slog.Attr{Key: nonConstKey, Value: slog.IntValue(1)}) // want `key must be a constant string`
	slog.Info("lit-unkeyed", slog.Attr{nonConstKey, slog.IntValue(1)})           // want `key must be a constant string`
	slog.Info("lit-value-only", slog.Attr{Value: slog.IntValue(1)})
	// Parentheses around the ARGUMENT change nothing about where the key is
	// written, exactly as parentheses around the callee change nothing about which
	// entrypoint is named. Without unwrapping them both spellings below are a
	// five-character escape from the rule.
	slog.Info("lit-parenthesised", (slog.Attr{nonConstKey, slog.IntValue(1)})) // want `key must be a constant string`
	slog.Info("ctor-parenthesised", (slog.Any(nonConstKey, 1)))                // want `key must be a constant string`
}

// attrFromHelper is the FORWARDER shape: its key is a parameter, so the key this
// call names was chosen by whoever called attrFromHelper and there is no remedy
// available here at any price. log/slog's own source is built of this shape —
// func Int(key string, value int) Attr { return Int64(key, int64(value)) } — and
// so is every ReplaceAttr hook rebuilding an attribute from a.Key.
func attrFromHelper(key string) slog.Attr {
	return slog.Any(key, 1)
}

// attrIndirectUsage is the SCOPE LIMITATION, sitting beside the reported spellings
// above so the silence is not mistaken for the analyzer being broken. Neither of
// these argument lists WRITES a key: one names a variable and one calls a helper,
// and following either to the key would take the dataflow this analyzer does not
// do. The identical keys written in place are reported two functions up.
func attrIndirectUsage() {
	built := slog.Any(nonConstKey, 1)
	slog.Info("attr-by-variable", built)
	slog.Info("attr-from-helper", attrFromHelper(nonConstKey))
}
