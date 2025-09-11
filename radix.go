package radix

import (
	"fmt"
	"strings"
	"sync/atomic"
)

type NodeType uint8

const (
	Static    NodeType = iota
	ParamNode          // :param
	Wildcard           // *wildcard
)

type Node struct {
	parent            *Node
	nodeSize          atomic.Uint32
	nodeType          NodeType
	path              string
	static_children   map[string]*Node
	params_children   map[string]*Node
	wildcard_children map[string]*Node
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
	return nw.node.nodeSize.Load()
}

func (nw *NodeWrapper) Equal(w *NodeWrapper) bool {
	return nw.node == w.node
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
	return r.root.nodeSize.Load()
}

func (r *RadixTree) Add(path []string, handler Handler) (*NodeWrapper, error) {
	return r.addRoute(r.root, path, handler)
}

func (r *RadixTree) Get(path []string) Routes {
	return r.getValue(r.root, path, nil)
}

func (r *RadixTree) Delete(path []string) error {
	// Deletion in a radix tree can be complex, especially with parameters and wildcards.
	// A simple approach is to traverse the tree to find the node and then remove the handler.
	// However, this does not handle pruning of empty nodes.
	return fmt.Errorf("delete operation is not implemented")
}

func (r *RadixTree) addRoute(node *Node, segments []string, handler Handler) (*NodeWrapper, error) {
	if len(segments) == 0 {
		if node.handler != nil {
			return nil, fmt.Errorf("handler already exists for this path")
		}
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
		node.nodeSize.Add(1)
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
	for _, child := range node.params_children {
		if child.paramName == segmentParam {
			return r.addRoute(child, remaining, handler)
		}
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
	if _, exists := node.params_children[child.paramName]; exists {
		return nil, fmt.Errorf("handler already exists for this path")
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
	}
	if node.wildcard_children == nil {
		node.wildcard_children = make(map[string]*Node)
	}
	if _, exists := node.wildcard_children[child.paramName]; exists {
		return nil, fmt.Errorf("handler already exists for this path")
	}
	node.wildcard_children[child.paramName] = child
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

	// Try static children first (highest priority)
	if node.static_children != nil {
		if child, exists := node.static_children[segment]; exists {
			if newRoutes := r.getValue(child, remaining, params); len(newRoutes) > 0 {
				routes = append(routes, newRoutes...)
			}
		}
	}

	// Try parameter children (medium priority)
	for _, child := range node.params_children {
		newParams := append(params, RouteParam{
			Key:    child.paramName,
			Values: segments[:1],
		})

		if newRoutes := r.getValue(child, remaining, newParams); len(newRoutes) > 0 {
			routes = append(routes, newRoutes...)
		}
	}

	if node.wildcard_children == nil {
		return routes
	}

	// Try wildcard child (lowest priority)
	// Wildcard consumes all remaining segments
	for _, child := range node.wildcard_children {
		if child.handler != nil {
			newParams := append(params, RouteParam{
				Key:    child.paramName,
				Values: segments,
			})
			routes = append(routes, Route{Handler: child.handler, Params: newParams})
		}
	}
	return routes
}
