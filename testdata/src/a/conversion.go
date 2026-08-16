package a

import "log/slog"

// keyFor RETURNS a key rather than converting one, so its call is not a conversion
// and nothing is stepped through: the result is judged as it stands.
func keyFor() string { return nonConstKey }

// conversionUsage wraps arguments in a conversion to an interface. any(x) boxes x
// unchanged and log/slog dispatches on x's dynamic type, so the conversion is
// stepped through and the argument is judged as what it actually is. Both sides of
// that boundary are here: a constant key stays silent through the conversion, a
// non-constant one is reported through it, and the pairs AFTER a converted argument
// are still walked instead of being abandoned.
func conversionUsage(raw []byte) {
	slog.Info("conv-const", any("k"), 1)
	slog.Info("conv-nonconst", any(nonConstKey), 1)            // want `key must be a constant string`
	slog.Info("conv-then-key", any("k"), 0, nonConstKey, 1)    // want `key must be a constant string`
	slog.Info("conv-empty-iface", interface{}(nonConstKey), 1) // want `key must be a constant string`
	slog.Info("conv-nested", any(any(nonConstKey)), 1)         // want `key must be a constant string`
	slog.Info("call-key", keyFor(), 1)                         // want `key must be a constant string`
	slog.Info("conv-to-string", string(raw), 1)                // want `key must be a constant string`
	// A conversion to a DEFINED string type is not boxing and is not stepped
	// through: the constant it produces is a namedString, which log/slog files
	// under !BADKEY along with the 1 after it. Stepping through it would judge
	// the untyped constant inside and call the call clean.
	slog.Info("conv-to-named", namedString("k"), 1) // want `key must be a constant string` `key must be a constant string`
	// A BUILTIN in key position: its callee names no type and no signature, which
	// is the shape that reaches the conversion test with nothing to unwrap.
	slog.Info("builtin-key", len(raw), 1) // want `key must be a constant string` `key must be a constant string`
}
