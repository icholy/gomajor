package modproxy

import (
	"fmt"
	"testing"
)

func TestLatest(t *testing.T) {
	tests := []string{
		"github.com/go-redis/redis",
		"github.com/russross/blackfriday",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			mod, err := Latest(tt, true)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("Latest %s %s", mod.Path, mod.MaxVersion("", false))
		})
	}
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
	t.Logf("Latest %s %s", mod.Path, mod.MaxVersion("", false))
}

func TestQueryPackage(t *testing.T) {
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
			mod, err := QueryPackage(tt.pkgpath, true)
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
		prefix   string
		max      string
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
			max:      "v6.14.1+incompatible",
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
			max:      "v0.3.0",
			nextpath: "golang.org/x/mod/v2",
		},
		{
			mod: &Module{
				Path: "gopkg.in/yaml.v2",
				Versions: []string{
					"v2.2.8",
				},
			},
			max:      "v2.2.8",
			nextpath: "gopkg.in/yaml.v3",
		},
		{
			mod: &Module{
				Path: "github.com/libp2p/go-libp2p",
				Versions: []string{
					"v0.18.0-rc3",
					"v0.15.0",
					"v0.0.20",
					"v0.0.28",
					"v0.16.0-dev",
					"v0.0.27",
					"v0.9.3",
					"v0.14.4",
					"v0.15.0-rc.1",
					"v0.10.0",
					"v0.7.4",
					"v0.20.0",
					"v0.10.2",
					"v0.24.0",
					"v0.23.0",
					"v0.5.0",
					"v0.18.1",
					"v0.19.3",
					"v0.18.0-rc5",
					"v0.9.5",
					"v0.22.0",
					"v0.18.0-rc2",
					"v0.0.31",
					"v0.18.0",
					"v0.10.3",
					"v0.21.0",
					"v0.19.1",
					"v0.0.5",
					"v0.12.0",
					"v0.19.0",
					"v0.0.29",
					"v0.14.1",
					"v0.1.2",
					"v0.9.0",
					"v0.4.0",
					"v0.4.1",
					"v0.0.10",
					"v0.0.3",
					"v0.5.1",
					"v0.18.0-rc6",
					"v0.24.0-dev",
					"v6.0.23+incompatible",
				},
			},
			max:      "v0.24.0",
			nextpath: "github.com/libp2p/go-libp2p/v2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.mod.Path, func(t *testing.T) {
			t.Run("MaxVersion", func(t *testing.T) {
				max := tt.mod.MaxVersion(tt.prefix, false)
				if max != tt.max {
					t.Fatalf("wrong max version, want %q, got %q", tt.max, max)
				}
			})
			t.Run("NextMajorPath", func(t *testing.T) {
				nextpath, ok := tt.mod.NextMajorPath()
				if !ok && tt.nextpath != "" {
					t.Fatal("failed to get next major version")
				}
				if nextpath != tt.nextpath {
					t.Fatalf("wrong next path: want %q, got %q", tt.nextpath, nextpath)
				}
			})
		})
	}
}

func TestMaxVersion(t *testing.T) {
	tests := []struct {
		lo, hi string
	}{
		{"v0.0.0", "v0.0.1"},
		{"v0.2.0", "v1.0.0"},
		{"v3.0.0+incompatible", "v0.0.1"},
		{"v3.0.0+incompatible", "v5.0.1+incompatible"},
		{"", "v6.14.1+incompatible"},
		{"invalid", ""},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%s < %s", tt.lo, tt.hi)
		t.Run(name, func(t *testing.T) {
			if got := MaxVersion(tt.lo, tt.hi); got != tt.hi {
				t.Fatalf("MaxVersion(%q, %q) = %q", tt.lo, tt.hi, got)
			}
			if got := MaxVersion(tt.hi, tt.lo); got != tt.hi {
				t.Fatalf("MaxVersion(%q, %q) = %q", tt.hi, tt.lo, got)
			}
		})
	}
}
