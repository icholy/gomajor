package fixdocs

import (
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/icholy/gomajor/internal/packages"
)

// RewriteModPath recursively searches for files with the provided extensions and
// rewrites any containing module paths.
func RewriteModPath(root string, extensions []string, modprefix, version string) error {
	modpath := packages.JoinPath(modprefix, version, "")
	re, err := regexp.Compile(modprefix + `(/v\d+)?`)
	if err != nil {
		return err
	}
	files, err := FindFiles(root, extensions)
	if err != nil {
		return err
	}
	for _, name := range files {
		filename := filepath.Join(root, name)
		if err := RewriteFile(filename, re, modpath); err != nil {
			return err
		}
	}
	return nil
}

// FindFiles recursively searches for files ending with the specified extensions.
// The extenions matching is case insensitive.
func FindFiles(root string, extensions []string) ([]string, error) {
	extset := map[string]struct{}{}
	for _, ext := range extensions {
		ext = strings.ToLower(ext)
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extset[ext] = struct{}{}
	}
	var files []string
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Println("doc rewrite:", err)
			return nil
		}
		if info.IsDir() {
			// don't recurse into vendor or .git directories
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		name := info.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if _, ok := extset[ext]; ok {
			files = append(files, name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// RewriteFile rewrites the file specified by filename by replacing all instances of
// search with replace.
func RewriteFile(filename string, search *regexp.Regexp, replace string) error {
	f, err := os.OpenFile(filename, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}
	data = search.ReplaceAll(data, []byte(replace))
	if _, err := f.Write(data); err != nil {
		return err
	}
	return f.Close()
}
