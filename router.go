package main

import (
	"fmt"
	"net/http"
	"strings"
)

type method uint8

const (
	GET = iota
	POST
	PUT
	PATCH
	HEAD
	DELETE
	nb
)

type Parameters map[string]string

type Handle func(resp http.ResponseWriter, req *http.Request, params Parameters)

var _ http.Handler = NewRouter()

func mthd(method string) method {
	switch method {
	case "GET":
		return GET
	case "POST":
		return POST
	case "PUT":
		return PUT
	case "PATCH":
		return PATCH
	case "HEAD":
		return HEAD
	case "DELETE":
		return DELETE
	default:
		panic(fmt.Sprintf("Unsupported method %s", method))
	}
}

type Router struct {
	roots []*node

	RedirectNotFound Handle

	PanicHandler Handle
}

func NewRouter() *Router {
	router := &Router{roots: make([]*node, nb, nb)}
	for i, _ := range router.roots {
		router.roots[i] = &node{make([]*node, 0), kind(1), "", nil}
	}
	return router
}

func (r *Router) Handle(method, path string, handle Handle) {
	cleaned := clean(path)
	addChild(r.roots[mthd(method)], cleaned, handle)
}

func (r *Router) Lookup(method, path string) (Handle, Parameters) {
	return r.roots[mthd(method)].lookup(path)
}

func (r *Router) Handler(method, path string, handler http.Handler) {
	r.Handle(method, path, func(resp http.ResponseWriter, req *http.Request, _ Parameters) {
		handler.ServeHTTP(resp, req)
	})
}

func (r *Router) HandlerFunc(method, path string, handler http.HandlerFunc) {
	r.Handle(method, path, func(resp http.ResponseWriter, req *http.Request, _ Parameters) {
		handler(resp, req)
	})
}

func (r *Router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			r.PanicHandler(resp, req, nil)
		}
	}()

	handler, params := r.Lookup(req.Method, req.URL.Path)
	if handler == nil {
		fmt.Println("Handler was not found")
		handler = r.RedirectNotFound
	}
	handler(resp, req, params)
}

func (r *Router) Debug() string {
	str := ""
	for _, n := range r.roots {
		str += strings.Join(n.String("", make([]string, 0)), "\n")
	}
	return str
}
