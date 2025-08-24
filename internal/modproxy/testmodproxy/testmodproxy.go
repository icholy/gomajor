package testmodproxy

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// ModuleProxy is a bare-bones Go module proxy for offline testing.
// It preloads all module data into memory for fast serving.
type ModuleProxy struct {
	Modules map[string]*Module
}

type Module struct {
	Path     string
	Versions map[string]*ModuleVersion
}

func (m *Module) List() []string {
	var versions []string
	for version := range m.Versions {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) < 0
	})
	return versions
}

type ModuleVersion struct {
	Version string
	Mod     []byte
	Zip     []byte
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
		Modules: make(map[string]*Module),
	}
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
		// Add version
		if p.Modules[modpath] == nil {
			p.Modules[modpath] = &Module{
				Path:     modpath,
				Versions: map[string]*ModuleVersion{},
			}
		}
		p.Modules[modpath].Versions[version] = &ModuleVersion{
			Version: version,
			Mod:     modfile,
			Zip:     zipdata,
		}
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ServeHTTP implements http.Handler
func (p *ModuleProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	modpath, file, ok := strings.Cut(path, "/@v/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	modpath, err := module.UnescapePath(modpath)
	if err != nil {
		http.Error(w, "Invalid module path", http.StatusBadRequest)
		return
	}
	// GET /{module}/@v/list
	if file == "list" {
		mod, ok := p.Modules[modpath]
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		for _, version := range mod.List() {
			fmt.Fprintln(w, version)
		}
		return
	}
	// GET /{module}/@v/{version}.zip
	if version, ok := strings.CutSuffix(file, ".zip"); ok {
		mod, ok := p.Modules[modpath]
		if !ok {
			http.NotFound(w, r)
			return
		}
		ver, ok := mod.Versions[version]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Write(ver.Zip)
		return
	}
	// GET /{module}/@v/{version}.mod
	if version, ok := strings.CutSuffix(file, ".mod"); ok {
		mod, ok := p.Modules[modpath]
		if !ok {
			http.NotFound(w, r)
			return
		}
		ver, ok := mod.Versions[version]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(ver.Mod)
		return
	}
	// GET /{module}/@v/{version}.info
	if version, ok := strings.CutSuffix(file, ".info"); ok {
		mod, ok := p.Modules[modpath]
		if !ok {
			http.NotFound(w, r)
			return
		}
		ver, ok := mod.Versions[version]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"Version":"%s","Time":"2023-01-01T00:00:00Z"}`, ver.Version)
		return
	}
	http.NotFound(w, r)
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
