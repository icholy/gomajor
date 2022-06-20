package packages

import (
	"fmt"

	"golang.org/x/mod/module"
)

// Index provides a quick way to lookup a module with a package
// import path.
type Index struct {
	root *indexNode
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
