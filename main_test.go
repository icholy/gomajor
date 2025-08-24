package main

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestHelpCommand(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/testscript/help",
	})
}

func TestPathCommand(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/testscript/path",
	})
}