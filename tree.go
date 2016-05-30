package main

import (
	"fmt"
	"strings"
)

type kind int8

const (
	normal   = 1
	named    = 58
	wildcard = 42
)

type node struct {
	children []*node
	kind     kind
	name     string
	handle   Handle
}

func addChild(root *node, path string, handle Handle) {
	nodes := strings.Split(path, "/")[1:]
	current := root
	var k kind
	for i, n := range nodes {

		if n[0] == named || n[0] == wildcard {
			k = kind(n[0])
			n = n[1:]
			if len(n) == 0 {
				panic(fmt.Sprintf("Named and wildcards parameters cannot be empty on path %s", path))
			}
		} else {
			k = 1
		}

		if k == wildcard && i != len(nodes)-1 {
			panic(fmt.Sprintf("Wildcard parameter must be at the end in %s", path))
		}

		var next *node = nil
		for _, node := range current.children {
			if node.name == n {
				next = node
				break
			}
		}

		if next == nil {
			next = &node{
				children: make([]*node, 0),
				kind:     k,
				name:     n,
				handle:   nil,
			}
			current.children = append(current.children, next)
		} else if next.kind != k {
			panic(fmt.Sprintf("Cannot insert a parameter which has the same name as %s for path %s", n, path))
		} else if k == named && n != current.name {
			panic(fmt.Sprintf("Named parameters mismatch between %s and %s", n, current.name))
		}

		if next.kind == wildcard && len(current.children) != 1 {
			panic(fmt.Sprintf("The wildcard in %s is masking other routes", path))
		}
		if next.kind == named && len(current.children) != 1 {
			panic(fmt.Sprintf("The named parameter in %s is masking other routes", path))
		}

		if i == len(nodes)-1 {
			if next.handle != nil {
				panic(fmt.Sprintf("There is already a handle for path %s", path))
			} else {
				next.handle = handle
			}
		}

		current = next

	}
}

func (r *node) lookup(path string) (Handle, Parameters) {
	current := r
	nodes := strings.Split(clean(path), "/")[1:]
	next := current
	params := make(map[string]string)
	for i, node := range nodes {
		for _, child := range current.children {
			if child.kind == wildcard && child.handle != nil {
				params[child.name] = "/" + strings.Join(nodes[i:], "/")
				return child.handle, params
			} else if child.kind == named {
				params[child.name] = node
				next = child
				break
			} else if child.name == node {
				next = child
				break
			}
		}
		// We didn't find anything
		if next == current {
			return nil, nil
		}
		current = next

	}
	if current.handle != nil {
		return current.handle, params
	}

	// UGLY HACK FOR '*' param with empty
	if len(current.children) == 1 && path[len(path)-1] == '/' {
		for _, h := range current.children {
			if h.kind == wildcard {
				params[h.name] = "/"
				return h.handle, params
			}
		}
	}

	return nil, nil
}

func (n *node) String(path string, res []string) []string {
	if path != "" {
		path = path + "/"
	}
	if n.kind == named {
		path += ":"
	} else if n.kind == wildcard {
		path += "*"
	}
	path += n.name
	for _, v := range n.children {
		res = v.String(path, res)
	}
	if n.handle != nil {
		res = append(res, path)
	}
	return res
}
