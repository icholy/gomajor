// Package importpaths implements import path rewriting
// credit: https://gist.github.com/jackspirou/61ce33574e9f411b8b4a
package importpaths

import (
	"errors"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrSkip is used to signal that an import should be skipped
var ErrSkip = errors.New("skip import")

// ReplaceFunc is called with every import path and returns the replacement path
// if the second return parameter is false, the replacement doesn't happen
type ReplaceFunc func(pos token.Position, path string) (string, error)

// Rewrite takes a directory path and a function for replacing imports paths
// Note: .git, vendor, and submodule directories are skipped.
func Rewrite(dir string, replace ReplaceFunc) error {
	return filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
		// check errors
		if err != nil {
			log.Println("import rewrite:", err)
			return nil
		}
		// skip directories
		if info.IsDir() {
			// don't recurse into vendor or .git directories
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			// don't recurse into sub-modules
			if name != dir {
				_, err := os.Lstat(filepath.Join(name, "go.mod"))
				if err == nil {
					return filepath.SkipDir
				}
				if !os.IsNotExist(err) {
					log.Println("import rewrite:", err)
				}
			}
			return nil
		}
		// check the file is a .go file.
		if strings.HasSuffix(name, ".go") {
			return RewriteFile(name, replace)
		}
		return nil
	})
}

// RewriteFile rewrites import statments in the named file
// according to the rules supplied by the map of strings.
func RewriteFile(name string, replace ReplaceFunc) error {
	// create an empty fileset.
	fset := token.NewFileSet()
	// parse the .go file.
	// we are parsing the entire file with comments, so we don't lose anything
	// if we need to write it back out.
	f, err := parser.ParseFile(fset, name, nil, parser.ParseComments)
	if err != nil {
		e := err.Error()
		msg := "expected 'package', found 'EOF'"
		if e[len(e)-len(msg):] == msg {
			return nil
		}
		return err
	}
	// iterate through the import paths. if a change occurs update bool.
	change := false
	for _, i := range f.Imports {
		// unquote the import path value.
		path, err := strconv.Unquote(i.Path.Value)
		if err != nil {
			return err
		}
		// replace the value using the replace function
		path, err = replace(fset.Position(i.Pos()), path)
		if err != nil {
			if err == ErrSkip {
				continue
			}
			return err
		}
		i.Path.Value = strconv.Quote(path)
		change = true
	}
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "// import \"") {
				// trim off extra comment stuff
				ctext := c.Text
				ctext = strings.TrimPrefix(ctext, "// import")
				ctext = strings.TrimSpace(ctext)
				// unquote the comment import path value
				ctext, err := strconv.Unquote(ctext)
				if err != nil {
					return err
				}
				// match the comment import path with the given replacement map
				ctext, err = replace(fset.Position(c.Pos()), ctext)
				if err != nil {
					if err == ErrSkip {
						continue
					}
					return err
				}
				c.Text = "// import " + strconv.Quote(ctext)
				change = true
			}
		}
	}
	// if no change occured, then we don't need to write to disk, just return.
	if !change {
		return nil
	}
	// create a temporary file, this easily avoids conflicts.
	temp := name + ".temp"
	w, err := os.Create(temp)
	if err != nil {
		return err
	}
	defer w.Close()
	// preserve permissions
	info, err := os.Lstat(name)
	if err != nil {
		return err
	}
	if err := w.Chmod(info.Mode()); err != nil {
		return err
	}
	// write changes to .temp file, and include proper formatting.
	cfg := &printer.Config{
		Mode:     printer.TabIndent | printer.UseSpaces,
		Tabwidth: 8,
	}
	if err := cfg.Fprint(w, fset, f); err != nil {
		return err
	}
	// close the writer
	if err := w.Close(); err != nil {
		return err
	}
	// rename the .temp to .go
	return os.Rename(temp, name)
}
