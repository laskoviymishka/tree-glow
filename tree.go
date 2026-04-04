package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// node represents a file or directory in the tree.
type node struct {
	name     string
	path     string
	isDir    bool
	children []*node
	expanded bool
	depth    int
	parent   *node
}

func (n *node) loadChildrenFiltered(showHidden bool) error {
	if !n.isDir {
		return nil
	}
	entries, err := os.ReadDir(n.path)
	if err != nil {
		return err
	}

	n.children = nil
	for _, e := range entries {
		if !showHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		childPath := filepath.Join(n.path, e.Name())
		isDir := e.IsDir()
		// Follow symlinks to detect directories
		if e.Type()&os.ModeSymlink != 0 {
			if info, err := os.Stat(childPath); err == nil {
				isDir = info.IsDir()
			}
		}
		child := &node{
			name:   e.Name(),
			path:   childPath,
			isDir:  isDir,
			depth:  n.depth + 1,
			parent: n,
		}
		n.children = append(n.children, child)
	}

	sort.Slice(n.children, func(i, j int) bool {
		if n.children[i].isDir != n.children[j].isDir {
			return n.children[i].isDir
		}
		return strings.ToLower(n.children[i].name) < strings.ToLower(n.children[j].name)
	})
	return nil
}

// flatten returns a visible list of nodes for rendering.
func flatten(n *node) []*node {
	var result []*node
	flattenInto(n, &result)
	return result
}

func flattenInto(n *node, out *[]*node) {
	for _, child := range n.children {
		*out = append(*out, child)
		if child.isDir && child.expanded && child.depth < 64 {
			flattenInto(child, out)
		}
	}
}

// toggle expands/collapses a directory node.
func (n *node) toggle(showHidden bool) {
	if !n.isDir {
		return
	}
	if n.expanded {
		n.expanded = false
		n.children = nil
	} else {
		n.expanded = true
		n.loadChildrenFiltered(showHidden)
	}
}

// icon returns a simple icon for the node.
func (n *node) icon() string {
	if n.isDir {
		if n.expanded {
			return " "
		}
		return " "
	}
	return fileIcon(n.name)
}

// fileIcon picks an icon based on extension.
func fileIcon(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return " "
	case ".md", ".markdown":
		return " "
	case ".json":
		return " "
	case ".yaml", ".yml":
		return " "
	case ".toml":
		return " "
	case ".sh", ".bash", ".zsh":
		return " "
	case ".py":
		return " "
	case ".js", ".ts", ".tsx", ".jsx":
		return " "
	case ".rs":
		return " "
	case ".html", ".css":
		return " "
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		return " "
	case ".lock":
		return " "
	default:
		return " "
	}
}

// newRootNode creates the root node for a given path.
func newRootNode(path string, showHidden bool) *node {
	info, err := os.Stat(path)
	isDir := err == nil && info.IsDir()
	root := &node{
		name:     filepath.Base(path),
		path:     path,
		isDir:    isDir,
		expanded: isDir,
		depth:    0,
	}
	if isDir {
		root.loadChildrenFiltered(showHidden)
	}
	return root
}
