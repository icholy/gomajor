package packages

import "testing"

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
			t.Parallel()
			pkg, err := Load(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if pkg.ModPrefix != tt.pkg.ModPrefix {
				t.Errorf("wrong ModPrefix: got %q, want %q", pkg.ModPrefix, tt.pkg.ModPrefix)
			}
			if pkgpath := tt.pkg.Path(tt.version); pkgpath != tt.pkgpath {
				t.Errorf("wrong package path: got %q, want %q", pkgpath, tt.pkgpath)
			}
		})
	}
}

func TestPackage_FindModPath(t *testing.T) {
	tests := []struct {
		pkg     *Package
		path    string
		modpath string
	}{
		{
			pkg: &Package{
				ModPrefix: "github.com/go-redis/redis",
			},
			path:    "github.com/go-redis/redis/internal/proto",
			modpath: "github.com/go-redis/redis",
		},
		{
			pkg: &Package{
				ModPrefix: "github.com/go-redis/redis",
			},
			path:    "github.com/go-redis/redis/v8",
			modpath: "github.com/go-redis/redis/v8",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			modpath, ok := tt.pkg.FindModPath(tt.path)
			if !ok {
				t.Fatal("failed to find modpath")
			}
			if modpath != tt.modpath {
				t.Errorf("wrong modpath: got %q, want %q", modpath, tt.modpath)
			}
		})
	}
}
