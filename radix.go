package radix

import (
	"fmt"
	"strings"
)

type NodeType uint8

const (
	Static    NodeType = iota
	ParamNode          // :param
	Wildcard           // *wildcard
)

type Node struct {
	parent            *Node
	nodeSize          uint32
	nodeType          NodeType
	path              string
	static_children   map[string]*Node
	params_children   map[string]*Node
	wildcard_children []*Node
	handler           Handler
	paramName         string
	isWildcard        bool
}

type Handler interface{}

type RouteParam struct {
	Key    string
	Values []string
}

type Params []RouteParam

type Route struct {
	Handler Handler
	Params  Params
}

type Routes []Route

type NodeWrapper struct {
	node *Node
}

type RadixTree struct {
	root *Node
}

func (ps Params) Get(name string) ([]string, bool) {
	for _, param := range ps {
		if param.Key == name {
			return param.Values, true
		}
	}
	return nil, false
}

func wrap(n *Node) *NodeWrapper {
	return &NodeWrapper{
		node: n,
	}
}

func (nw *NodeWrapper) PathName() string {
	return nw.node.path
}

func (nw *NodeWrapper) Parent() (*NodeWrapper, bool) {
	return wrap(nw.node.parent), nw.node.parent != nil
}

func (nw *NodeWrapper) Size() uint32 {
	return nw.node.nodeSize
}

func (nw *NodeWrapper) Equal(w *NodeWrapper) bool {
	return nw.node == w.node
}

func (nw *NodeWrapper) Path() []string {
	segments := []string{}
	current := nw.node
	for current != nil {
		segments = append([]string{current.path}, segments...)
		current = current.parent
	}
	return segments[1:]
}

func NewRadixTree() *RadixTree {
	return &RadixTree{
		root: &Node{
			parent: nil,
		},
	}
}

func (r *RadixTree) Root() *NodeWrapper {
	return wrap(r.root)
}

func (r *RadixTree) Size() uint32 {
	return r.root.nodeSize
}

func (r *RadixTree) Add(path []string, handler Handler) (*NodeWrapper, error) {
	return r.addRoute(r.root, path, handler)
}

func (r *RadixTree) Get(path []string) Routes {
	return r.getValue(r.root, path, nil)
}

func (r *RadixTree) Delete(path []string) error {
	return r.deleteRoute(r.root, path)
}

func (r *RadixTree) addRoute(node *Node, segments []string, handler Handler) (*NodeWrapper, error) {
	if len(segments) == 0 {
		if node.handler != nil {
			return nil, fmt.Errorf("handler already exists for this path")
		}
		node.nodeSize++
		node.handler = handler
		return wrap(node), nil
	}

	segment := segments[0]
	remaining := segments[1:]
	err := error(nil)
	var nw *NodeWrapper

	if strings.HasPrefix(segment, "*") {
		nw, err = r.addWildcardChild(node, segment, remaining, handler)
	} else if strings.HasPrefix(segment, ":") {
		nw, err = r.addParamChild(node, segment, remaining, handler)
	} else {
		nw, err = r.addStaticChild(node, segment, remaining, handler)
	}
	if err == nil {
		node.nodeSize++
	}
	return nw, err
}

func (r *RadixTree) addStaticChild(node *Node, segment string, remaining []string, handler Handler) (*NodeWrapper, error) {
	if child, exists := node.static_children[segment]; exists {
		return r.addRoute(child, remaining, handler)
	}

	child := &Node{
		nodeType: Static,
		path:     segment,
		parent:   node,
	}
	nw, err := r.addRoute(child, remaining, handler)
	if err != nil {
		return nil, err
	}

	if node.static_children == nil {
		node.static_children = make(map[string]*Node)
	}
	node.static_children[child.path] = child
	return nw, nil
}

func (r *RadixTree) addParamChild(node *Node, segment string, remaining []string, handler Handler) (*NodeWrapper, error) {
	segmentParam := segment[1:]

	if child, exists := node.params_children[segmentParam]; exists {
		return r.addRoute(child, remaining, handler)
	}
	child := &Node{
		nodeType:  ParamNode,
		path:      segment,
		paramName: segmentParam,
		parent:    node,
	}
	nw, err := r.addRoute(child, remaining, handler)
	if err != nil {
		return nil, err
	}

	if node.params_children == nil {
		node.params_children = make(map[string]*Node)
	}
	node.params_children[child.paramName] = child
	return nw, nil
}

func (r *RadixTree) addWildcardChild(node *Node, segment string, remaining []string, handler Handler) (*NodeWrapper, error) {
	if len(remaining) > 0 {
		return nil, fmt.Errorf("wildcard must be the last segment")
	}
	child := &Node{
		nodeType:   Wildcard,
		path:       segment,
		paramName:  segment[1:],
		isWildcard: true,
		handler:    handler,
		parent:     node,
		nodeSize:   1,
	}
	node.wildcard_children = append(node.wildcard_children, child)
	return wrap(child), nil
}

func (r *RadixTree) getValue(node *Node, segments []string, params Params) Routes {
	if len(segments) == 0 {
		if node.handler != nil {
			return Routes{{Handler: node.handler, Params: params}}
		}
		return Routes{}
	}

	segment := segments[0]
	remaining := segments[1:]

	routes := Routes{}

	// Snapshot child pointers while holding the read lock to avoid
	// iterating maps/slices that may be mutated by writers.
	var staticChild *Node
	if node.static_children != nil {
		staticChild = node.static_children[segment]
	}

	var paramChildren []*Node
	if len(node.params_children) > 0 {
		paramChildren = make([]*Node, 0, len(node.params_children))
		for _, child := range node.params_children {
			paramChildren = append(paramChildren, child)
		}
	}

	var wildcardChildren []*Node
	if len(node.wildcard_children) > 0 {
		wildcardChildren = make([]*Node, len(node.wildcard_children))
		copy(wildcardChildren, node.wildcard_children)
	}

	// Try static children first (highest priority)
	if staticChild != nil {
		if newRoutes := r.getValue(staticChild, remaining, params); len(newRoutes) > 0 {
			routes = append(routes, newRoutes...)
		}
	}

	// Try parameter children (medium priority)
	if len(paramChildren) > 0 {
		paramsRoutes := segments[:1]
		for _, child := range paramChildren {
			newParams := append(params, RouteParam{
				Key:    child.paramName,
				Values: paramsRoutes,
			})
			if newRoutes := r.getValue(child, remaining, newParams); len(newRoutes) > 0 {
				routes = append(routes, newRoutes...)
			}
		}
	}

	// Try wildcard child (lowest priority)
	if len(wildcardChildren) > 0 {
		for _, child := range wildcardChildren {
			if child.handler != nil {
				newParams := append(params, RouteParam{
					Key:    child.paramName,
					Values: segments,
				})
				routes = append(routes, Route{Handler: child.handler, Params: newParams})
			}
		}
	}

	return routes
}

func (r *RadixTree) deleteRoute(node *Node, path []string) error {
	if len(path) == 0 {
		if node.handler != nil {
			node.handler = nil
			node.nodeSize--
			return nil
		}
		return fmt.Errorf("path cannot be empty")
	}
	segment := path[0]
	remaining := path[1:]

	var child *Node
	if strings.HasPrefix(segment, "*") {
		for _, wc := range node.wildcard_children {
			if wc.path == segment {
				child = wc
				break
			}
		}
	} else if strings.HasPrefix(segment, ":") {
		if node.params_children != nil {
			child = node.params_children[segment[1:]]
		}
	} else {
		if node.static_children != nil {
			child = node.static_children[segment]
		}
	}

	if child == nil {
		return fmt.Errorf("path not found")
	}

	err := r.deleteRoute(child, remaining)
	if err != nil {
		return err
	}

	if child.nodeSize == 0 {
		switch child.nodeType {
		case Static:
			delete(node.static_children, child.path)
			if len(node.static_children) == 0 {
				node.static_children = nil
			}
		case ParamNode:
			delete(node.params_children, child.paramName)
			if len(node.params_children) == 0 {
				node.params_children = nil
			}
		case Wildcard:
			for i, wc := range node.wildcard_children {
				if wc == child {
					node.wildcard_children = append(node.wildcard_children[:i], node.wildcard_children[i+1:]...)
					break
				}
			}
		}
	}

	node.nodeSize--
	return nil
}
