package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNoPanicAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), CustomAnalyzer, "./...")
}
