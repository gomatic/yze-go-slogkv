package slogkv_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis/analysistest"

	slogkv "github.com/gomatic/yze-go-slogkv"
)

func TestMalformedSlogPairsAreReported(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), slogkv.Analyzer, "a")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, slogkv.Registration.Validate())
	assert.Equal(t, "yze/slogkv", slogkv.Registration.RuleID())
	assert.Same(t, slogkv.Analyzer, slogkv.Registration.Analyzer)
}
