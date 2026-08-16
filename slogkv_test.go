package slogkv_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	goyze "github.com/gomatic/go-yze"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	slogkv "github.com/gomatic/yze-go-slogkv"
)

// The two diagnostic texts, written here from the package doc comment rather than
// read from the code, so the test and the implementation can disagree.
const (
	wantOddPairs = "slog call has an odd number of key/value arguments; each key needs a value"
	wantKeyConst = "slog key must be a constant string"
)

func TestMalformedSlogPairsAreReported(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), slogkv.Analyzer, "a")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, slogkv.Registration.Validate())
	assert.Equal(t, "yze/slogkv", slogkv.Registration.RuleID())
	assert.Same(t, slogkv.Analyzer, slogkv.Registration.Analyzer)
}

// TestRegistrationCategoryIsLogging pins the category, which nothing else can
// fail on: goyze.Category is a bare string, Registration.Validate checks only
// Name and Analyzer, and a corpus Finding carries no category at all. "logging"
// is the tag this rule shares with yze/stdlog, so narrowing a run to the logging
// standard yields the whole standard rather than half of it.
func TestRegistrationCategoryIsLogging(t *testing.T) {
	assert.Equal(t, []goyze.Category{"logging"}, slogkv.Registration.Categories)
}

// TestDiagnosticsCarryTheContractedMessageAndPosition pins the two things a
// corpus Finding of {rule, path, line} structurally cannot see: the message text
// and the column. The contract comes from the doc comment — a key that is not a
// constant string is reported AT THE KEY, and an unpairable call is reported AT
// THE CALL, because in an unpairable call no key position is knowable.
func TestDiagnosticsCarryTheContractedMessageAndPosition(t *testing.T) {
	results := analysistest.Run(t, analysistest.TestData(), slogkv.Analyzer, "a")
	require.Len(t, results, 1)

	calls, args := callAndArgumentPositions(t, filepath.Join(analysistest.TestData(), "src", "a"))
	fset := results[0].Pass.Fset

	var sawOdd, sawKey bool
	require.NotEmpty(t, results[0].Diagnostics)
	for _, diagnostic := range results[0].Diagnostics {
		at := shortPosition(fset.Position(diagnostic.Pos))
		switch diagnostic.Message {
		case wantOddPairs:
			sawOdd = true
			assert.Contains(t, calls, at, "the odd-pairs diagnostic must sit on the call")
		case wantKeyConst:
			sawKey = true
			assert.Contains(t, args, at, "the key diagnostic must sit on the key argument")
		default:
			t.Fatalf("undocumented diagnostic message %q at %s", diagnostic.Message, at)
		}
	}
	assert.True(t, sawOdd, "no odd-pairs diagnostic, so its position contract went unchecked")
	assert.True(t, sawKey, "no key diagnostic, so its position contract went unchecked")
}

// callAndArgumentPositions parses the fixture package independently of the
// analyzer and returns the set of call positions and the set of positions a key
// may be WRITTEN at, so the assertions above compare against the source rather
// than against a recorded run. A key is written in three places — a loose
// argument, a constructor's argument, and a composite literal's field — and all
// three are collected here because all three are held to the same rule.
func callAndArgumentPositions(t *testing.T, dir string) (map[string]bool, map[string]bool) {
	t.Helper()
	sources, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	require.NotEmpty(t, sources)

	fset := token.NewFileSet()
	calls, keys := map[string]bool{}, map[string]bool{}
	for _, source := range sources {
		file, err := parser.ParseFile(fset, source, nil, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(n ast.Node) bool {
			collectCallPositions(fset, n, calls, keys)
			collectLiteralPositions(fset, n, keys)
			return true
		})
	}
	return calls, keys
}

// collectCallPositions records a call's own position and the position of each of
// its arguments.
func collectCallPositions(fset *token.FileSet, n ast.Node, calls, keys map[string]bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return
	}
	calls[shortPosition(fset.Position(call.Pos()))] = true
	for _, arg := range call.Args {
		keys[shortPosition(fset.Position(arg.Pos()))] = true
	}
}

// collectLiteralPositions records the position of each composite-literal element,
// which for a `Field: value` element is the position of the value, because that is
// where the field's expression is written.
func collectLiteralPositions(fset *token.FileSet, n ast.Node, keys map[string]bool) {
	lit, ok := n.(*ast.CompositeLit)
	if !ok {
		return
	}
	for _, element := range lit.Elts {
		if field, keyed := element.(*ast.KeyValueExpr); keyed {
			element = field.Value
		}
		keys[shortPosition(fset.Position(element.Pos()))] = true
	}
}

// shortPosition renders a position as "file:line:column" with the directory
// dropped, so positions from two different FileSets are comparable.
func shortPosition(position token.Position) string {
	position.Filename = filepath.Base(position.Filename)
	return position.String()
}
