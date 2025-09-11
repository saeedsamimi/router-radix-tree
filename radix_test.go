package radix_test

import (
	"fmt"
	"math/rand"
	"testing"

	radix "github.com/saeedsamimi/router-radix-tree"
	"github.com/stretchr/testify/assert"
)

func TestBasicRouting(t *testing.T) {
	tree := radix.NewRadixTree()

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
		routes := tree.Get(test.path)
		found := len(routes) > 0
		assert.Equal(t, found, test.found, fmt.Sprintf("Route %v found status", test.path))
		if found {
			handler := routes[0].Handler
			assert.Equal(t, handler.(string), test.expected, fmt.Sprintf("Route %v handler", test.path))
		}
	}
}

func TestParameterRouting(t *testing.T) {
	tree := radix.NewRadixTree()

	// Add routes with parameters
	tree.Add([]string{"users", ":id"}, "user_show")
	tree.Add([]string{"users", ":id", "posts"}, "user_posts")
	tree.Add([]string{"users", ":id", "posts", ":post_id"}, "user_post_show")
	tree.Add([]string{"articles", ":slug", "comments", ":comment_id"}, "article_comment")

	tests := []struct {
		path            []string
		expectedHandler string
		expectedParams  radix.Params
		found           bool
	}{
		{
			[]string{"users", "123"},
			"user_show",
			radix.Params{{Key: "id", Values: []string{"123"}}},
			true,
		},
		{
			[]string{"users", "456", "posts"},
			"user_posts",
			radix.Params{{Key: "id", Values: []string{"456"}}},
			true,
		},
		{
			[]string{"users", "789", "posts", "101"},
			"user_post_show",
			radix.Params{{Key: "id", Values: []string{"789"}}, {Key: "post_id", Values: []string{"101"}}},
			true,
		},
		{
			[]string{"articles", "golang-tips", "comments", "5"},
			"article_comment",
			radix.Params{{Key: "slug", Values: []string{"golang-tips"}}, {Key: "comment_id", Values: []string{"5"}}},
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
		routes := tree.Get(test.path)
		found := len(routes) > 0
		assert.Equal(t, found, test.found, fmt.Sprintf("Route %v found status", test.path))
		if found {
			route := routes[0]
			handler := route.Handler
			params := route.Params
			assert.Equal(t, handler.(string), test.expectedHandler, fmt.Sprintf("Route %v handler", test.path))
			assert.Equal(t, params, test.expectedParams, fmt.Sprintf("Route %v params", test.path))
		}
	}
}

func TestWildcardRouting(t *testing.T) {
	tree := radix.NewRadixTree()

	// Add wildcard routes
	tree.Add([]string{"files", "*filepath"}, "files")
	tree.Add([]string{"admin", "*path"}, "admin_catch_all")
	tree.Add([]string{"static", "*filename"}, "static_files")

	tests := []struct {
		path            []string
		expectedHandler string
		expectedParams  radix.Params
		found           bool
	}{
		{
			[]string{"files", "documents", "readme.txt"},
			"files",
			radix.Params{{Key: "filepath", Values: []string{"documents", "readme.txt"}}},
			true,
		},
		{
			[]string{"admin", "dashboard"},
			"admin_catch_all",
			radix.Params{{Key: "path", Values: []string{"dashboard"}}},
			true,
		},
		{
			[]string{"admin", "users", "settings", "advanced"},
			"admin_catch_all",
			radix.Params{{Key: "path", Values: []string{"users", "settings", "advanced"}}},
			true,
		},
		{
			[]string{"static", "css", "style.css"},
			"static_files",
			radix.Params{{Key: "filename", Values: []string{"css", "style.css"}}},
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
		routes := tree.Get(test.path)
		found := len(routes) > 0
		assert.Equal(t, found, test.found, fmt.Sprintf("Route %v found status", test.path))
		if found {
			route := routes[0]
			handler := route.Handler
			params := route.Params
			assert.Equal(t, handler.(string), test.expectedHandler, fmt.Sprintf("Route %v handler", test.path))
			assert.Equal(t, params, test.expectedParams, fmt.Sprintf("Route %v params", test.path))
		}
	}
}

func TestMixedRouting(t *testing.T) {
	tree := radix.NewRadixTree()

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
		expectedParams  radix.Params
		found           bool
	}{
		{[]string{}, "root", nil, true},
		{[]string{"api"}, "api_root", nil, true},
		{[]string{"api", "users"}, "api_users", nil, true},
		{[]string{"api", "users", "123"}, "api_user_show", radix.Params{{Key: "id", Values: []string{"123"}}}, true},
		{[]string{"api", "users", "456", "profile"}, "api_user_profile", radix.Params{{Key: "id", Values: []string{"456"}}}, true},
		{[]string{"api", "posts", "789", "comments", "101"}, "api_comment", radix.Params{{Key: "post_id", Values: []string{"789"}}, {Key: "comment_id", Values: []string{"101"}}}, true},
		{[]string{"files", "images", "logo.png"}, "serve_files", radix.Params{{Key: "filepath", Values: []string{"images", "logo.png"}}}, true},
		{[]string{"admin", "dashboard", "stats"}, "admin_panel", radix.Params{{Key: "path", Values: []string{"dashboard", "stats"}}}, true},
		{[]string{"files", "~", "", "config.json"}, "static_filename_tilde", radix.Params{{Key: "filename", Values: []string{"config.json"}}}, true},
		{[]string{"nonexistent"}, "", nil, false},
		{[]string{"api", "users", "123", "invalid"}, "", nil, false},
		{[]string{"files"}, "", nil, false},
	}

	for _, test := range tests {
		routes := tree.Get(test.path)
		found := len(routes) > 0
		assert.Equal(t, found, test.found, fmt.Sprintf("Route %v found status", test.path))
		if found {
			route := routes[0]
			handler := route.Handler
			params := route.Params
			assert.Equal(t, handler.(string), test.expectedHandler, fmt.Sprintf("Route %v handler", test.path))
			assert.Equal(t, params, test.expectedParams, fmt.Sprintf("Route %v params", test.path))
		}
	}
}

func TestPriorityOrdering(t *testing.T) {
	tree := radix.NewRadixTree()

	// Add routes in a specific order to test priority
	tree.Add([]string{"static", "*filepath"}, "static_files")
	tree.Add([]string{"static", "js", "app.js"}, "app_js")
	tree.Add([]string{"api", ":version", "users"}, "api_users")
	tree.Add([]string{"api", "v1", "users"}, "api_v1_users")

	tests := []struct {
		path            []string
		expectedHandler string
		expectedParams  radix.Params
	}{
		{
			[]string{"static", "js", "app.js"},
			"app_js",
			nil,
		},
		{
			[]string{"static", "css", "style.css"},
			"static_files",
			radix.Params{{Key: "filepath", Values: []string{"css", "style.css"}}},
		},
		{
			[]string{"api", "v1", "users"},
			"api_v1_users",
			nil,
		},
		{
			[]string{"api", "v2", "users"},
			"api_users",
			radix.Params{{Key: "version", Values: []string{"v2"}}},
		},
	}

	for _, test := range tests {
		routes := tree.Get(test.path)
		if len(routes) == 0 {
			t.Errorf("Route %v should be found", test.path)
			continue
		}
		route := routes[0]
		handler := route.Handler
		params := route.Params
		assert.Equal(t, handler.(string), test.expectedHandler, fmt.Sprintf("Route %v handler", test.path))
		assert.Equal(t, params, test.expectedParams, fmt.Sprintf("Route %v params", test.path))
	}
}

func TestConflictingRoutes1(t *testing.T) {
	tree := radix.NewRadixTree()
	tree.Add([]string{"users", ":id"}, "handler1")
	_, err := tree.Add([]string{"users", ":id"}, "handler2")
	if err == nil {
		t.Errorf("Expected error when adding conflicting route")
	}
}

func TestConflictingRoutes2(t *testing.T) {
	tree := radix.NewRadixTree()
	tree.Add([]string{"users", "id"}, "handler1")
	tree.Add([]string{"users", ":id"}, "handler2")
	_, err := tree.Add([]string{"users", "id"}, "handler3")
	if err == nil {
		t.Errorf("Expected error when adding conflicting route")
	}
}

func TestConflictingRoutes3(t *testing.T) {
	tree := radix.NewRadixTree()
	tree.Add([]string{":id"}, "handler1")
	tree.Add([]string{"id"}, "handler2")
	_, err := tree.Add([]string{":id"}, "handler3")
	if err == nil {
		t.Errorf("Expected error when adding conflicting route")
	}
}

func TestConflictingWildcardRoutes(t *testing.T) {
	tree := radix.NewRadixTree()
	tree.Add([]string{"files", "*filepath"}, "handler1")
	tree.Add([]string{"files", "*filepath2"}, "handler2")
	_, err := tree.Add([]string{"files", "*filepath"}, "handler3")
	if err != nil {
		t.Errorf("Did not expect error when adding conflicting wildcard routes")
	}
}

func TestEmptyParameterName(t *testing.T) {
	tree := radix.NewRadixTree()
	_, err := tree.Add([]string{"users", ":"}, "handler")
	if err != nil {
		t.Errorf("Did not Expect error when adding route with empty parameter name")
	}
}

func TestEmptyWildcardName(t *testing.T) {
	tree := radix.NewRadixTree()
	_, err := tree.Add([]string{"files", "*"}, "handler")
	if err != nil {
		t.Errorf("Did not expect error when adding wildcard with empty name")
	}
}

func TestTreeSize(t *testing.T) {
	tree := radix.NewRadixTree()
	assert.Zero(t, tree.Size())

	tree.Add([]string{"users"}, "handler1")
	assert.Equal(t, uint32(1), tree.Size())

	tree.Add([]string{"users", ":id"}, "handler2")
	assert.Equal(t, uint32(2), tree.Size())

	tree.Add([]string{}, "root_handler")
	assert.Equal(t, uint32(3), tree.Size())

	tree.Add([]string{"files", "*filepath"}, "handler3")
	assert.Equal(t, uint32(4), tree.Size())

	tree.Add([]string{"*files"}, "handler_file")
	assert.Equal(t, uint32(5), tree.Size())
}

func TestTreeInsertion1(t *testing.T) {
	tree := radix.NewRadixTree()
	nw1, _ := tree.Add([]string{"users", ":id"}, "user_show")
	nw2, _ := tree.Add([]string{"users", ":id", "posts"}, "user_posts")
	assert.Equal(t, nw1.PathName(), ":id")
	assert.Equal(t, nw2.PathName(), "posts")
	parent1, ok1 := nw1.Parent()
	parent2, ok2 := nw2.Parent()
	assert.Equal(t, ok1, true)
	assert.Equal(t, ok2, true)
	assert.Equal(t, parent1.PathName(), "users")
	assert.Equal(t, parent2.PathName(), ":id")
	assert.Equal(t, parent2, nw1)
}

func TestTreeInsertion2(t *testing.T) {
	tree := radix.NewRadixTree()
	nw1, _ := tree.Add([]string{"files", "*filepath"}, "files")
	nw2, _ := tree.Add([]string{"files", "~", "", ":filename"}, "static_filename_tilde")
	assert.Equal(t, "*filepath", nw1.PathName())
	assert.Equal(t, ":filename", nw2.PathName())
	parent1, ok1 := nw1.Parent()
	parent2, ok2 := nw2.Parent()
	assert.Equal(t, true, ok1)
	assert.Equal(t, true, ok2)
	assert.Equal(t, "files", parent1.PathName())
	assert.Equal(t, "", parent2.PathName())
}

func TestTreeInsertion3(t *testing.T) {
	tree := radix.NewRadixTree()
	tree.Add([]string{"api", "v1", "users"}, "api_v1_users")
	tree.Add([]string{"api", ":version", "users"}, "api_users")
	tree.Add([]string{"api", "v2", "users"}, "api_v2_users")
	nw, _ := tree.Add([]string{"api", "v2", "users", "profile"}, "api_v2_user_profile")
	assert.Equal(t, nw.PathName(), "profile")
	treeRoot := tree.Root()
	counter := 0
	parent, _ := nw.Parent()
	ok := false
	for !parent.Equal(treeRoot) {
		counter++
		if counter > 5 {
			t.Errorf("Exceeded expected tree depth")
			return
		}
		nw = parent
		parent, ok = nw.Parent()
		if !ok {
			t.Errorf("Expected parent node, got none")
			return
		}
	}
	assert.Equal(t, "api", nw.PathName())
	assert.Equal(t, uint32(4), tree.Size())
	assert.Equal(t, uint32(4), nw.Size())
}

func TestInvalidRoutes(t *testing.T) {
	// Test invalid route patterns that should return errors
	invalidRoutes := []struct {
		path []string
		desc string
	}{
		{[]string{"static", ":param", "*wildcard", ":param2"}, "parameter after wildcard"},
		{[]string{"*wildcard", "static"}, "path segment after wildcard"},
		{[]string{"*wildcard1", "*wildcard2"}, "wildcard after a wildcard"},
	}

	for _, test := range invalidRoutes {
		tree := radix.NewRadixTree()
		_, err := tree.Add(test.path, "handler")
		if err == nil {
			t.Errorf("Expected error for %s: %v", test.desc, test.path)
		}
	}
}

func TestEmptyTree(t *testing.T) {
	tree := radix.NewRadixTree()

	routes := tree.Get([]string{})
	found := len(routes) > 0
	assert.Equal(t, found, false, "Empty tree should not find root")

	routes = tree.Get([]string{"users"})
	found = len(routes) > 0
	assert.Equal(t, found, false, "Empty tree should not find any route")
}

func TestMultipleMatchingRoutes(t *testing.T) {
	tree := radix.NewRadixTree()

	// Add routes that can match the same path
	tree.Add([]string{"api", ":version"}, "api_version")
	tree.Add([]string{"api", "*path"}, "api_catch_all")
	tree.Add([]string{"files", ":filename"}, "file_param")
	tree.Add([]string{"files", "*filepath"}, "file_wildcard")
	tree.Add([]string{"files", "~", ":apiname", ":filename"}, "filename1")
	tree.Add([]string{"files", "~", ":apiname", ":address"}, "filename2")

	tests := []struct {
		path             []string
		expectedHandlers []string
		expectedParams   []radix.Params
	}{
		{
			[]string{"api", "v1"},
			[]string{"api_version", "api_catch_all"},
			[]radix.Params{
				{{Key: "version", Values: []string{"v1"}}},
				{{Key: "path", Values: []string{"v1"}}},
			},
		},
		{
			[]string{"files", "test.txt"},
			[]string{"file_param", "file_wildcard"},
			[]radix.Params{
				{{Key: "filename", Values: []string{"test.txt"}}},
				{{Key: "filepath", Values: []string{"test.txt"}}},
			},
		},
		{
			[]string{"files", "~", "myapi", "data.json"},
			[]string{"filename1", "filename2", "file_wildcard"},
			[]radix.Params{
				{{Key: "apiname", Values: []string{"myapi"}}, {Key: "filename", Values: []string{"data.json"}}},
				{{Key: "apiname", Values: []string{"myapi"}}, {Key: "address", Values: []string{"data.json"}}},
				{{Key: "filepath", Values: []string{"~", "myapi", "data.json"}}},
			},
		},
	}

	for _, test := range tests {
		routes := tree.Get(test.path)
		if len(routes) != len(test.expectedHandlers) {
			t.Errorf("Expected %d routes, got %d for path %v", len(test.expectedHandlers), len(routes), test.path)
			continue
		}

		// Create maps for unordered comparison
		actualHandlers := make(map[string]bool)
		actualParams := make(map[string]radix.Params)

		for _, route := range routes {
			handler := route.Handler.(string)
			actualHandlers[handler] = true
			actualParams[handler] = route.Params
		}

		// Check handlers exist (unordered)
		for _, expectedHandler := range test.expectedHandlers {
			if !actualHandlers[expectedHandler] {
				t.Errorf("Expected handler %s not found for path %v", expectedHandler, test.path)
			}
		}

		// Check params match for corresponding handlers
		for i, expectedHandler := range test.expectedHandlers {
			if actualParam, exists := actualParams[expectedHandler]; exists {
				assert.Equal(t, actualParam, test.expectedParams[i], fmt.Sprintf("Params for handler %s in path %v", expectedHandler, test.path))
			}
		}
	}
}

// TestParamsGet tests the radix.Params.Get method
func TestParamsGet(t *testing.T) {
	params := radix.Params{
		{Key: "id", Values: []string{"123"}},
		{Key: "name", Values: []string{"john"}},
		{Key: "category", Values: []string{"tech"}},
	}

	// Test existing parameters
	value, found := params.Get("id")
	assert.Equal(t, found, true, "Should find existing parameter")
	if found && len(value) > 0 {
		assert.Equal(t, value[0], "123", "Should return correct value")
	}

	value, found = params.Get("name")
	assert.Equal(t, found, true, "Should find existing parameter")
	if found && len(value) > 0 {
		assert.Equal(t, value[0], "john", "Should return correct value")
	}

	// Test non-existing parameter
	value, found = params.Get("nonexistent")
	assert.Equal(t, found, false, "Should not find non-existing parameter")
	assert.Equal(t, len(value), 0, "Should return nil slice for non-existing parameter")
}

func TestDeletion(t *testing.T) {
	tree := radix.NewRadixTree()

	// Add routes
	tree.Add([]string{"users"}, "handler1")
	tree.Add([]string{"users", ":id"}, "handler2")
	tree.Add([]string{"users", ":id", ":policy", "*filename"}, "advanced_handler")
	tree.Add([]string{"admin"}, "handler3")
	tree.Add([]string{"files", "*filepath"}, "handler4")

	assert.Equal(t, uint32(5), tree.Size())

	// Delete a route
	err := tree.Delete([]string{"users", ":id", ":policy", "*filename"})
	assert.Nil(t, err, "Route should be deleted without error")
	assert.Equal(t, uint32(4), tree.Size(), "Tree size should decrease")

	routes := tree.Get([]string{"users", "123"})
	assert.Equal(t, len(routes), 1, "Deleted route should not be found")
	assert.Equal(t, routes[0].Handler.(string), "handler2", "Non-deleted route should have correct handler")

	routes = tree.Get([]string{"users"})
	assert.Equal(t, len(routes), 1, "Non-deleted route should be found")
	assert.Equal(t, routes[0].Handler.(string), "handler1", "Non-deleted route should have correct handler")

	err = tree.Delete([]string{"files", "*filepath"})
	assert.Nil(t, err, "Route should be deleted without error")
	assert.Equal(t, uint32(3), tree.Size(), "Tree size should decrease")

	routes = tree.Get([]string{"files", "documents", "file.txt"})
	assert.Equal(t, len(routes), 0, "Deleted route should not be found")

	routes = tree.Get([]string{"files"})
	assert.Len(t, routes, 0, "Non-deleted route should be found")
	assert.Equal(t, tree.Size(), uint32(3), "Tree size should remain the same")
}

func BenchmarkStaticRoutes(b *testing.B) {
	tree := radix.NewRadixTree()

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

func BenchmarkParameterRoutes(b *testing.B) {
	tree := radix.NewRadixTree()

	tree.Add([]string{"users", ":id"}, "user_show")
	tree.Add([]string{"users", ":id", "posts"}, "user_posts")
	tree.Add([]string{"users", ":id", "posts", ":post_id"}, "user_post_show")
	tree.Add([]string{"articles", ":slug", "comments", ":comment_id"}, "article_comment")

	for b.Loop() {
		tree.Get([]string{"users", "123", "posts", "456"})
	}
}

func BenchmarkWildcardRoutes(b *testing.B) {
	tree := radix.NewRadixTree()

	tree.Add([]string{"files", "*filepath"}, "files")
	tree.Add([]string{"admin", "*path"}, "admin")
	tree.Add([]string{"static", "*filename"}, "static")

	for b.Loop() {
		tree.Get([]string{"files", "documents", "images", "logo.png"})
	}
}

func BenchmarkMixedRoutes(b *testing.B) {
	tree := radix.NewRadixTree()

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
	tree := radix.NewRadixTree()
	count := 5000
	stringsList := []string{}

	for i := range count {
		randomStr := fmt.Sprintf("%d-%d", rand.Int(), i)
		tree.Add([]string{"api", "serviceRandom@3", randomStr}, randomStr+"_handler")
		stringsList = append(stringsList, randomStr)
	}

	for b.Loop() {
		randomIndex := rand.Intn(len(stringsList))
		path := stringsList[randomIndex]
		routes := tree.Get([]string{"api", "serviceRandom@3", path})
		exists := len(routes) > 0
		if !exists {
			b.Errorf("Expected to found %s path in tree, but not found!", path)
		}
	}
}
