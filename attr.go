package slogkv

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// elementIndex is a composite literal element's position among its siblings, which
// is what says whether an element written without a field name is the Key.
type elementIndex int

// checkWrittenAttrKey holds an Attr WRITTEN in this argument list to the same rule
// as a loose key, in the two spellings that write one: a constructor's key argument
// and a composite literal's Key field. An Attr arriving any other way — a variable,
// a helper's return value — carries a key its own caller chose, written where this
// call site does not reach.
func checkWrittenAttrKey(pass *analysis.Pass, attr ast.Expr) {
	switch written := ast.Unparen(attr).(type) {
	case *ast.CallExpr:
		checkConstructorKey(pass, written)
	case *ast.CompositeLit:
		checkLiteralKey(pass, written)
	}
}

// checkConstructorKey holds an Attr constructor's declared key parameter to the
// same rule as a loose key. No list of constructor names is needed and none is
// kept: the expression's type is already known to be slog.Attr, and every log/slog
// function returning an Attr is a constructor taking its key first — which is also
// why slog.Value's zero-argument methods sharing those names (v.Group()) do not
// reach here, since they return a Value or a slice.
func checkConstructorKey(pass *analysis.Pass, call *ast.CallExpr) {
	if _, ok := slogFunc(pass, calleeIdent(call.Fun)); !ok || isMultiValueCall(pass, call) {
		return
	}
	checkKey(pass, call.Args[0])
}

// checkLiteralKey holds a slog.Attr composite literal's Key field to the same rule.
// slog.Attr{k, slog.AnyValue(v)} emits the record slog.Any(k, v) emits and is one
// token shorter, so leaving it out would make the constructor rule free to evade.
func checkLiteralKey(pass *analysis.Pass, lit *ast.CompositeLit) {
	for index, element := range lit.Elts {
		if key, ok := attrLiteralKey(elementIndex(index), element); ok {
			checkKey(pass, key)
		}
	}
}

// attrLiteralKey returns the expression a composite-literal element assigns to the
// Attr's Key field: the value of a `Key:` element, or the first element of a
// literal written without field names, slog.Attr's fields being Key then Value.
func attrLiteralKey(index elementIndex, element ast.Expr) (ast.Expr, bool) {
	field, keyed := element.(*ast.KeyValueExpr)
	if !keyed {
		return element, index == 0
	}
	name, named := field.Key.(*ast.Ident)
	return field.Value, named && name.Name == attrKeyField
}
