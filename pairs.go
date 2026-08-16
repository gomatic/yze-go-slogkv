package slogkv

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// checkPairs walks the loose arguments branch for branch with log/slog's
// argsToAttr, reporting every key that is not a constant string and the call whose
// last key is left without a value.
func checkPairs(pass *analysis.Pass, call *ast.CallExpr, args []ast.Expr) {
	for len(args) > 0 {
		key := unwrapDynamic(pass, args[0])
		switch {
		case isAttrArg(pass, key):
			checkWrittenAttrKey(pass, key)
			args = args[1:]
		case isStringArg(pass, key):
			args = checkStringKey(pass, call, args, key)
		case mayBeStringArg(pass, key):
			args = resumeAfterUnknown(pass, args[1:])
		default:
			checkKey(pass, key)
			args = args[1:]
		}
	}
}

// checkStringKey judges the argument log/slog treats as a key taking the next
// argument as its value, and returns what remains. A string with nothing after it
// is a value filed under !BADKEY rather than a key, so what is wrong there is the
// pairing of the call, and the report sits on the call rather than on an argument.
func checkStringKey(pass *analysis.Pass, call *ast.CallExpr, args []ast.Expr, key ast.Expr) []ast.Expr {
	if len(args) == 1 {
		pass.Reportf(call.Pos(), messageOddPairs)
		return nil
	}
	checkKey(pass, key)
	return args[2:]
}

// resumeAfterUnknown returns the arguments from the next KNOWN key position on,
// having just stepped over an argument whose branch was not determined. Only a
// string takes a second argument with it, so the walk resumes at the argument
// following the first one whose type rules string out — by then log/slog has
// consumed one argument for that one and one for the unknowable one before it,
// whichever branch each took.
func resumeAfterUnknown(pass *analysis.Pass, args []ast.Expr) []ast.Expr {
	for i, arg := range args {
		if !mayBeStringArg(pass, unwrapDynamic(pass, arg)) {
			return args[i+1:]
		}
	}
	return nil
}

// unwrapDynamic returns the expression whose TYPE log/slog's argsToAttr switches
// on. A conversion to an interface boxes its operand and dispatches on the
// operand's own type, so the conversion is stepped through and the operand judged
// in its place — which is what stops any(...) being a five-character way to make an
// argument's role unknowable.
func unwrapDynamic(pass *analysis.Pass, arg ast.Expr) ast.Expr {
	conversion, ok := ast.Unparen(arg).(*ast.CallExpr)
	if !ok || !isInterfaceConversion(pass, conversion) {
		return arg
	}
	return unwrapDynamic(pass, conversion.Args[0])
}

// isInterfaceConversion reports whether call converts its operand to an interface
// type. A conversion to a TYPE PARAMETER is not one of these: it yields the type
// argument's dynamic type rather than the operand's, so stepping through it would
// judge a type the record does not see.
//
// Nothing here asks whether the callee NAMES a type, because only a conversion has
// an interface there — an ordinary call's callee is a signature and a builtin's is
// a builtin — and a redundant conjunct is a guard no case can kill. Measured:
// adding `pass.TypesInfo.Types[call.Fun].IsType() &&` leaves the suite green with
// the corpus unchanged.
func isInterfaceConversion(pass *analysis.Pass, call *ast.CallExpr) bool {
	target := types.Unalias(pass.TypesInfo.TypeOf(call.Fun))
	_, parameterised := target.(*types.TypeParam)
	return !parameterised && types.IsInterface(target)
}
