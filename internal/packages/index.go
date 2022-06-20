package packages

import (
	"fmt"

	"golang.org/x/mod/module"
)

// Index provides a quick way to lookup a module with a package
// import path.
type Index struct {
	root *indexNode
	list []module.Version
}

type indexNode struct {
	children map[rune]*indexNode
	mod      *module.Version
}

func (i *Index) init() {
	if i.root == nil {
		i.root = &indexNode{
			children: map[rune]*indexNode{},
		}
	}
}

// Add a module to the index.
func (i *Index) Add(m module.Version) error {
	i.init()
	node := i.root
	for _, r := range m.Path {
		if _, ok := node.children[r]; !ok {
			node.children[r] = &indexNode{
				children: map[rune]*indexNode{},
			}
		}
		node = node.children[r]
	}
	if node.mod != nil {
		return fmt.Errorf("duplicate mod: %s", m)
	}
	node.mod = &m
	i.list = append(i.list, m)
	return nil
}

// Lookup the module for a package path.
func (i *Index) Lookup(pkgpath string) (module.Version, bool) {
	i.init()
	node := i.root
	var mod *module.Version
	for _, r := range pkgpath {
		var ok bool
		node, ok = node.children[r]
		if !ok {
			break
		}
		if node.mod != nil {
			mod = node.mod
		}
	}
	if mod == nil {
		return module.Version{}, false
	}
	return *mod, true
}

// Related returns all versions of the provided module.
func (i *Index) Related(modpath string) []module.Version {
	modprefix := ModPrefix(modpath)
	var mods []module.Version
	for _, mod := range i.list {
		modprefix0, _, ok := module.SplitPathVersion(mod.Path)
		if ok && modprefix == modprefix0 {
			mods = append(mods, mod)
		}
	}
	return mods
}

// LoadIndex reads the modfile into an Index.
func LoadIndex(dir string) (*Index, error) {
	file, err := loadModFile(dir)
	if err != nil {
		return nil, err
	}
	var idx Index
	for _, req := range file.Require {
		if !req.Indirect {
			if err := idx.Add(req.Mod); err != nil {
				return nil, err
			}
		}
	}
	return &idx, nil
}
