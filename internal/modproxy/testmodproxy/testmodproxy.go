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
	Modules map[string]*Module
}

type Module struct {
	Path     string
	List     []byte
	Versions map[string]*ModuleVersion
}

type ModuleVersion struct {
	Version string
	Mod     []byte
	Zip     []byte
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
	// Generate list data for each module
	for _, mod := range p.Modules {
		var list bytes.Buffer
		for _, version := range slices.SortedFunc(maps.Keys(mod.Versions), semver.Compare) {
			list.WriteString(version)
			list.WriteByte('\n')
		}
		mod.List = list.Bytes()
	}
	return p, nil
}

func (p *ModuleProxy) FS() fs.FS {
	files := fstest.MapFS{}
	for modpath, mod := range p.Modules {
		escaped, err := module.EscapePath(modpath)
		if err != nil {
			continue
		}
		files[escaped+"/@v/list"] = &fstest.MapFile{Data: mod.List}
		for version, ver := range mod.Versions {
			prefix := escaped + "/@v/" + version
			maps.Copy(files, fstest.MapFS{
				prefix + ".mod":  {Data: ver.Mod},
				prefix + ".zip":  {Data: ver.Zip},
				prefix + ".info": {Data: fmt.Appendf(nil, `{"Version":"%s","Time":"2023-01-01T00:00:00Z"}`, version)},
			})
		}
	}
	return files
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
