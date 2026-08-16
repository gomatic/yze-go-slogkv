// Package slogkv provides a go/analysis analyzer enforcing the gomatic structured-
// logging standard: a slog key is written as a constant string, and every key has
// a value.
//
// The entrypoints carrying loose key/value pairs are slog.Debug/Info/Warn/Error,
// their Context variants (slog.DebugContext and the rest), slog.Log and
// slog.Group, plus the same methods on a *slog.Logger. They differ only in how
// many arguments precede the loose pairs — one for the plain forms and for Group,
// two for the Context forms, three for Log — which the analyzer accounts for, so
// none of them is out of scope. slog.LogAttrs is: its trailing arguments are all
// slog.Attr, so it has no loose pairs to judge.
//
// The Attr constructors — slog.String, Int, Int64, Uint64, Float64, Bool, Time,
// Duration, Any and Group — take their key as a declared parameter rather than as
// a loose pair, and that key is held to the same rule. Otherwise slog.Any(k, v) is
// a free way to write exactly the dynamic key the rule exists to forbid, emitting
// a byte-identical record.
//
// A slog.Attr argument is consumed on its own, exactly as log/slog consumes it: it
// is stepped over and the loose pairs around it are still checked. An Attr
// therefore cannot silence the call it appears in.
//
// A key must be a constant whose type is string or an alias of string. A constant
// of a NAMED string type is reported: log/slog's own conversion matches the string
// type exactly, so a defined type falls through to !BADKEY at runtime.
//
// Three shapes are not statically knowable and are skipped for that one reason. A
// spread call (slog.Info(msg, kvs...)) hides its arguments. The f(g()) multi-value
// form has a single syntactic argument standing for every parameter, so no
// argument position names a key. And an argument whose static type is an empty
// interface may hold a key, a value or an Attr — log/slog dispatches on the
// dynamic type — so nothing after it can be judged, and the walk stops there;
// keys before it are still reported.
//
// Dot-imported calls and method expressions ((*slog.Logger).Info(l, ...)) are
// checked, and so is a parenthesised callee ((slog.Info)(...)), which names the
// same entrypoint. Method values (f := slog.Info; f(...)) are out of scope,
// because once the function is bound to a plain variable the call site no longer
// names a slog entrypoint. (*slog.Logger).With and (*slog.Record).Add carry loose
// pairs too and are not yet checked — slogkv.with-and-add-carry-loose-pairs.
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

// slogPkgPath is the import path whose functions/methods this analyzer checks.
const slogPkgPath = "log/slog"

// slogAttrType is the fully-qualified slog.Attr type, which log/slog consumes as a
// whole attribute rather than as a key or a value.
const slogAttrType = "log/slog.Attr"

// entrypoint is the name a call site uses for a log/slog function or method.
type entrypoint string

// leadingArgs maps each log/slog entrypoint whose trailing arguments are loose
// key/value pairs to the number of arguments preceding them. LogAttrs is absent on
// purpose — its trailing arguments are all slog.Attr, so it has no loose pairs.
var leadingArgs = map[entrypoint]int{
	"Debug":        1,
	"Info":         1,
	"Warn":         1,
	"Error":        1,
	"DebugContext": 2,
	"InfoContext":  2,
	"WarnContext":  2,
	"ErrorContext": 2,
	"Log":          3,
	"Group":        1,
}

// attrKeyFuncs are the log/slog Attr constructor FUNCTIONS whose first argument is
// the attribute's key. Every one of these names is also a zero-argument method on
// slog.Value — v.String(), v.Group(), v.Any() and the rest — so a name alone does
// not identify a constructor, and the receiver test in checkAttrKey is what keeps
// them apart.
var attrKeyFuncs = map[entrypoint]bool{
	"String":   true,
	"Int":      true,
	"Int64":    true,
	"Uint64":   true,
	"Float64":  true,
	"Bool":     true,
	"Time":     true,
	"Duration": true,
	"Any":      true,
	"Group":    true,
}

// Analyzer reports malformed key/value arguments to slog calls.
var Analyzer = &analysis.Analyzer{
	Name:     "slogkv",
	Doc:      "reports slog calls whose key/value arguments are unpaired or use a non-constant key",
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

// run checks every slog call in the analyzed package.
func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		check(pass, n.(*ast.CallExpr))
	})
	return nil, nil
}

// check reports a slog call whose keys are malformed, in either place a key is
// written: an Attr constructor's declared key parameter, and the loose pairs of a
// leveled entrypoint.
func check(pass *analysis.Pass, call *ast.CallExpr) {
	fn, ok := slogFunc(pass, calleeIdent(call.Fun))
	if !ok || isMultiValueCall(pass, call) {
		return
	}
	checkAttrKey(pass, fn, call)
	checkLoosePairs(pass, fn, call)
}

// slogFunc returns the log/slog function or method id resolves to. A nil id — the
// callee shape calleeIdent refuses — resolves to no object and falls out of the
// type assertion, so it needs no guard of its own; adding one is dead code no case
// can kill, which was measured rather than assumed.
//
// The nil-PACKAGE guard is the load-bearing one: "Error" is one of the names below
// and the universe error interface's Error method is a *types.Func with no package
// at all, so without it the analyzer dereferences nil on err.Error().
func slogFunc(pass *analysis.Pass, id *ast.Ident) (*types.Func, bool) {
	fn, ok := pass.TypesInfo.ObjectOf(id).(*types.Func)
	return fn, ok && fn.Pkg() != nil && fn.Pkg().Path() == slogPkgPath
}

// isMultiValueCall reports whether call is the f(g()) form, whose single syntactic
// argument carries every parameter, so no argument position names a key. Go allows
// it wherever the arities match, which includes every entrypoint here.
func isMultiValueCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) != 1 {
		return false
	}
	_, tuple := pass.TypesInfo.TypeOf(call.Args[0]).(*types.Tuple)
	return tuple
}

// checkAttrKey reports a non-constant key handed to an Attr constructor. The key
// is a declared parameter rather than a loose pair, and it is a slog key all the
// same. The receiver test is not cosmetic: slog.Value carries a zero-argument
// method for every constructor name, so keying on the name alone reads v.Group()
// as a constructor call and indexes an empty argument list.
func checkAttrKey(pass *analysis.Pass, fn *types.Func, call *ast.CallExpr) {
	if fn.Signature().Recv() != nil || !attrKeyFuncs[entrypoint(fn.Name())] {
		return
	}
	checkKey(pass, call.Args[0])
}

// checkLoosePairs walks the trailing key/value pairs of an entrypoint that has
// them, past however many arguments precede them.
//
// The length guard earns its place twice over: slog.Value.Group() is a real
// zero-argument call whose name is in the map, and the f(g()) form packs every
// parameter into one syntactic argument. Both index past the end without it, and
// both were found by running the analyzer over the fleet rather than by reading.
func checkLoosePairs(pass *analysis.Pass, fn *types.Func, call *ast.CallExpr) {
	leading, ok := leadingArgs[entrypoint(fn.Name())]
	if !ok {
		return
	}
	leading += methodExprShift(pass, call)
	if call.Ellipsis.IsValid() || len(call.Args) <= leading {
		return
	}
	checkPairs(pass, call, call.Args[leading:])
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

// isUnknownArg reports whether arg's static type leaves its role unknowable: an
// empty interface may hold a key, a value or a slog.Attr, and log/slog dispatches
// on the dynamic type. An interface a string is unassignable to — error is the
// common one — stays knowable: it is no constant string, so it is a bad key.
func isUnknownArg(pass *analysis.Pass, arg ast.Expr) bool {
	argType := pass.TypesInfo.TypeOf(arg)
	return types.IsInterface(argType) && types.AssignableTo(types.Typ[types.String], argType)
}

// checkPairs walks the loose arguments the way log/slog does — an Attr stands
// alone, anything else is a key expecting a value — reporting the first key with
// no value, and every key that is not a constant string. It stops at the first
// argument whose role is unknowable, because nothing past it can be judged.
func checkPairs(pass *analysis.Pass, call *ast.CallExpr, args []ast.Expr) {
	for len(args) > 0 {
		if isUnknownArg(pass, args[0]) {
			return
		}
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
