package packages

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		path string
		pkg  *Package
	}{
		{
			path: "gotest.tools",
			pkg: &Package{
				PkgDir:    "",
				ModPrefix: "gotest.tools",
			},
		},
		{
			path: "gotest.tools/v3",
			pkg: &Package{
				PkgDir:    "",
				ModPrefix: "gotest.tools",
			},
		},
		{
			path: "gotest.tools/v3/assert/opt",
			pkg: &Package{
				PkgDir:    "assert/opt",
				ModPrefix: "gotest.tools",
			},
		},
		{
			path: "github.com/go-redis/redis/internal/proto",
			pkg: &Package{
				PkgDir:    "internal/proto",
				ModPrefix: "github.com/go-redis/redis",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			pkg, err := Load(tt.path)
			assert.NilError(t, err)
			assert.DeepEqual(t, pkg, tt.pkg)
		})
	}
}
