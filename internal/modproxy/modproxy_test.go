package modproxy

import (
	"testing"
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
