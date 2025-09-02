package testmodproxy

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// Proxy is a named test proxy url.
type Proxy struct {
	Name string
	URL  string
}

// LoadProxies creates an http:// and file:// proxy for testing.
// See LoadFS for input directory format.
func LoadProxies(t *testing.T, rootDir string) []Proxy {
	proxyfs, err := LoadFS(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.FileServer(http.FS(proxyfs)))
	t.Cleanup(func() { server.Close() })
	proxydir := t.TempDir()
	if err := os.CopyFS(proxydir, proxyfs); err != nil {
		t.Fatal(err)
	}
	return []Proxy{
		{Name: "http", URL: server.URL},
		{Name: "file", URL: "file://" + proxydir},
	}
}

// LoadFS creates a virtual filesystem that implements the Go module proxy protocol.
// It scans a directory of module source code and automatically generates all the
// necessary proxy files: /@v/list, /@v/{version}.mod, /@v/{version}.zip, and /@v/{version}.info.
//
// The input directory structure should be:
// rootDir/
//
//	example.com/
//	  module1/
//	    @v/
//	      v1.0.0/
//	        go.mod
//	        main.go
//	        ...
//	      v1.1.0/
//	        go.mod
//	        main.go
//	        ...
//
// The returned filesystem can be used with http.FileServer(http.FS(fsys)) to serve
// a module proxy over HTTP, or with os.CopyFS(dir, fsys) to write the proxy files
// to disk for use with file:// URLs.
func LoadFS(rootDir string) (fs.FS, error) {
	fsys := fstest.MapFS{}
	// Track versions per module for generating list files
	versions := map[string][]string{}
	// Walk the root directory to find all modules
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// Split on /@v/ to separate module path from version
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		modpath, version, ok := strings.Cut(rel, "/@v/")
		if !ok {
			return nil // Must have /@v/ separator
		}
		modfile, err := os.ReadFile(filepath.Join(path, "go.mod"))
		if err != nil {
			return err
		}
		zipdata, err := zipmod(path, modpath, version)
		if err != nil {
			return err
		}
		escaped, err := module.EscapePath(modpath)
		if err != nil {
			return err
		}
		prefix := escaped + "/@v/" + version
		maps.Copy(fsys, fstest.MapFS{
			prefix + ".mod":  {Data: modfile},
			prefix + ".zip":  {Data: zipdata},
			prefix + ".info": {Data: fmt.Appendf(nil, `{"Version":"%s","Time":"2023-01-01T00:00:00Z"}`, version)},
		})
		versions[escaped] = append(versions[escaped], version)
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	for escaped, versions := range versions {
		var list bytes.Buffer
		slices.SortFunc(versions, semver.Compare)
		for _, version := range versions {
			list.WriteString(version)
			list.WriteByte('\n')
		}
		fsys[escaped+"/@v/list"] = &fstest.MapFile{Data: list.Bytes()}
	}
	return fsys, nil
}

func zipmod(dir, modpath, version string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relpath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Create zip entry with module@version prefix
		f, err := zw.Create(modpath + "@" + version + "/" + filepath.ToSlash(relpath))
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		if _, err := io.Copy(f, file); err != nil {
			return err
		}
		return file.Close()
	})
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
