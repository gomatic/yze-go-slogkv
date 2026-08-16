package a

import "log/slog"

// namedString is a DEFINED string type. log/slog's own conversion matches the
// string type exactly, so a constant of this type reaches the record as !BADKEY.
type namedString string

const namedKey namedString = "nk"

// aliasString is an ALIAS of string, which IS string, and log/slog accepts it.
type aliasString = string

const aliasKey aliasString = "ak"

func keyUsage(err error) {
	slog.Info("named-key", namedKey, 1) // want `key must be a constant string` `key must be a constant string`
	slog.Info("alias-key", aliasKey, 1)
	// A universe named type in KEY position: error has no package at all, which
	// is the branch namedPath guards against.
	slog.Info("universe-key", err, 1) // want `key must be a constant string` `key must be a constant string`
}

// universeErrorUsage calls Error on the universe error interface, whose method is
// a *types.Func with no package at all. Without the nil-package guard the
// analyzer dereferences nil here.
func universeErrorUsage(err error) string {
	return err.Error()
}
