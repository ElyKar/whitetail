// Whitetail is a lightweight and performant http router.
// It uses less memory than the well known httprouter,
// though it loses in terms of pure speed in general.
//
// Unlike the standard router from the http package, it
// supports three kinds of parameter: named, catchall and regex:
//  Syntax            Type
//  :name             named parameter
//  *name             catchall parameter
//  #name:^[a-z]+$    regexp parameter
//
// Named parameter are custom path segment. They match an entire element, excluding the trailing slash
//  Path: /api/user/:id/:post
//
//  Requests:
//  /api/user/12345/request            match: id=12345 ; post=request
//  /api/user/12345/request/           match: id=12345 ; post=request
//  /api/user/12345/                   no match
//  /api/user/12345                    no match
//  /api/user/12345/request/comments   no match
//
// Catchall parameters match the end of a URL, trailing slash included
//  Path: /api/*filename
//
//  Requests:
//  /api                 no match
//  /api/                match, filename=/
//  /api/a/file/         match, filename=/a/file/
//  /api/another/file    match, filename=/another/file
//
// Regexp parameters are composer of a name and a regular expression separed by a colon.
// In the path "/api/#user:^[a-z]+$/profile" the name is "user", the
// regex is "^[a-z]$".
// Behind the scened the regex is compiled using regexp.Compile.
// For retrieving a handler, the element of path is matched using regexp.MatchString
//  Path: /api/#user:^[a-z]$/profile
//
//  Requests:
//  /api/username/profile          match, user=username
//  /api/username/other/profile    no match
//
// A complete documentation of what is matched and why is present in the documentation of the regexp package.
//
// When an handler is retrieved, all of the matched elements are stored temporarily.
// They can be retrieved in the form of a map. Once fetched, the map is deleted to prevent memory leak.
//
//  parameters := whitetail.GetVars(req.URL.Path)
//
//  // The corresponding element in the URL
//  name := parameters["name"]
package whitetail

import "net/http"

// The map holding the parameters of paths
type parameters map[string]map[string]string

var params parameters = make(map[string]map[string]string)

// Get the parameters for a given path (deletes it from the router).
//
//  func handler(resp http.ReponseWriter, req *http.Request) {
//      params := whitetail.GetVars(req.UTL.Path)
//      ...
//  }
//
// Once the parameters are fetched, they are deleted, so it's
// not possible to get them anymore.
//
// If no map is available for the path, nil is returned instead
func GetVars(path string) map[string]string {
	defer DeleteQuietlyVars(path)

	path = clean(path)
	m, ok := params[path]
	if !ok {
		return nil
	}
	return m
}

// Deletes the parameters associated to a path. If for some
// reason the handler did not fetch them, you can delete them
// explicitely, you can delete them to prevent memory leaks.
func DeleteQuietlyVars(path string) {
	path = clean(path)
	if _, ok := params[path]; ok {
		delete(params, path)
	}
}

// Prepare a new map of parameters for a given path
// (path must be cleaned before calling this)
func prepare(path string) map[string]string {
	m := make(map[string]string)
	params[path] = m
	return m
}

// Router is the struct holding all routes and their associated handlers.
// It implements the http.Handler interface and can be used to dispatch incoming requests
type Router struct {
	// Roots of the trees. There is one root per method (GET/PUT/...)
	roots map[string]*node

	// The Panic handler is called whenever there is an internal error
	// It takes an interface{} in parameter, which is the error which
	// has been recovered.
	//
	// There is no default panic handler, thus it must be explicitely set when creating a Router.
	PanicHandler func(http.ResponseWriter, *http.Request, interface{})

	// This handler is called whenever one of the route cannot be found.
	// There is no default handler, thus it must be explicitely set when creating a Router.
	NotFoundHandler http.HandlerFunc
}

// Creates a new router.
func NewRouter() *Router {
	return &Router{roots: make(map[string]*node)}
}

// Handle registers a new request for the given path and method.
// It is intended for bulk-loading of routes, or for non-standard methods.
func (r *Router) Handle(method, path string, handle http.HandlerFunc) {
	root := r.roots[method]

	if root == nil {
		root = &node{}
		r.roots[method] = root
	}

	addChild(root, path, handle)
}

// Get is a shorthand for router.Handle("GET", path, handle)
func (r *Router) Get(path string, handle http.HandlerFunc) {
	r.Handle("GET", path, handle)
}

// Post is a shorthand for router.Handle("POST", path, handle)
func (r *Router) Post(path string, handle http.HandlerFunc) {
	r.Handle("POST", path, handle)
}

// Put is a shorthand for router.Handle("PUT", path, handle)
func (r *Router) Put(path string, handle http.HandlerFunc) {
	r.Handle("PUT", path, handle)
}

// Delete is a shorthand for router.Handle("DELETE", path, handle)
func (r *Router) Delete(path string, handle http.HandlerFunc) {
	r.Handle("DELETE", path, handle)
}

// Patch is a shorthand for router.Handle("PATCH", path, handle)
func (r *Router) Patch(path string, handle http.HandlerFunc) {
	r.Handle("PATCH", path, handle)
}

// Lookup is used to retrieve the handler of a given route.
// If the handler is not found, a nil handler is retrieved.
// Therefore, NotFoundHandler will never be returned
func (r *Router) Lookup(method, path string) http.HandlerFunc {
	root := r.roots[method]

	if root == nil {
		return nil
	}

	return root.lookup(path)
}

// ServeHTTP implements the http.Router interface.
// The workflow is as this:
//  - For the given route, try to find a handler
//  - If found, call it
//  - If not found, call the NotFoundHandler if it is set
//  - If the handler panic, recover the error and keep going
func (r *Router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer func() {
			if err := recover(); err != nil {
				r.PanicHandler(resp, req, err)
			}
		}()
	}
	path := req.URL.Path

	if root := r.roots[req.Method]; root != nil {
		handler := root.lookup(path)
		if handler != nil {
			handler(resp, req)
		} else if r.NotFoundHandler != nil {
			r.NotFoundHandler(resp, req)
		}
	}
}
