package main

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

// Test helper functions
func assertEqual(t *testing.T, actual, expected interface{}, message string) {
	if actual != expected {
		t.Errorf("%s: expected %v, got %v", message, expected, actual)
	}
}

func assertParamsEqual(t *testing.T, actual, expected Params, message string) {
	if len(actual) != len(expected) {
		t.Errorf("%s: expected %d params, got %d", message, len(expected), len(actual))
		return
	}

	for i, param := range expected {
		if i >= len(actual) || actual[i].Key != param.Key || !reflect.DeepEqual(actual[i].Values, param.Values) {
			t.Errorf("%s: expected param %d to be {%s: %v}, got {%s: %v}",
				message, i, param.Key, param.Values, actual[i].Key, actual[i].Values)
		}
	}
}

// TestBasicRouting tests basic static route matching
func TestBasicRouting(t *testing.T) {
	tree := NewRadixTree()

	// Add routes
	tree.Add([]string{}, "root")
	tree.Add([]string{"users"}, "users")
	tree.Add([]string{"admin"}, "admin")
	tree.Add([]string{"api", "v1"}, "api_v1")

	tests := []struct {
		path     []string
		expected string
		found    bool
	}{
		{[]string{}, "root", true},
		{[]string{"users"}, "users", true},
		{[]string{"admin"}, "admin", true},
		{[]string{"api", "v1"}, "api_v1", true},
		{[]string{"nonexistent"}, "", false},
		{[]string{"user"}, "", false},
		{[]string{"api"}, "", false},
	}

	for _, test := range tests {
		handler, _, found := tree.Get(test.path)
		assertEqual(t, found, test.found, fmt.Sprintf("Route %v found status", test.path))
		if found {
			assertEqual(t, handler.(string), test.expected, fmt.Sprintf("Route %v handler", test.path))
		}
	}
}

// TestParameterRouting tests dynamic parameter matching
func TestParameterRouting(t *testing.T) {
	tree := NewRadixTree()

	// Add routes with parameters
	tree.Add([]string{"users", ":id"}, "user_show")
	tree.Add([]string{"users", ":id", "posts"}, "user_posts")
	tree.Add([]string{"users", ":id", "posts", ":post_id"}, "user_post_show")
	tree.Add([]string{"articles", ":slug", "comments", ":comment_id"}, "article_comment")

	tests := []struct {
		path            []string
		expectedHandler string
		expectedParams  Params
		found           bool
	}{
		{
			[]string{"users", "123"},
			"user_show",
			Params{{Key: "id", Values: []string{"123"}}},
			true,
		},
		{
			[]string{"users", "456", "posts"},
			"user_posts",
			Params{{Key: "id", Values: []string{"456"}}},
			true,
		},
		{
			[]string{"users", "789", "posts", "101"},
			"user_post_show",
			Params{{Key: "id", Values: []string{"789"}}, {Key: "post_id", Values: []string{"101"}}},
			true,
		},
		{
			[]string{"articles", "golang-tips", "comments", "5"},
			"article_comment",
			Params{{Key: "slug", Values: []string{"golang-tips"}}, {Key: "comment_id", Values: []string{"5"}}},
			true,
		},
		{
			[]string{"users"},
			"",
			nil,
			false,
		},
		{
			[]string{"users", "123", "posts", "456", "extra"},
			"",
			nil,
			false,
		},
	}

	for _, test := range tests {
		handler, params, found := tree.Get(test.path)
		assertEqual(t, found, test.found, fmt.Sprintf("Route %v found status", test.path))
		if found {
			assertEqual(t, handler.(string), test.expectedHandler, fmt.Sprintf("Route %v handler", test.path))
			assertParamsEqual(t, params, test.expectedParams, fmt.Sprintf("Route %v params", test.path))
		}
	}
}

// TestWildcardRouting tests wildcard (*) parameter matching
func TestWildcardRouting(t *testing.T) {
	tree := NewRadixTree()

	// Add wildcard routes
	tree.Add([]string{"files", "*filepath"}, "files")
	tree.Add([]string{"admin", "*path"}, "admin_catch_all")
	tree.Add([]string{"static", "*filename"}, "static_files")

	tests := []struct {
		path            []string
		expectedHandler string
		expectedParams  Params
		found           bool
	}{
		{
			[]string{"files", "documents", "readme.txt"},
			"files",
			Params{{Key: "filepath", Values: []string{"documents", "readme.txt"}}},
			true,
		},
		{
			[]string{"admin", "dashboard"},
			"admin_catch_all",
			Params{{Key: "path", Values: []string{"dashboard"}}},
			true,
		},
		{
			[]string{"admin", "users", "settings", "advanced"},
			"admin_catch_all",
			Params{{Key: "path", Values: []string{"users", "settings", "advanced"}}},
			true,
		},
		{
			[]string{"static", "css", "style.css"},
			"static_files",
			Params{{Key: "filename", Values: []string{"css", "style.css"}}},
			true,
		},
		{
			[]string{"files"},
			"",
			nil,
			false,
		},
		{
			[]string{"unknown", "path"},
			"",
			nil,
			false,
		},
	}

	for _, test := range tests {
		handler, params, found := tree.Get(test.path)
		assertEqual(t, found, test.found, fmt.Sprintf("Route %v found status", test.path))
		if found {
			assertEqual(t, handler.(string), test.expectedHandler, fmt.Sprintf("Route %v handler", test.path))
			assertParamsEqual(t, params, test.expectedParams, fmt.Sprintf("Route %v params", test.path))
		}
	}
}

// TestMixedRouting tests combination of static, parameter, and wildcard routes
func TestMixedRouting(t *testing.T) {
	tree := NewRadixTree()

	// Add mixed routes
	tree.Add([]string{}, "root")
	tree.Add([]string{"api"}, "api_root")
	tree.Add([]string{"api", "users"}, "api_users")
	tree.Add([]string{"api", "users", ":id"}, "api_user_show")
	tree.Add([]string{"api", "users", ":id", "profile"}, "api_user_profile")
	tree.Add([]string{"api", "posts", ":post_id", "comments", ":comment_id"}, "api_comment")
	tree.Add([]string{"files", "*filepath"}, "serve_files")
	tree.Add([]string{"admin", "*path"}, "admin_panel")
	tree.Add([]string{"files", "~", "", ":filename"}, "static_filename_tilde")

	tests := []struct {
		path            []string
		expectedHandler string
		expectedParams  Params
		found           bool
	}{
		{[]string{}, "root", Params{}, true},
		{[]string{"api"}, "api_root", Params{}, true},
		{[]string{"api", "users"}, "api_users", Params{}, true},
		{[]string{"api", "users", "123"}, "api_user_show", Params{{Key: "id", Values: []string{"123"}}}, true},
		{[]string{"api", "users", "456", "profile"}, "api_user_profile", Params{{Key: "id", Values: []string{"456"}}}, true},
		{[]string{"api", "posts", "789", "comments", "101"}, "api_comment", Params{{Key: "post_id", Values: []string{"789"}}, {Key: "comment_id", Values: []string{"101"}}}, true},
		{[]string{"files", "images", "logo.png"}, "serve_files", Params{{Key: "filepath", Values: []string{"images", "logo.png"}}}, true},
		{[]string{"admin", "dashboard", "stats"}, "admin_panel", Params{{Key: "path", Values: []string{"dashboard", "stats"}}}, true},
		{[]string{"files", "~", "", "config.json"}, "static_filename_tilde", Params{{Key: "filename", Values: []string{"config.json"}}}, true},
		{[]string{"nonexistent"}, "", nil, false},
		{[]string{"api", "users", "123", "invalid"}, "", nil, false},
		{[]string{"files"}, "", nil, false},
	}

	for _, test := range tests {
		handler, params, found := tree.Get(test.path)
		assertEqual(t, found, test.found, fmt.Sprintf("Route %v found status", test.path))
		if found {
			assertEqual(t, handler.(string), test.expectedHandler, fmt.Sprintf("Route %v handler", test.path))
			assertParamsEqual(t, params, test.expectedParams, fmt.Sprintf("Route %v params", test.path))
		}
	}
}

// TestPriorityOrdering tests that routes are matched in the correct priority order
func TestPriorityOrdering(t *testing.T) {
	tree := NewRadixTree()

	// Add routes in a specific order to test priority
	tree.Add([]string{"static", "*filepath"}, "static_files")
	tree.Add([]string{"static", "js", "app.js"}, "app_js")
	tree.Add([]string{"api", ":version", "users"}, "api_users")
	tree.Add([]string{"api", "v1", "users"}, "api_v1_users")

	tests := []struct {
		path            []string
		expectedHandler string
		expectedParams  Params
	}{
		{
			[]string{"static", "js", "app.js"},
			"app_js",
			Params{},
		},
		{
			[]string{"static", "css", "style.css"},
			"static_files",
			Params{{Key: "filepath", Values: []string{"css", "style.css"}}},
		},
		{
			[]string{"api", "v1", "users"},
			"api_v1_users",
			Params{},
		},
		{
			[]string{"api", "v2", "users"},
			"api_users",
			Params{{Key: "version", Values: []string{"v2"}}},
		},
	}

	for _, test := range tests {
		handler, params, found := tree.Get(test.path)
		if !found {
			t.Errorf("Route %v should be found", test.path)
			continue
		}
		assertEqual(t, handler.(string), test.expectedHandler, fmt.Sprintf("Route %v handler", test.path))
		assertParamsEqual(t, params, test.expectedParams, fmt.Sprintf("Route %v params", test.path))
	}
}

// TestConflictingRoutes tests that conflicting routes are handled correctly
func TestConflictingRoutes(t *testing.T) {
	tree := NewRadixTree()
	tree.Add([]string{"users", ":id"}, "handler1")
	err := tree.Add([]string{"users", ":id"}, "handler2")
	if err == nil {
		t.Errorf("Expected error when adding conflicting route")
	}
}

func TestConflictingWildcardRoutes(t *testing.T) {
	tree := NewRadixTree()
	tree.Add([]string{"files", "*filepath"}, "handler1")
	err := tree.Add([]string{"files", "*filepath2"}, "handler2")
	if err == nil {
		t.Errorf("Expected error when adding conflicting wildcard route")
	}
}

func TestEmptyParameterName(t *testing.T) {
	tree := NewRadixTree()
	err := tree.Add([]string{"users", ":"}, "handler")
	if err == nil {
		t.Errorf("Expected error when adding route with empty parameter name")
	}
}

func TestEmptyWildcardName(t *testing.T) {
	tree := NewRadixTree()
	err := tree.Add([]string{"files", "*"}, "handler")
	if err != nil {
		t.Errorf("Did not expect error when adding wildcard with empty name")
	}
}

// TestInvalidRoutes tests invalid route patterns
func TestInvalidRoutes(t *testing.T) {
	// Test invalid route patterns that should return errors
	invalidRoutes := []struct {
		path []string
		desc string
	}{
		{[]string{":param", "*wildcard", ":param2"}, "parameter after wildcard"},
		{[]string{"*wildcard", "static"}, "path segment after wildcard"},
		{[]string{"*wildcard1", "*wildcard2"}, "wildcard after a wildcard"},
	}

	for _, test := range invalidRoutes {
		func() {
			tree := NewRadixTree()
			err := tree.Add(test.path, "handler")
			if err == nil {
				t.Errorf("Expected error for %s: %v", test.desc, test.path)
			}
		}()
	}
}

// TestEmptyTree tests operations on empty tree
func TestEmptyTree(t *testing.T) {
	tree := NewRadixTree()

	_, _, found := tree.Get([]string{})
	assertEqual(t, found, false, "Empty tree should not find root")

	_, _, found = tree.Get([]string{"users"})
	assertEqual(t, found, false, "Empty tree should not find any route")
}

// TestParamsGet tests the Params.Get method
func TestParamsGet(t *testing.T) {
	params := Params{
		{Key: "id", Values: []string{"123"}},
		{Key: "name", Values: []string{"john"}},
		{Key: "category", Values: []string{"tech"}},
	}

	// Test existing parameters
	value, found := params.Get("id")
	assertEqual(t, found, true, "Should find existing parameter")
	if found && len(value) > 0 {
		assertEqual(t, value[0], "123", "Should return correct value")
	}

	value, found = params.Get("name")
	assertEqual(t, found, true, "Should find existing parameter")
	if found && len(value) > 0 {
		assertEqual(t, value[0], "john", "Should return correct value")
	}

	// Test non-existing parameter
	value, found = params.Get("nonexistent")
	assertEqual(t, found, false, "Should not find non-existing parameter")
	assertEqual(t, len(value), 0, "Should return nil slice for non-existing parameter")
}

// BenchmarkStaticRoutes benchmarks static route lookup
func BenchmarkStaticRoutes(b *testing.B) {
	tree := NewRadixTree()

	routes := [][]string{
		{},
		{"api"},
		{"api", "users"},
		{"api", "posts"},
		{"api", "comments"},
		{"admin"},
		{"admin", "users"},
		{"admin", "posts"},
		{"public"},
		{"public", "css"},
		{"public", "js"},
		{"public", "images"},
	}

	for _, route := range routes {
		tree.Add(route, "handler")
	}

	for b.Loop() {
		tree.Get([]string{"api", "users"})
	}
}

// BenchmarkParameterRoutes benchmarks parameter route lookup
func BenchmarkParameterRoutes(b *testing.B) {
	tree := NewRadixTree()

	tree.Add([]string{"users", ":id"}, "user_show")
	tree.Add([]string{"users", ":id", "posts"}, "user_posts")
	tree.Add([]string{"users", ":id", "posts", ":post_id"}, "user_post_show")
	tree.Add([]string{"articles", ":slug", "comments", ":comment_id"}, "article_comment")

	for b.Loop() {
		tree.Get([]string{"users", "123", "posts", "456"})
	}
}

// BenchmarkWildcardRoutes benchmarks wildcard route lookup
func BenchmarkWildcardRoutes(b *testing.B) {
	tree := NewRadixTree()

	tree.Add([]string{"files", "*filepath"}, "files")
	tree.Add([]string{"admin", "*path"}, "admin")
	tree.Add([]string{"static", "*filename"}, "static")

	for b.Loop() {
		tree.Get([]string{"files", "documents", "images", "logo.png"})
	}
}

// BenchmarkMixedRoutes benchmarks mixed route types lookup
func BenchmarkMixedRoutes(b *testing.B) {
	tree := NewRadixTree()

	// Add a realistic set of routes
	routes := []struct {
		path    []string
		handler string
	}{
		{[]string{}, "home"},
		{[]string{"api"}, "api_root"},
		{[]string{"api", "v1"}, "api_v1"},
		{[]string{"api", "v1", "users"}, "users_index"},
		{[]string{"api", "v1", "users", ":id"}, "user_show"},
		{[]string{"api", "v1", "users", ":id", "posts"}, "user_posts"},
		{[]string{"api", "v1", "users", ":id", "posts", ":post_id"}, "user_post_show"},
		{[]string{"api", "v1", "posts"}, "posts_index"},
		{[]string{"api", "v1", "posts", ":id"}, "post_show"},
		{[]string{"api", "v1", "posts", ":id", "comments"}, "post_comments"},
		{[]string{"api", "v1", "posts", ":id", "comments", ":comment_id"}, "post_comment_show"},
		{[]string{"profile", ":username"}, "profile_show"},
		{[]string{"profile", ":username", "settings"}, "profile_settings"},
		{[]string{"profile", ":username", ":id", "hello"}, "profile_hello"},
		{[]string{"profile", ":username", "pic", "*picture"}, "profile_picture"},
		{[]string{"search", "*"}, "search"},
		{[]string{"search"}, "search"},
		{[]string{"admin"}, "admin_root"},
		{[]string{"admin", "users"}, "admin_users"},
		{[]string{"admin", "posts"}, "admin_posts"},
		{[]string{"admin", "*path"}, "admin_catch_all"},
		{[]string{"files", "*filepath"}, "serve_files"},
		{[]string{"static", "*filename"}, "static_files"},
	}

	for _, route := range routes {
		tree.Add(route.path, route.handler)
	}

	testPaths := [][]string{
		{},
		{"api", "v1", "users", "123"},
		{"api", "v1", "users", "456", "posts", "789"},
		{"admin", "dashboard", "stats"},
		{"files", "documents", "readme.txt"},
		{"static", "css", "style.css"},
		{"profile", "johndoe", "42", "hello"},
		{"profile", "janedoe", "pic", "avatar.png"},
		{"profile", "janedoe", "pic", "avatars", "2024", "avatar.png"},
		{"nonexistent"},
		{"search", "query", "advanced"},
	}

	for b.Loop() {
		for _, path := range testPaths {
			tree.Get(path)
		}
	}
}

func BenchmarkManyRoutes(b *testing.B) {
	tree := NewRadixTree()
	count := 5000
	batchLog := 1000
	stringsList := []string{}

	for i := range count {
		randomStr := fmt.Sprintf("%d-%d", rand.Int(), i)
		tree.Add([]string{"api", "serviceRandom@3", randomStr}, randomStr+"_handler")
		stringsList = append(stringsList, randomStr)
		if i%batchLog == 0 && i > 0 {
			b.Logf("Generated %d/%d items, %%%f", i, count, 100*float32(i)/float32(count))
		}
	}

	b.Logf("Generated %d items done.", count)

	for b.Loop() {
		randomIndex := rand.Intn(len(stringsList))
		path := stringsList[randomIndex]
		_, _, exists := tree.Get([]string{"api", "serviceRandom@3", path})
		if !exists {
			b.Errorf("Expected to found %s path in tree, but not found!", path)
		}
	}
}
