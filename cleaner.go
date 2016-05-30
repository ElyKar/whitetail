package main

import (
	"fmt"
	"strings"
)

func clean(path string) string {
	elements := strings.Split(path, "/")
	fmt.Printf("Path is %s\n", path)
	fmt.Printf("Split: %v\n", elements)
	var l int
	res := make([]string, 0)
	for _, elt := range elements {
		if elt == ".." {
			if len(res) < 1 {
				l = 0
			} else {
				l = len(res) - 1
			}
			res = res[:l]
		} else if elt == "." || strings.TrimLeft(elt, ".") == "" {
			continue
		} else {
			res = append(res, elt)
		}
	}
	return "/" + strings.Join(res, "/")
}
