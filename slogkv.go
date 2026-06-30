// Package slogkv provides a go/analysis analyzer enforcing the gomatic structured-
// logging standard: a leveled slog call (slog.Info/Warn/Error/Debug, or the same
// methods on a *slog.Logger) passes its attributes as constant-string key/value
// pairs — an even number of trailing arguments whose keys are constant strings.
//
// Calls that pass any slog.Attr argument are intentionally not checked: mixing
// pre-built attributes with loose pairs is a valid, harder-to-verify shape, so the
// analyzer skips it rather than risk a false positive.
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
func check(pass *analysis.Pass, call *ast.CallExpr) {
	if !isLeveledSlogCall(pass, call) {
		return
	}
	args := call.Args[1:]
	if hasAttrArg(pass, args) {
		return
	}
	checkPairs(pass, call, args)
}

// isLeveledSlogCall reports whether call invokes a leveled slog function or method.
func isLeveledSlogCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !leveledMethods[sel.Sel.Name] {
		return false
	}
	fn, ok := pass.TypesInfo.ObjectOf(sel.Sel).(*types.Func)
	return ok && fn.Pkg() != nil && fn.Pkg().Path() == slogPkgPath
}

// hasAttrArg reports whether any argument is a slog.Attr, marking the attribute
// shape the analyzer skips.
func hasAttrArg(pass *analysis.Pass, args []ast.Expr) bool {
	for _, arg := range args {
		if named, ok := pass.TypesInfo.TypeOf(arg).(*types.Named); ok && namedPath(named) == slogAttrType {
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
