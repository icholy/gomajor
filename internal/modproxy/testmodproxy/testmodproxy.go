package testmodproxy

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing/fstest"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// ModuleProxy is a bare-bones Go module proxy for offline testing.
// It preloads all module data into memory for fast serving.
type ModuleProxy struct {
	fs fstest.MapFS
}

// ServeHTTP implements http.Handler
func (p *ModuleProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, p.FS(), r.URL.Path)
}

// WriteToDir writes the proxy contents to a directory in the format expected by file:// URLs.
func (p *ModuleProxy) WriteToDir(dir string) error {
	return os.CopyFS(dir, p.FS())
}

// Load creates a new TestModuleProxy that serves modules from the given directory.
// The directory structure should be:
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
func Load(rootDir string) (*ModuleProxy, error) {
	p := &ModuleProxy{
		fs: fstest.MapFS{},
	}

	// Track versions per module for generating list files
	versions := make(map[string][]string)

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
		// Load module file
		modfile, err := os.ReadFile(filepath.Join(path, "go.mod"))
		if err != nil {
			return err
		}
		// Create zip file
		zipdata, err := p.zip(path, modpath, version)
		if err != nil {
			return err
		}
		// Escape module path
		escaped, err := module.EscapePath(modpath)
		if err != nil {
			return err
		}
		// Track version for list generation
		versions[escaped] = append(versions[escaped], version)
		// Add files to MapFS
		prefix := escaped + "/@v/" + version
		maps.Copy(p.fs, fstest.MapFS{
			prefix + ".mod":  {Data: modfile},
			prefix + ".zip":  {Data: zipdata},
			prefix + ".info": {Data: fmt.Appendf(nil, `{"Version":"%s","Time":"2023-01-01T00:00:00Z"}`, version)},
		})
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}

	// Generate list files for each module
	for escaped, versions := range versions {
		var list bytes.Buffer
		slices.SortFunc(versions, semver.Compare)
		for _, version := range versions {
			list.WriteString(version)
			list.WriteByte('\n')
		}
		p.fs[escaped+"/@v/list"] = &fstest.MapFile{Data: list.Bytes()}
	}

	return p, nil
}

func (p *ModuleProxy) FS() fs.FS {
	return p.fs
}

func (p *ModuleProxy) zip(dir, modpath, version string) ([]byte, error) {
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
