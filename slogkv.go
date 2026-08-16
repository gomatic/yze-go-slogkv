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
// # The walk is log/slog's own, branch for branch
//
// argsToAttr ($GOROOT/src/log/slog/record.go) has THREE branches and the analyzer
// has the same three. A string is a key and takes the NEXT argument as its value,
// so it consumes two — and a string with nothing after it is not a key at all, but
// a value filed under !BADKEY, which is the unpairable call. A slog.Attr is a whole
// attribute and consumes ONE, so an Attr can never silence the pairs around it.
// ANYTHING ELSE is a bad key: log/slog gives it !BADKEY and consumes ONE, not two.
// Consuming two there desynchronises every later position from the record the
// program actually emits, which is how a value gets reported as a key and a fully
// paired call gets reported as unpairable.
//
// The walk does not stop at a bad key, and reports every later key position too.
// Because the consumption matches, those positions are the ones log/slog itself
// keys on — slog.Info("m", 1, 2) really does emit two !BADKEY entries — so each
// report names a real key and not a consequence of an earlier one. go vet stops
// there ("so we report at most one missing key per call"); stopping would be a
// fourth rule this walk does not have, and it hides the dynamic keys after it.
//
// # Where a key is written
//
// A key is written in three spellings and all three are held to the same rule,
// because the record log/slog emits is identical for all three: a loose pair, an
// Attr constructor's key argument (slog.Any(k, v)), and an Attr composite literal's
// Key field (slog.Attr{k, slog.AnyValue(v)}, one token shorter than the
// constructor). Leaving any of them out makes the other two free to evade.
//
// The constructor and literal spellings are judged WHERE THEY ARE WRITTEN — in the
// argument list of a slog call. An Attr arriving any other way carries a key its
// own call site did not write: a forwarder (func Int(key string, value int) Attr {
// return Int64(key, int64(value)) }) and a ReplaceAttr hook rebuilding an attribute
// (slog.String(a.Key, ...)) both name a key that belongs to their CALLER, and no
// remedy exists at the forwarder for a key the caller chose. log/slog's own source
// is built of both shapes. The cost of this scope is stated rather than hidden: an
// Attr built into a variable or returned by a helper is not judged, so the rule is
// evaded by a function — which is a function more than the composite literal used
// to cost.
//
// # What cannot be judged, and what can
//
// A key must be a constant whose type is string or an alias of string. A constant
// of a NAMED string type is a bad key: log/slog's conversion matches the string
// type exactly, so a defined type falls through to !BADKEY at runtime.
//
// A spread call (slog.Info(msg, kvs...)) hides its arguments, and the f(g())
// multi-value form has a single syntactic argument standing for every parameter, so
// neither names a key position at all.
//
// An argument whose static type admits the dynamic type string — an empty
// interface, a type parameter — could take any of the three branches, so the
// positions after it are not known. They are RECOVERED rather than abandoned: an
// argument that cannot be a string consumes exactly one whichever branch it takes,
// so the argument after it stands in key position again however the unknowable one
// was consumed, and every key from there on is judged. A conversion is stepped
// through first: any(x) boxes x unchanged and dispatches on x's type, so wrapping an
// argument in any(...) does not make its role unknowable. What remains unknowable is
// a value that reached an interface variable elsewhere, which needs dataflow to
// resolve and is out of scope.
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

// attrKeyField is the name of the slog.Attr field holding the attribute's key, and
// the field a composite literal writes first when it writes no field names.
const attrKeyField = "Key"

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

// check judges the loose key/value pairs of a leveled slog entrypoint. The keys
// written as Attr constructors and literals inside those pairs are judged there,
// by the walk, rather than wherever an Attr happens to be built.
func check(pass *analysis.Pass, call *ast.CallExpr) {
	fn, ok := slogFunc(pass, calleeIdent(call.Fun))
	if !ok {
		return
	}
	checkLoosePairs(pass, fn, call)
}

// slogFunc returns the log/slog function or method id resolves to. A nil id — the
// callee shape calleeIdent refuses — resolves to no object and falls out of the
// type assertion, so it needs no guard of its own; adding one is dead code no case
// can kill, which was measured rather than assumed.
//
// The nil-PACKAGE guard is the load-bearing one: "Error" is one of the entrypoint
// names and the universe error interface's Error method is a *types.Func with no
// package at all, so without it the analyzer dereferences nil on err.Error().
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
