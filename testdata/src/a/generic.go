package a

import "log/slog"

// stringish constrains a type parameter to string-like types, so a conversion TO
// the parameter is legal — and that conversion yields the type ARGUMENT's dynamic
// type, not the operand's, which is why it is not the boxing an interface
// conversion is.
type stringish interface{ ~string }

// genericUsage passes a value whose type is a TYPE PARAMETER. types.AssignableTo
// reports that no string may be assigned to one, but the type argument may BE
// string, so the branch log/slog takes is not known and neither is the position
// after it. Both cases below are well-formed records under the instantiation this
// package would give them, and reporting either would be a finding with no remedy:
// the type cannot be narrowed without changing the function's signature.
func genericUsage[T any](extra T) {
	slog.Info("typeparam-attr", slog.Int("n", 1), extra, "k", 1)
	slog.Info("typeparam-then-known", extra, 1, nonConstKey, 2) // want `key must be a constant string`
}

// genericKeyConversion converts to a type parameter rather than to an interface,
// so the conversion is NOT stepped through: T(s) is a T, whose branch is unknown,
// and judging s instead would judge a type the record never sees.
func genericKeyConversion[T stringish](s string) {
	slog.Info("typeparam-conv", T(s), 1)
}
