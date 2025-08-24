package modproxy

import (
	"testing"
)

func TestLatestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	tests := []string{
		"github.com/go-redis/redis",
		"github.com/russross/blackfriday",
		"github.com/urfave/cli",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			mod, err := Latest(tt, true, true)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("Latest %s %s", mod.Path, mod.MaxVersion("", false))
		})
	}
}

func TestQueryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
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

func TestQueryPackageIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
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
