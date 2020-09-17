package importpaths

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ReplaceFunc is called with every import path and returns the replacement path
// if the second return parameter is false, the replacement doesn't happen
type ReplaceFunc func(name, path string) (string, bool)

// Rewrite takes a directory path and a function for replacing imports paths
func Rewrite(dir string, replace ReplaceFunc) error {
	return filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
		// check errors
		if err != nil {
			log.Println("import rewrite:", err)
			return nil
		}
		// check the file is a .go file.
		if info.IsDir() || !strings.HasSuffix(name, ".go") {
			return nil
		}
		return RewriteFile(name, replace)
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
		if path, ok := replace(name, path); ok {
			i.Path.Value = strconv.Quote(path)
			change = true
		}
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
				if ctext, ok := replace(name, ctext); ok {
					c.Text = "// import " + strconv.Quote(ctext)
					change = true
				}
			}
		}
	}

	// if no change occured, then we don't need to write to disk, just return.
	if !change {
		return nil
	}

	// since the imports changed, resort them.
	ast.SortImports(fset, f)

	// create a temporary file, this easily avoids conflicts.
	temp := name + ".temp"
	w, err := os.Create(temp)
	if err != nil {
		return err
	}
	defer w.Close()

	// write changes to .temp file, and include proper formatting.
	err = (&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(w, fset, f)
	if err != nil {
		return err
	}

	// close the writer
	err = w.Close()
	if err != nil {
		return err
	}

	// rename the .temp to .go
	return os.Rename(temp, name)
}
