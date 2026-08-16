package slogkv

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// isAttrArg reports whether arg is a slog.Attr (or an alias of it), which
// log/slog consumes as a whole attribute rather than as a key or a value.
func isAttrArg(pass *analysis.Pass, arg ast.Expr) bool {
	named, ok := types.Unalias(pass.TypesInfo.TypeOf(arg)).(*types.Named)
	return ok && namedPath(named) == slogAttrType
}

// isStringArg reports whether arg's type is string, the one type argsToAttr treats
// as a key taking the next argument as its value. A DEFINED string type is not
// string and does not match, which is why log/slog files it under !BADKEY.
func isStringArg(pass *analysis.Pass, arg ast.Expr) bool {
	basic, ok := types.Unalias(pass.TypesInfo.TypeOf(arg)).(*types.Basic)
	return ok && basic.Info()&types.IsString != 0
}

// mayBeStringArg reports whether arg's static type ADMITS the dynamic type string.
// Only a string takes a second argument with it, so an argument this is false of
// takes just itself whichever branch log/slog puts it down, and that is what makes
// the position after it knowable again.
//
// A type parameter is asked separately because assignability answers the wrong
// question about one: no string may be ASSIGNED to a T, and yet T's type argument
// may BE string. String itself needs no separate question — measured: adding
// `isStringArg(pass, arg) ||` leaves the suite green, assignability already
// covering string, its aliases and the untyped constant.
func mayBeStringArg(pass *analysis.Pass, arg ast.Expr) bool {
	argType := pass.TypesInfo.TypeOf(arg)
	_, parameterised := types.Unalias(argType).(*types.TypeParam)
	return parameterised || types.AssignableTo(types.Typ[types.String], argType)
}

// checkKey reports a key position that is not a constant string.
func checkKey(pass *analysis.Pass, key ast.Expr) {
	if !isConstString(pass, key) {
		pass.Reportf(key.Pos(), messageKeyConst)
	}
}

// isConstString reports whether arg is a constant of string type. An alias of
// string is string; a NAMED string type is not, and log/slog rejects it.
func isConstString(pass *analysis.Pass, arg ast.Expr) bool {
	tv := pass.TypesInfo.Types[arg]
	if tv.Value == nil {
		return false
	}
	basic, ok := types.Unalias(tv.Type).(*types.Basic)
	return ok && basic.Info()&types.IsString != 0
}

// namedPath returns the fully-qualified "pkgpath.Name" of a named type, or "" when
// it has no package (a universe type).
func namedPath(named *types.Named) string {
	if named.Obj().Pkg() == nil {
		return ""
	}
	return named.Obj().Pkg().Path() + "." + named.Obj().Name()
}
