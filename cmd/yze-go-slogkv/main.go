// Command yze-go-slogkv runs the slogkv analyzer as a standalone go/analysis
// checker (text and -json output, and usable as a `go vet -vettool`).
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	slogkv "github.com/gomatic/yze-go-slogkv"
)

// run is the analysis entry point, indirected so the binary's wiring is testable
// without invoking the real driver (which loads packages and exits the process).
var run = singlechecker.Main

func main() { run(slogkv.Analyzer) }
