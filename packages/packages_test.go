package packages

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestPackage(t *testing.T) {
	tests := []struct {
		path    string
		pkg     *Package
		version string
		pkgpath string
	}{
		{
			path: "gotest.tools",
			pkg: &Package{
				PkgDir:    "",
				ModPrefix: "gotest.tools",
			},
			version: "v3.0.0",
			pkgpath: "gotest.tools/v3",
		},
		{
			path: "gotest.tools/v3",
			pkg: &Package{
				PkgDir:    "",
				ModPrefix: "gotest.tools",
			},
			version: "v2.0.1",
			pkgpath: "gotest.tools/v2",
		},
		{
			path: "gotest.tools/v3/assert/opt",
			pkg: &Package{
				PkgDir:    "assert/opt",
				ModPrefix: "gotest.tools",
			},
			version: "v1.0.0",
			pkgpath: "gotest.tools/assert/opt",
		},
		{
			path: "github.com/go-redis/redis/internal/proto",
			pkg: &Package{
				PkgDir:    "internal/proto",
				ModPrefix: "github.com/go-redis/redis",
			},
			version: "v8.0.0",
			pkgpath: "github.com/go-redis/redis/v8/internal/proto",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			pkg, err := Load(tt.path)
			assert.NilError(t, err)
			assert.DeepEqual(t, pkg, tt.pkg)
			assert.Equal(t, tt.pkg.Path(tt.version), tt.pkgpath)
		})
	}
}

func TestPackage_FindModPath(t *testing.T) {
	tests := []struct {
		pkg     *Package
		path    string
		ok      bool
		modpath string
	}{
		{
			pkg: &Package{
				ModPrefix: "github.com/go-redis/redis",
			},
			path:    "github.com/go-redis/redis/internal/proto",
			ok:      true,
			modpath: "github.com/go-redis/redis",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			modpath, ok := tt.pkg.FindModPath(tt.path)
			assert.Equal(t, ok, tt.ok)
			assert.Equal(t, modpath, tt.modpath)
		})
	}
}
