package packages

import (
	"testing"
)

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

func TestSplitPath(t *testing.T) {
	tests := []struct {
		modprefix string
		pkgpath   string
		pkgdir    string
		modpath   string
		bad       bool
	}{
		{
			pkgpath:   "github.com/go-redis/redis/internal/proto",
			modprefix: "github.com/go-redis/redis",
			modpath:   "github.com/go-redis/redis",
			pkgdir:    "internal/proto",
		},
		{
			pkgpath:   "gopkg.in/src-d/go-git.v4/plumbing",
			modprefix: "gopkg.in/src-d/go-git",
			modpath:   "gopkg.in/src-d/go-git.v4",
			pkgdir:    "plumbing",
		},
		{
			pkgpath:   "github.com/go-redis/redis/v8",
			modprefix: "github.com/go-redis/redis",
			modpath:   "github.com/go-redis/redis/v8",
			pkgdir:    "",
		},
		{
			pkgpath:   "gopkg.in/src-d/go-git.v4/plumbing",
			modprefix: "gopkg.in/src-d/go-git",
			pkgdir:    "plumbing",
			modpath:   "gopkg.in/src-d/go-git.v4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.pkgpath, func(t *testing.T) {
			modpath, pkgdir, ok := SplitPath(tt.modprefix, tt.pkgpath)
			if ok == tt.bad {
				t.Fatalf("bad ok: want %t, got %t", !tt.bad, ok)
			}
			if modpath != tt.modpath {
				t.Errorf("bad modpath: want %q, got %q", tt.modpath, modpath)
			}
			if pkgdir != tt.pkgdir {
				t.Errorf("bad pkgdir: want %q, got %q", tt.pkgdir, pkgdir)
			}
		})
	}
}
