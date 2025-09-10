package radix

import (
	"fmt"
	"strings"
)

// NodeType represents the type of a radix tree node
type NodeType uint8

const (
	Static    NodeType = iota
	ParamNode          // :param
	Wildcard           // *wildcard
)

// Node represents a node in the radix tree
type Node struct {
	nodeType        NodeType
	path            string
	static_children map[string]*Node
	params_children []*Node
	wildcard_child  *Node
	handler         Handler
	paramName       string
	isWildcard      bool
}

// Handler represents a route handler
type Handler interface{}

// RouteParam represents a URL parameter
type RouteParam struct {
	Key    string
	Values []string
}

// Params is a slice of parameters
type Params []RouteParam

// Get returns the value of the first parameter with the given name
func (ps Params) Get(name string) ([]string, bool) {
	for _, param := range ps {
		if param.Key == name {
			return param.Values, true
		}
	}
	return nil, false
}

// RadixTree represents a radix tree for routing
type RadixTree struct {
	root *Node
}

// NewRadixTree creates a new radix tree
func NewRadixTree() *RadixTree {
	return &RadixTree{
		root: &Node{},
	}
}

// Add adds a route to the radix tree
func (r *RadixTree) Add(path []string, handler Handler) error {
	return r.addRoute(r.root, path, handler)
}

// Get searches for a route in the radix tree
func (r *RadixTree) Get(path []string) (Handler, Params, bool) {
	return r.getValue(r.root, path, nil)
}

// addRoute adds a route to the tree
func (r *RadixTree) addRoute(node *Node, segments []string, handler Handler) error {
	if len(segments) == 0 {
		if node.handler != nil {
			return fmt.Errorf("handler already exists for this path")
		}
		node.handler = handler
		return nil
	}

	segment := segments[0]
	remaining := segments[1:]

	if strings.HasPrefix(segment, "*") {
		if len(remaining) > 0 {
			return fmt.Errorf("wildcard must be the last segment")
		}
		if node.wildcard_child != nil {
			return fmt.Errorf("wildcard child already exists")
		}
		child := &Node{
			nodeType:   Wildcard,
			path:       segment,
			paramName:  segment[1:],
			isWildcard: true,
		}
		node.wildcard_child = child
		return r.addRoute(child, remaining, handler)
	}

	if strings.HasPrefix(segment, ":") {
		segmentParam := segment[1:]
		if segmentParam == "" {
			return fmt.Errorf("parameter name cannot be empty")
		}
		for _, child := range node.params_children {
			if child.paramName == segmentParam {
				return r.addRoute(child, remaining, handler)
			}
		}
		child := &Node{
			nodeType:  ParamNode,
			path:      segment,
			paramName: segmentParam,
		}
		node.params_children = append(node.params_children, child)
		return r.addRoute(child, remaining, handler)
	}

	for _, child := range node.static_children {
		if child.path == segment {
			return r.addRoute(child, remaining, handler)
		}
	}

	// Create new child
	child := &Node{
		nodeType: Static,
		path:     segment,
	}
	if node.static_children == nil {
		node.static_children = make(map[string]*Node)
	}
	node.static_children[child.path] = child
	return r.addRoute(child, remaining, handler)
}

// getValue searches for a route and extracts parameters
func (r *RadixTree) getValue(node *Node, segments []string, params Params) (Handler, Params, bool) {
	if len(segments) == 0 {
		return node.handler, params, node.handler != nil
	}

	segment := segments[0]
	remaining := segments[1:]

	// Try static children first (highest priority)
	if node.static_children != nil {
		if child, exists := node.static_children[segment]; exists {
			if handler, newParams, found := r.getValue(child, remaining, params); found {
				return handler, newParams, true
			}
		}
	}

	// Try parameter children (medium priority)
	for _, child := range node.params_children {
		newParams := append(params, RouteParam{
			Key:    child.paramName,
			Values: segments[:1],
		})

		if handler, finalParams, found := r.getValue(child, remaining, newParams); found {
			return handler, finalParams, true
		}
	}

	if node.wildcard_child == nil {
		return nil, nil, false
	}

	// Try wildcard child (lowest priority)
	// Wildcard consumes all remaining segments
	newParams := params
	newParams = append(params, RouteParam{
		Key:    node.wildcard_child.paramName,
		Values: segments,
	})
	return node.wildcard_child.handler, newParams, node.wildcard_child.handler != nil
}
