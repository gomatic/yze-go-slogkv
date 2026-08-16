// Package slogkv provides a go/analysis analyzer enforcing the gomatic structured-
// logging standard: a leveled slog call passes its attributes as constant-string
// key/value pairs — every key position holds a constant of string type, and every
// key has a value.
//
// The leveled entrypoints are slog.Debug/Info/Warn/Error, their Context variants
// (slog.DebugContext and the rest), and slog.Log, plus the same methods on a
// *slog.Logger. They differ only in how many arguments precede the loose pairs —
// one for the plain forms, two for the Context forms, three for Log — which the
// analyzer accounts for, so none of them is out of scope. slog.LogAttrs is: its
// trailing arguments are all slog.Attr, so it has no loose pairs to judge.
//
// A slog.Attr argument (or an alias of it) is consumed on its own, exactly as
// log/slog consumes it: it is stepped over and the loose pairs around it are
// still checked. An Attr therefore cannot silence the call it appears in.
//
// A key must be a constant whose type is string or an alias of string. A constant
// of a NAMED string type is reported: log/slog's own conversion matches the string
// type exactly, so a defined type falls through to !BADKEY at runtime.
//
// Spread calls (slog.Info(msg, kvs...)) are skipped — the spread contents are not
// statically knowable. Dot-imported calls and method expressions
// ((*slog.Logger).Info(l, ...)) are checked, and so is a parenthesised callee
// ((slog.Info)(...)), which names the same entrypoint. Method values
// (f := slog.Info; f(...)) are out of scope, because once the function is bound to
// a plain variable the call site no longer names a slog entrypoint.
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

// slogAttrType is the fully-qualified slog.Attr type, which log/slog consumes as a
// whole attribute rather than as a key or a value.
const slogAttrType = "log/slog.Attr"

// leadingArgs maps each leveled log/slog entrypoint to the number of arguments
// that precede its loose key/value pairs: the message alone for the plain forms,
// a context before it for the Context forms, and a context and a level for Log.
// LogAttrs is absent on purpose — its trailing arguments are all slog.Attr, so it
// has no loose pairs.
var leadingArgs = map[string]int{
	"Debug":        1,
	"Info":         1,
	"Warn":         1,
	"Error":        1,
	"DebugContext": 2,
	"InfoContext":  2,
	"WarnContext":  2,
	"ErrorContext": 2,
	"Log":          3,
}

// Analyzer reports malformed key/value arguments to leveled slog calls.
var Analyzer = &analysis.Analyzer{
	Name:     "slogkv",
	Doc:      "reports leveled slog calls whose key/value arguments are unpaired or use a non-constant key",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Registration declares this analyzer to the yze framework. The category is
// "logging", shared with yze/stdlog, so narrowing a run to the logging standard
// yields the whole standard rather than half of it.
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
	leading, ok := leveledPrefix(pass, call)
	if !ok || call.Ellipsis.IsValid() {
		return
	}
	checkPairs(pass, call, call.Args[leading:])
}

// leveledPrefix reports whether call invokes a leveled slog function or method —
// qualified (slog.Info), on a logger (l.Info), as a method expression
// ((*slog.Logger).Info), dot-imported (a bare Info resolving to log/slog), or any
// of those parenthesised — and returns how many of its arguments precede the
// loose key/value pairs.
func leveledPrefix(pass *analysis.Pass, call *ast.CallExpr) (int, bool) {
	id := calleeIdent(call.Fun)
	if id == nil {
		return 0, false
	}
	leading, ok := leadingArgs[id.Name]
	if !ok || !isSlogFunc(pass, id) {
		return 0, false
	}
	return leading + methodExprShift(pass, call), true
}

// isSlogFunc reports whether id resolves to a function declared in log/slog. The
// nil-package guard is load-bearing: "Error" is a leveled name and the universe
// error interface's Error method is a *types.Func with no package at all.
func isSlogFunc(pass *analysis.Pass, id *ast.Ident) bool {
	fn, ok := pass.TypesInfo.ObjectOf(id).(*types.Func)
	return ok && fn.Pkg() != nil && fn.Pkg().Path() == slogPkgPath
}

// calleeIdent returns the identifier naming the called function: the selected
// name of a selector callee, a bare identifier callee (dot imports), or nil for
// any other callee shape. Parentheses are unwrapped first — they change nothing
// about which entrypoint the call site names.
func calleeIdent(fun ast.Expr) *ast.Ident {
	switch f := ast.Unparen(fun).(type) {
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
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return 0
	}
	selection := pass.TypesInfo.Selections[sel]
	if selection != nil && selection.Kind() == types.MethodExpr {
		return 1
	}
	return 0
}

// isAttrArg reports whether arg is a slog.Attr (or an alias of it), which
// log/slog consumes as a whole attribute rather than as a key or a value.
func isAttrArg(pass *analysis.Pass, arg ast.Expr) bool {
	named, ok := types.Unalias(pass.TypesInfo.TypeOf(arg)).(*types.Named)
	return ok && namedPath(named) == slogAttrType
}

// checkPairs walks the loose arguments the way log/slog does — an Attr stands
// alone, anything else is a key expecting a value — reporting the first key with
// no value, and every key that is not a constant string.
func checkPairs(pass *analysis.Pass, call *ast.CallExpr, args []ast.Expr) {
	for len(args) > 0 {
		if isAttrArg(pass, args[0]) {
			args = args[1:]
			continue
		}
		if len(args) == 1 {
			pass.Reportf(call.Pos(), messageOddPairs)
			return
		}
		checkKey(pass, args[0])
		args = args[2:]
	}
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
