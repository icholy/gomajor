package importpaths

import (
	"bytes"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestRewriteFile(t *testing.T) {
	tests := []struct {
		input   string
		expect  string
		replace ReplaceFunc
	}{
		{
			input:  "testdata/a.go",
			expect: "testdata/a_expect_0.go",
			replace: func(pos token.Position, path string) (string, error) {
				return "", ErrSkip
			},
		},
		{
			input:  "testdata/a.go",
			expect: "testdata/a_expect_1.go",
			replace: func(pos token.Position, path string) (string, error) {
				if path == "github.com/foo/a" {
					return "github.com/bar/a", nil
				}
				return "", ErrSkip
			},
		},
		{
			input:  "testdata/a.go",
			expect: "testdata/a_expect_2.go",
			replace: func(pos token.Position, path string) (string, error) {
				if path == "github.com/fix/b" {
					return "github.com/bug/b", nil
				}
				return "", ErrSkip
			},
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "a.go")
			input, err := os.ReadFile(tt.input)
			if err != nil {
				t.Fatalf("read input: %v", err)
			}
			if err := os.WriteFile(name, input, 0755); err != nil {
				t.Fatalf("write input: %v", err)
			}
			if err := RewriteFile(name, tt.replace); err != nil {
				t.Fatalf("rewrite: %v", err)
			}
			actual, err := os.ReadFile(name)
			if err != nil {
				t.Fatalf("read actual: %v", err)
			}
			expect, err := os.ReadFile(tt.expect)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}
			if !bytes.Equal(actual, expect) {
				t.Fatalf("expected:\n---\n%s\n--\nactual:\n---\n%s\n---\n", expect, actual)
			}
		})
	}
}
