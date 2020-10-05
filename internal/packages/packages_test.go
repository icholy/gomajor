package packages

import (
	"testing"
)

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
		{
			pkg: &Package{
				PkgDir:    "plumbing",
				ModPrefix: "gopkg.in/src-d/go-git",
			},
			path:    "gopkg.in/src-d/go-git.v4/plumbing",
			modpath: "gopkg.in/src-d/go-git.v4",
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

func TestJoinPathMajor(t *testing.T) {
	tests := []struct {
		modprefix string
		version   string
		modpath   string
	}{
		{
			modprefix: "github.com/google/go-cmp",
			version:   "v0.1.2",
			modpath:   "github.com/google/go-cmp",
		},
		{
			modprefix: "github.com/go-redis/redis",
			version:   "v6.0.1+incompatible",
			modpath:   "github.com/go-redis/redis",
		},
		{
			modprefix: "github.com/go-redis/redis",
			version:   "v8.0.1",
			modpath:   "github.com/go-redis/redis/v8",
		},
		{
			modprefix: "gopkg.in/yaml",
			version:   "v3",
			modpath:   "gopkg.in/yaml.v3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.modpath, func(t *testing.T) {
			modpath := JoinPathMajor(tt.modprefix, tt.version)
			if modpath != tt.modpath {
				t.Fatalf("bad modpath: want %q, got %q", tt.modpath, modpath)
			}
		})
	}
}
