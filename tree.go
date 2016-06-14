package whitetail

import (
	"fmt"
	"net/http"
	"regexp"
)

type kind int8

const (
	// Normal node
	normal = 0
	// Named node
	named = 58
	// Catchall node
	catchall = 42
	// Regexp node
	re = 35
)

type node struct {
	// The array of childs
	children []*node
	// The type of nodes
	kind kind
	// Name of the parameter
	name string
	// Storage of the regexp
	reg *regexp.Regexp
	// handle of the node (if any)
	handle http.HandlerFunc
}

func addChild(root *node, path string, handle http.HandlerFunc) {
	if handle == nil {
		panic(fmt.Sprintf("Must provide a handler for path %s", path))
	}
	// Clean the path beforehand
	path = clean(path)

	// Do some assignations to prepare what's next
	current := root
	var i, j, r int
	var k kind
	var name, data string
	var next *node
	var regex *regexp.Regexp
	var err error
	n := len(path)

	// This iteration is a bit tricky because usually one would iterate through the entire string. Here we iterate until we reached the character before the last one.
	// In fact, the last character can be a trailing slash, and we do not want to count it as a new node.
	// If it is not a trailing slash, then it's a letter preceded by a slash (eg '/a') so it enters the loop nonetheless
	for i < n-1 {

		// Get the next segment of path, it starts with a slash
		j++
		for j < n && path[j] != '/' {
			j++
		}
		name = path[i:j]
		i = j

		// Use the appropriate kind of node for further treatment
		if name[1] == named || name[1] == catchall || name[1] == re {
			k = kind(name[1])
			name = "/" + name[2:]
		} else {
			k = normal
		}

		// If this is not a normal node, there are several checks that need to be performed
		if k != normal {
			// In case there is something like '/:' in the path
			if len(name) == 1 {
				panic(fmt.Sprintf("Error on path %s: a special parameter must be named (%s)", path, name))
			}

			// Catchall are allowed only for the last node
			if k == catchall && j < n {
				panic(fmt.Sprintf("Error on path %s: a catchall parameter can only be at the end", path))
			}

			// Process the regular expression. The name will contain the id, and the data will be the actual regex
			if k == re {
				r = 1
				for r < len(name) && name[r] != ':' && name[r] != '/' {
					r++
				}
				data = name
				if r < len(name) {
					name = name[:r]
				}
			}
		}

		// Check if this node already exist
		next = nil
		for _, child := range current.children {
			if k != child.kind {
				panic(fmt.Sprintf("Error on path %s: a special parameter cannot have brothers", path))
			}

			if name == child.name {
				next = child
				break
			}
		}

		// The child does not exist yet
		if next == nil {
			if k == re {
				if len(data) < len(name)+2 {
					panic(fmt.Sprintf("Error on path %s: you must provide a regular expression for parameter %s", path, data))
				}
				regex, err = regexp.Compile(data[r+1:])
				if err != nil {
					panic(fmt.Sprintf("Error on path %s: the regular expression provided is not correct(%s): [%s]", path, data, err.Error()))
				}
			} else {
				regex = nil
			}

			next = &node{
				kind: k,
				name: name,
				reg:  regex,
			}

			current.children = append(current.children, next)
		}

		// Move on to the next node
		current = next
	}

	// Now, assert we can add the handle to this node
	if current.handle != nil {
		panic(fmt.Sprintf("There is already a handle for path %s", path))
	}

	current.handle = handle
	return

}

func (root *node) lookup(path string) http.HandlerFunc {

	var m map[string]string

	path = clean(path)
	current := root
	n := len(path)
	var next *node
	var i, j int
	var name string

	for i < n-1 {

		// Get the next segment of path, it starts with a slash
		j++
		for j < n && path[j] != '/' {
			j++
		}
		name = path[i:j]
		i = j

		for _, child := range current.children {
			if child.kind != normal {
				next = child
				break
			}

			if child.name == name {
				next = child
				break
			}
		}

		if next == current {
			if m != nil {
				DeleteQuietlyVars(path)
			}
			return nil
		}

		switch next.kind {
		case catchall:
			m = lazyParams(path, next.name[1:], path[j-len(name):], m)
			return next.handle

			// Store the named node
		case named:
			m = lazyParams(path, next.name[1:], name[1:], m)
		case re:
			if next.reg.MatchString(name[1:]) {
				m = lazyParams(path, next.name[1:], name[1:], m)
			} else {
				if m != nil {
					DeleteQuietlyVars(path)
				}
				return nil
			}
		}

		current = next

	}

	// Workaround to handle the case where there is a catchall parameter catching only a slash
	if path[n-1] == '/' && len(current.children) == 1 && current.children[0].kind == catchall {
		_ = lazyParams(path, current.children[0].name[1:], "/", m)
		return current.children[0].handle
	}

	return current.handle
}

// Lazy instantiation of the map of parameters for a route
func lazyParams(path, key, value string, m map[string]string) map[string]string {
	if m == nil {
		m = prepare(path)
	}
	m[key] = value
	return m
}
