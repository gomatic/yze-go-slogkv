// Package slogkv provides a go/analysis analyzer enforcing the gomatic structured-
// logging standard: a leveled slog call (slog.Info/Warn/Error/Debug, or the same
// methods on a *slog.Logger) passes its attributes as constant-string key/value
// pairs — an even number of trailing arguments whose keys are constant strings.
//
// Calls that pass any slog.Attr argument (or an alias of it) are intentionally
// not checked: mixing pre-built attributes with loose pairs is a valid,
// harder-to-verify shape, so the analyzer skips it rather than risk a false
// positive. Spread calls (slog.Info(msg, kvs...)) are likewise skipped — the
// spread contents are not statically knowable. Dot-imported calls and method
// expressions ((*slog.Logger).Info(l, ...)) are checked; method values
// (f := slog.Info; f(...)) are out of scope, because once the function is bound
// to a plain variable the call site no longer names a slog entrypoint.
package slogkv

import (
	"go/ast"
	"go/types"

	goyze "github.com/gomatic/go-yze"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	messageOddPairs = "slog call has an odd number of key/value arguments; each key needs a value"
	messageKeyConst = "slog key must be a constant string"
)

// slogPkgPath is the import path whose leveled functions/methods this analyzer checks.
const slogPkgPath = "log/slog"

// slogAttrType is the fully-qualified slog.Attr type whose presence makes a call
// use the attribute shape, which the analyzer skips.
const slogAttrType = "log/slog.Attr"

// leveledMethods are the slog logging entrypoints whose trailing arguments are
// loose key/value pairs (msg followed by pairs). The Context/Log/LogAttrs variants
// have a different argument shape and are out of scope for v1.
var leveledMethods = map[string]bool{
	"Debug": true,
	"Info":  true,
	"Warn":  true,
	"Error": true,
}

// Analyzer reports malformed key/value arguments to leveled slog calls.
var Analyzer = &analysis.Analyzer{
	Name:     "slogkv",
	Doc:      "reports leveled slog calls whose key/value arguments are unpaired or use a non-constant key",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Registration declares this analyzer to the yze framework.
var Registration = goyze.Registration{
	Name:       "slogkv",
	Categories: []goyze.Category{"logging"},
	URL:        "https://docs.gomatic.dev/yze/slogkv",
	Analyzer:   Analyzer,
}

// run checks every leveled slog call in the analyzed package.
func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		check(pass, n.(*ast.CallExpr))
	})
	return nil, nil
}

// check reports a leveled slog call whose loose key/value pairs are malformed.
// Spread calls (call.Ellipsis set) are skipped entirely: the spread contents
// are not statically knowable, so no pairing can be judged.
func check(pass *analysis.Pass, call *ast.CallExpr) {
	if !isLeveledSlogCall(pass, call) || call.Ellipsis.IsValid() {
		return
	}
	args := call.Args[1+methodExprShift(pass, call):]
	if hasAttrArg(pass, args) {
		return
	}
	checkPairs(pass, call, args)
}

// isLeveledSlogCall reports whether call invokes a leveled slog function or
// method — qualified (slog.Info), on a logger (l.Info), as a method expression
// ((*slog.Logger).Info), or dot-imported (a bare Info resolving to log/slog).
func isLeveledSlogCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	id := calleeIdent(call.Fun)
	if id == nil || !leveledMethods[id.Name] {
		return false
	}
	fn, ok := pass.TypesInfo.ObjectOf(id).(*types.Func)
	return ok && fn.Pkg() != nil && fn.Pkg().Path() == slogPkgPath
}

// calleeIdent returns the identifier naming the called function: the selected
// name of a selector callee, a bare identifier callee (dot imports), or nil for
// any other callee shape.
func calleeIdent(fun ast.Expr) *ast.Ident {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel
	case *ast.Ident:
		return f
	default:
		return nil
	}
}

// methodExprShift returns 1 when call invokes a leveled method as a method
// expression ((*slog.Logger).Info(l, ...)), whose first argument is the
// receiver, so the message/pair window shifts by one; otherwise 0.
func methodExprShift(pass *analysis.Pass, call *ast.CallExpr) int {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return 0
	}
	selection := pass.TypesInfo.Selections[sel]
	if selection != nil && selection.Kind() == types.MethodExpr {
		return 1
	}
	return 0
}

// hasAttrArg reports whether any argument is a slog.Attr (or an alias of it),
// marking the attribute shape the analyzer skips.
func hasAttrArg(pass *analysis.Pass, args []ast.Expr) bool {
	for _, arg := range args {
		named, ok := types.Unalias(pass.TypesInfo.TypeOf(arg)).(*types.Named)
		if ok && namedPath(named) == slogAttrType {
			return true
		}
	}
	return false
}

// checkPairs reports an odd argument count, or a key position that is not a
// constant string.
func checkPairs(pass *analysis.Pass, call *ast.CallExpr, args []ast.Expr) {
	if len(args)%2 != 0 {
		pass.Reportf(call.Pos(), messageOddPairs)
		return
	}
	for i := 0; i < len(args); i += 2 {
		if !isConstString(pass, args[i]) {
			pass.Reportf(args[i].Pos(), messageKeyConst)
		}
	}
}

// isConstString reports whether arg is a constant of string type.
func isConstString(pass *analysis.Pass, arg ast.Expr) bool {
	tv := pass.TypesInfo.Types[arg]
	if tv.Value == nil {
		return false
	}
	basic, ok := tv.Type.Underlying().(*types.Basic)
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
