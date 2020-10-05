package modproxy

import (
	"testing"

	"github.com/icholy/gomajor/internal/packages"
)

func TestLatest(t *testing.T) {
	mod, err := Latest("github.com/go-redis/redis", true)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Latest %s %s", mod.Path, mod.Latest(false))
}

func TestQuery(t *testing.T) {
	mod, ok, err := Query("github.com/DATA-DOG/go-sqlmock", true)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("not found")
		return
	}
	t.Logf("Latest %s %s", mod.Path, mod.Latest(false))
}

func TestForPackage(t *testing.T) {
	tests := []struct {
		pkgpath string
		modpath string
	}{
		{
			pkgpath: "github.com/go-redis/redis",
			modpath: "github.com/go-redis/redis",
		},
		{
			pkgpath: "github.com/google/go-cmp/cmp",
			modpath: "github.com/google/go-cmp",
		},
		{
			pkgpath: "github.com/go-git/go-git/v5/plumbing/format/commitgraph",
			modpath: "github.com/go-git/go-git/v5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.pkgpath, func(t *testing.T) {
			mod, err := ForPackage(tt.pkgpath, true)
			if err != nil {
				t.Fatal(err)
			}
			if mod.Path != tt.modpath {
				t.Fatalf("invalid modpath, want %q, got %q", tt.modpath, mod.Path)
			}
		})
	}
}

func TestModule(t *testing.T) {
	tests := []struct {
		mod      *Module
		nextpath string
		latest   string
	}{
		{
			mod: &Module{
				Path: "github.com/go-redis/redis",
				Versions: []string{
					"v3.2.30+incompatible",
					"v5.1.2+incompatible",
					"v4.1.11+incompatible",
					"v4.1.3+incompatible",
					"v3.2.13+incompatible",
					"v3.2.22+incompatible",
					"v3.6.4+incompatible",
					"v6.2.3+incompatible",
					"v6.14.1+incompatible",
					"v4.2.3+incompatible",
					"v3.2.16+incompatible",
					"v4.0.1+incompatible",
					"v6.0.0+incompatible",
					"v6.8.2+incompatible",
				},
			},
			latest:   "v6.14.1+incompatible",
			nextpath: "github.com/go-redis/redis/v7",
		},
		{
			mod: &Module{
				Path: "golang.org/x/mod",
				Versions: []string{
					"v0.3.0",
					"v0.1.0",
					"v0.2.0",
				},
			},
			latest:   "v0.3.0",
			nextpath: "",
		},
		{
			mod: &Module{
				Path: "gopkg.in/yaml.v2",
				Versions: []string{
					"v2.2.8",
				},
			},
			latest:   "v2.2.8",
			nextpath: "gopkg.in/yaml.v3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.mod.Path, func(t *testing.T) {
			t.Run("Latest", func(t *testing.T) {
				latest := tt.mod.Latest(false)
				if latest != tt.latest {
					t.Fatalf("wrong latest version, want %q, got %q", tt.latest, latest)
				}
			})
			t.Run("NextMajorPath", func(t *testing.T) {
				nextpath, ok := tt.mod.NextMajorPath()
				if !ok && tt.nextpath != "" {
					t.Fatal("failed to get next major version")
				}
				if nextpath != tt.nextpath {
					t.Fatalf("wrong next path, want %q, got %q", tt.nextpath, nextpath)
				}
			})
		})
	}
}

func TestLoadPackage(t *testing.T) {
	tests := []struct {
		path    string
		pkg     *packages.Package
		version string
		pkgpath string
	}{
		{
			path: "gotest.tools",
			pkg: &packages.Package{
				PkgDir:    "",
				ModPrefix: "gotest.tools",
			},
			version: "v3.0.0",
			pkgpath: "gotest.tools/v3",
		},
		{
			path: "gotest.tools/v3",
			pkg: &packages.Package{
				PkgDir:    "",
				ModPrefix: "gotest.tools",
			},
			version: "v2.0.1",
			pkgpath: "gotest.tools/v2",
		},
		{
			path: "gotest.tools/v3/assert/opt",
			pkg: &packages.Package{
				PkgDir:    "assert/opt",
				ModPrefix: "gotest.tools",
			},
			version: "v1.0.0",
			pkgpath: "gotest.tools/assert/opt",
		},
		{
			path: "github.com/go-redis/redis/internal/proto",
			pkg: &packages.Package{
				PkgDir:    "internal/proto",
				ModPrefix: "github.com/go-redis/redis",
			},
			version: "v8.0.0",
			pkgpath: "github.com/go-redis/redis/v8/internal/proto",
		},
		{
			path: "github.com/go-redis/redis/internal/proto",
			pkg: &packages.Package{
				PkgDir:    "internal/proto",
				ModPrefix: "github.com/go-redis/redis",
			},
			version: "v6.0.1+incompatible",
			pkgpath: "github.com/go-redis/redis/internal/proto",
		},
		{
			path: "gopkg.in/yaml.v1",
			pkg: &packages.Package{
				ModPrefix: "gopkg.in/yaml",
			},
			version: "v2.0.0",
			pkgpath: "gopkg.in/yaml.v2",
		},
		{
			path: "gopkg.in/src-d/go-git.v4/plumbing",
			pkg: &packages.Package{
				PkgDir:    "plumbing",
				ModPrefix: "gopkg.in/src-d/go-git",
			},
			version: "v3.3.1",
			pkgpath: "gopkg.in/src-d/go-git.v3/plumbing",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			pkg, err := LoadPackage(tt.path, false, true)
			if err != nil {
				t.Fatal(err)
			}
			pkg.Version = tt.version
			if pkg.ModPrefix != tt.pkg.ModPrefix {
				t.Errorf("wrong ModPrefix: got %q, want %q", pkg.ModPrefix, tt.pkg.ModPrefix)
			}
			if pkgpath := pkg.Path(); pkgpath != tt.pkgpath {
				t.Errorf("wrong package path: got %q, want %q", pkgpath, tt.pkgpath)
			}
		})
	}
}
