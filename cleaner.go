package whitetail

// Cleans the path given.
// Rules for cleaning:
//
// - All duplicate slashes are removed ( '//' -> '/')
// - Possibility to navigate the path with '..' and '.' (cannot go beyond '/')
// - If there is no leading slash, add it
//
// If no cleaning is necessary (as it is likely), just returns the given path.
func clean(path string) string {

	if path == "" {
		return "/"
	}

	// Do nothing if no cleaning needed
	if is_clean(path) {
		return path
	}

	n := len(path)
	var buf []byte

	r := 1
	w := 1

	if path[0] != '/' {
		buf = make([]byte, n+1)
		r = 0
	} else {
		buf = make([]byte, n)
	}
	buf[0] = '/'

	trailing := n > 2 && path[n-1] == '/'

	for r < n {

		switch {

		// Remove duplicate slashes: do nothing
		case path[r] == '/':
			r++

		// The path ends with '/.'
		case path[r] == '.' && r == n-1:
			trailing = true
			r++

		// The path ends with '/.'
		case path[r] == '.' && path[r+1] == '/':
			r++

		// Handle the .. element
		case path[r] == '.' && path[r+1] == '.' && (r+1 == n-1 || path[r+2] == '/'):
			r += 2
			if w != 1 {
				w--
				for w > 0 && buf[w-1] != '/' {
					w--
				}
			}

			// Path ends with ".."
			if r == n {
				trailing = true
			}

		default:

			for r < n && path[r] != '/' {
				buf[w] = path[r]
				w++
				r++
			}

			if r < n || trailing {
				buf[w] = '/'
				w++
			}

		}

	}

	return string(buf[:w])
}

// Assert the given path does not need cleaning
func is_clean(path string) bool {

	if path[0] != '/' {
		return false
	}

	n := len(path)
	var cur int

	for cur < n {

		if path[cur] != '.' {
			cur++
		} else {
			if path[cur-1] == '/' && (cur == n-1 || path[cur+1] == '/' || (path[cur+1] == '.' && (cur+2 == n-1 || path[cur+2] == '/'))) {
				return false
			} else {
				cur += 2
			}
		}
	}

	return true
}
