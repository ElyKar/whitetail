package whitetail

import "net/http"

type parameters map[string]map[string]string

var params parameters = make(map[string]map[string]string)

func GetVars(path string) map[string]string {
	defer DeleteQuietlyVars(path)

	path = clean(path)
	m, ok := params[path]
	if !ok {
		return nil
	}
	return m
}

func DeleteQuietlyVars(path string) {
	path = clean(path)
	if _, ok := params[path]; ok {
		delete(params, path)
	}
}

func prepare(path string) map[string]string {
	m := make(map[string]string)
	params[path] = m
	return m
}

type Router struct {
	roots map[string]*node

	PanicHandler func(http.ResponseWriter, *http.Request, interface{})

	NotFoundHandler http.HandlerFunc
}

func NewRouter() *Router {
	return &Router{roots: make(map[string]*node)}
}

func (r *Router) Handle(method, path string, handle http.HandlerFunc) {
	root := r.roots[method]

	if root == nil {
		root = &node{}
		r.roots[method] = root
	}

	addChild(root, path, handle)
}

func (r *Router) Get(path string, handle http.HandlerFunc) {
	r.Handle("GET", path, handle)
}

func (r *Router) Post(path string, handle http.HandlerFunc) {
	r.Handle("POST", path, handle)
}

func (r *Router) Put(path string, handle http.HandlerFunc) {
	r.Handle("PUT", path, handle)
}

func (r *Router) Delete(path string, handle http.HandlerFunc) {
	r.Handle("DELETE", path, handle)
}

func (r *Router) Patch(path string, handle http.HandlerFunc) {
	r.Handle("PATCH", path, handle)
}

func (r *Router) Lookup(method, path string) http.HandlerFunc {
	root := r.roots[method]

	if root == nil {
		return nil
	}

	return root.lookup(path)
}

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
