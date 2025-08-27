package modproxy

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/icholy/gomajor/internal/modproxy/testmodproxy"
)

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
			nextpath: "",
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
			nextpath: "",
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

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		old, new     string
		major, newer bool
	}{
		{
			old:   "v1.0.0",
			new:   "v1.0.1",
			newer: true,
		},
		{
			old:   "v1.0.1",
			new:   "v1.0.0",
			newer: false,
		},
		{
			old:   "v1.0.0",
			new:   "v2.0.0",
			major: true,
			newer: true,
		},
		{
			old:   "v1.0.9",
			new:   "v1.4.0",
			major: true,
			newer: false,
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if ok := IsNewerVersion(tt.old, tt.new, tt.major); tt.newer != ok {
				t.Fatalf("IsNewerVersion(%q, %q, %v) = %v", tt.old, tt.new, tt.major, ok)
			}
		})
	}
}

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		v, w string
		want int
	}{
		{v: "v0.0.0", w: "v1.0.0", want: -1},
		{v: "v1.0.0", w: "v0.0.0", want: 1},
		{v: "v0.0.0", w: "v0.0.0", want: 0},
		{v: "v12.0.0+incompatible", w: "v0.0.0", want: -1},
		{v: "", w: "", want: 0},
		{v: "v0.1.0", w: "bad", want: 1},
		{v: "v0.0.0+incompatible", w: "v0.0.0", want: -1},
		{v: "v0.0.0", w: "v0.0.1", want: -1},
		{v: "v0.2.0", w: "v1.0.0", want: -1},
		{v: "v3.0.0+incompatible", w: "v0.0.1", want: -1},
		{v: "v3.0.0+incompatible", w: "v5.0.1+incompatible", want: -1},
		{v: "", w: "v6.14.1+incompatible", want: -1},
		{v: "invalid", w: ""},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := CompareVersion(tt.v, tt.w); got != tt.want {
				t.Fatalf("CompareVersion(%q, %q) = %v, want %v", tt.v, tt.w, got, tt.want)
			}
		})
	}
}

func TestVersionRange(t *testing.T) {
	tests := []struct {
		r    VersionRange
		v    string
		want bool
	}{
		{
			r:    VersionRange{Low: "v0.0.0", High: "v0.0.1"},
			v:    "v0.0.0",
			want: true,
		},
		{
			r:    VersionRange{Low: "v0.0.0", High: "v0.0.1"},
			v:    "v0.0.1",
			want: true,
		},
		{
			r:    VersionRange{Low: "v0.0.0", High: "v0.0.1"},
			v:    "v0.0.2",
			want: false,
		},
		{
			r:    VersionRange{Low: "v0.0.0", High: "v0.0.1"},
			v:    "v0.0.0+incompatible",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := tt.r.Includes(tt.v); got != tt.want {
				t.Fatalf("VersionRange{Low: %q, High: %q}.Includes(%q) = %v, want %v", tt.r.Low, tt.r.High, tt.v, got, tt.want)
			}
		})
	}
}

func TestQuery(t *testing.T) {
	// Start the test mod proxy
	proxy, err := testmodproxy.Load("testdata/modules")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(proxy)
	defer server.Close()

	tests := []struct {
		name    string
		modpath string
		want    *Module
		exist   bool
	}{
		{
			name:    "existing module",
			modpath: "example.com/testmod",
			want: &Module{
				Path:     "example.com/testmod",
				Versions: []string{"v1.0.0", "v1.1.0", "v1.2.0"},
			},
			exist: true,
		},
		{
			name:    "non-existent module",
			modpath: "example.com/nonexistent",
			exist:   false,
		},
	}

	t.Run("http", func(t *testing.T) {
		t.Setenv("GOPROXY", server.URL)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mod, ok, err := Query(tt.modpath, false)
				if err != nil {
					t.Fatal(err)
				}
				if ok != tt.exist {
					t.Fatalf("Query() ok = %v, want %v", ok, tt.exist)
				}
				if !reflect.DeepEqual(mod, tt.want) {
					t.Fatalf("Query() = %+v, want %+v", mod, tt.want)
				}
			})
		}
	})

	t.Run("file", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := proxy.WriteToDir(tmpDir); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GOPROXY", "file://"+tmpDir)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mod, ok, err := Query(tt.modpath, false)
				if err != nil {
					t.Fatal(err)
				}
				if ok != tt.exist {
					t.Fatalf("Query() ok = %v, want %v", ok, tt.exist)
				}
				if !reflect.DeepEqual(mod, tt.want) {
					t.Fatalf("Query() = %+v, want %+v", mod, tt.want)
				}
			})
		}
	})
}

func TestLatest(t *testing.T) {
	// Start the test mod proxy
	proxy, err := testmodproxy.Load("testdata/modules")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(proxy)
	defer server.Close()

	tests := []struct {
		name    string
		modpath string
		pre     bool
		want    *Module
	}{
		{
			name:    "latest from v1 base",
			modpath: "example.com/testmod",
			pre:     false,
			want: &Module{
				Path:     "example.com/testmod/v3",
				Versions: []string{"v3.0.0"},
			},
		},
		{
			name:    "latest from v2 path",
			modpath: "example.com/testmod/v2",
			pre:     false,
			want: &Module{
				Path:     "example.com/testmod/v3",
				Versions: []string{"v3.0.0"},
			},
		},
		{
			name:    "latest from v3 path",
			modpath: "example.com/testmod/v3",
			pre:     false,
			want: &Module{
				Path:     "example.com/testmod/v3",
				Versions: []string{"v3.0.0"},
			},
		},
	}

	t.Run("http", func(t *testing.T) {
		t.Setenv("GOPROXY", server.URL)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mod, err := Latest(tt.modpath, false, tt.pre)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(mod, tt.want) {
					t.Fatalf("Latest() = %+v, want %+v", mod, tt.want)
				}
			})
		}
	})

	t.Run("file", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := proxy.WriteToDir(tmpDir); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GOPROXY", "file://"+tmpDir)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mod, err := Latest(tt.modpath, false, tt.pre)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(mod, tt.want) {
					t.Fatalf("Latest() = %+v, want %+v", mod, tt.want)
				}
			})
		}
	})
}

func TestQueryPackage(t *testing.T) {
	// Start the test mod proxy
	proxy, err := testmodproxy.Load("testdata/modules")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(proxy)
	defer server.Close()
	t.Setenv("GOPROXY", server.URL)

	tests := []struct {
		name    string
		pkgpath string
		want    *Module
	}{
		{
			name:    "module root",
			pkgpath: "example.com/testmod",
			want: &Module{
				Path:     "example.com/testmod",
				Versions: []string{"v1.0.0", "v1.1.0", "v1.2.0"},
			},
		},
		{
			name:    "existing module package",
			pkgpath: "example.com/testmod/v2/pkg",
			want: &Module{
				Path:     "example.com/testmod/v2",
				Versions: []string{"v2.0.0", "v2.1.0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod, err := QueryPackage(tt.pkgpath, false)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(mod, tt.want) {
				t.Fatalf("QueryPackage() = %+v, want %+v", mod, tt.want)
			}
		})
	}
}

