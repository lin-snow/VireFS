package virefs

import (
	"path"
	"strings"
)

// Route defines a single routing rule that maps matching keys to a prefix.
type Route struct {
	prefix string
	match  func(key string) bool
}

// RouteByExt creates a route that matches keys by file extension (case-insensitive).
// The prefix is prepended to matching keys.
//
//	RouteByExt("images/", ".jpg", ".jpeg", ".png")
func RouteByExt(prefix string, exts ...string) Route {
	lower := make(map[string]struct{}, len(exts))
	for _, ext := range exts {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		lower[strings.ToLower(ext)] = struct{}{}
	}
	return Route{
		prefix: prefix,
		match: func(key string) bool {
			ext := strings.ToLower(path.Ext(key))
			_, ok := lower[ext]
			return ok
		},
	}
}

// RouteByFunc creates a route with a custom match function.
//
//	RouteByFunc("archives/", func(key string) bool {
//	    return strings.HasSuffix(key, ".tar.gz") || strings.HasSuffix(key, ".zip")
//	})
func RouteByFunc(prefix string, fn func(key string) bool) Route {
	return Route{prefix: prefix, match: fn}
}

// DefaultRoute creates a catch-all route that matches any key.
// Place it last in the route list.
func DefaultRoute(prefix string) Route {
	return Route{prefix: prefix, match: func(string) bool { return true }}
}

// Schema organises files into directory prefixes based on routing rules.
// Rules are evaluated in declaration order; the first match wins.
//
// Schema.Resolve has the same signature as KeyFunc, so it plugs directly
// into WithLocalKeyFunc / WithObjectKeyFunc.
type Schema struct {
	routes []Route
}

// NewSchema creates a Schema from the given routes.
func NewSchema(routes ...Route) *Schema {
	return &Schema{routes: routes}
}

// Resolve maps a key to its storage path by applying the first matching route.
// If no route matches, the key is returned unchanged.
func (s *Schema) Resolve(key string) string {
	for _, r := range s.routes {
		if r.match(key) {
			return r.prefix + key
		}
	}
	return key
}
