package router

import (
	"net/http"
	"regexp"
	"strings"
)

// Route represents a single route
type Route struct {
	Method  string
	Pattern *regexp.Regexp
	Handler http.HandlerFunc
	Auth    bool // Whether this route requires authentication
}

// Router handles HTTP request routing
type Router struct {
	routes []Route
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		routes: make([]Route, 0),
	}
}

// AddRoute adds a route to the router
func (r *Router) AddRoute(method, pattern string, handler http.HandlerFunc, requireAuth bool) {
	regex := regexp.MustCompile("^" + pattern + "$")
	route := Route{
		Method:  method,
		Pattern: regex,
		Handler: handler,
		Auth:    requireAuth,
	}
	r.routes = append(r.routes, route)
}

// GET adds a GET route
func (r *Router) GET(pattern string, handler http.HandlerFunc, requireAuth bool) {
	r.AddRoute("GET", pattern, handler, requireAuth)
}

// POST adds a POST route
func (r *Router) POST(pattern string, handler http.HandlerFunc, requireAuth bool) {
	r.AddRoute("POST", pattern, handler, requireAuth)
}

// PUT adds a PUT route
func (r *Router) PUT(pattern string, handler http.HandlerFunc, requireAuth bool) {
	r.AddRoute("PUT", pattern, handler, requireAuth)
}

// DELETE adds a DELETE route
func (r *Router) DELETE(pattern string, handler http.HandlerFunc, requireAuth bool) {
	r.AddRoute("DELETE", pattern, handler, requireAuth)
}

// OPTIONS adds an OPTIONS route
func (r *Router) OPTIONS(pattern string, handler http.HandlerFunc, requireAuth bool) {
	r.AddRoute("OPTIONS", pattern, handler, requireAuth)
}

// Match finds a matching route for the given method and path
func (r *Router) Match(method, path string) (*Route, map[string]string) {
	for _, route := range r.routes {
		if route.Method == method || route.Method == "*" {
			if matches := route.Pattern.FindStringSubmatch(path); matches != nil {
				// Extract path parameters
				params := make(map[string]string)
				subexpNames := route.Pattern.SubexpNames()
				for i, match := range matches[1:] {
					if i+1 < len(subexpNames) && subexpNames[i+1] != "" {
						params[subexpNames[i+1]] = match
					}
				}
				return &route, params
			}
		}
	}
	return nil, nil
}

// ServeHTTP implements http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route, params := r.Match(req.Method, req.URL.Path)
	if route == nil {
		http.NotFound(w, req)
		return
	}

	// Add path parameters to request context
	if len(params) > 0 {
		ctx := req.Context()
		for key, value := range params {
			ctx = req.Context()
			req = req.WithContext(ctx)
			// Store params in a way that can be retrieved later
			req.Header.Set("X-Path-Param-"+key, value)
		}
	}

	route.Handler(w, req)
}

// GetPathParam retrieves a path parameter from the request
func GetPathParam(r *http.Request, key string) string {
	return r.Header.Get("X-Path-Param-" + key)
}

// RouteGroup represents a group of routes with common prefix and middleware
type RouteGroup struct {
	router     *Router
	prefix     string
	middleware []func(http.Handler) http.Handler
}

// NewRouteGroup creates a new route group
func (r *Router) NewRouteGroup(prefix string) *RouteGroup {
	return &RouteGroup{
		router:     r,
		prefix:     prefix,
		middleware: make([]func(http.Handler) http.Handler, 0),
	}
}

// Use adds middleware to the route group
func (rg *RouteGroup) Use(middleware func(http.Handler) http.Handler) {
	rg.middleware = append(rg.middleware, middleware)
}

// GET adds a GET route to the group
func (rg *RouteGroup) GET(pattern string, handler http.HandlerFunc, requireAuth bool) {
	fullPattern := rg.prefix + pattern
	wrappedHandler := rg.wrapHandler(handler)
	rg.router.GET(fullPattern, wrappedHandler, requireAuth)
}

// POST adds a POST route to the group
func (rg *RouteGroup) POST(pattern string, handler http.HandlerFunc, requireAuth bool) {
	fullPattern := rg.prefix + pattern
	wrappedHandler := rg.wrapHandler(handler)
	rg.router.POST(fullPattern, wrappedHandler, requireAuth)
}

// PUT adds a PUT route to the group
func (rg *RouteGroup) PUT(pattern string, handler http.HandlerFunc, requireAuth bool) {
	fullPattern := rg.prefix + pattern
	wrappedHandler := rg.wrapHandler(handler)
	rg.router.PUT(fullPattern, wrappedHandler, requireAuth)
}

// DELETE adds a DELETE route to the group
func (rg *RouteGroup) DELETE(pattern string, handler http.HandlerFunc, requireAuth bool) {
	fullPattern := rg.prefix + pattern
	wrappedHandler := rg.wrapHandler(handler)
	rg.router.DELETE(fullPattern, wrappedHandler, requireAuth)
}

// wrapHandler applies group middleware to a handler
func (rg *RouteGroup) wrapHandler(handler http.HandlerFunc) http.HandlerFunc {
	wrapped := http.Handler(handler)

	// Apply middleware in reverse order
	for i := len(rg.middleware) - 1; i >= 0; i-- {
		wrapped = rg.middleware[i](wrapped)
	}

	return wrapped.ServeHTTP
}

// PathMatcher provides utilities for path matching
type PathMatcher struct{}

// NewPathMatcher creates a new path matcher
func NewPathMatcher() *PathMatcher {
	return &PathMatcher{}
}

// MatchPrefix checks if path starts with prefix
func (pm *PathMatcher) MatchPrefix(path, prefix string) bool {
	return strings.HasPrefix(path, prefix)
}

// MatchExact checks for exact path match
func (pm *PathMatcher) MatchExact(path, pattern string) bool {
	return path == pattern
}

// MatchRegex checks if path matches regex pattern
func (pm *PathMatcher) MatchRegex(path, pattern string) bool {
	matched, err := regexp.MatchString(pattern, path)
	if err != nil {
		return false
	}
	return matched
}

// ExtractPathSegments splits path into segments
func (pm *PathMatcher) ExtractPathSegments(path string) []string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 1 && segments[0] == "" {
		return []string{}
	}
	return segments
}

// BuildPath constructs a path from segments
func (pm *PathMatcher) BuildPath(segments ...string) string {
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

// ServiceRouter provides service-specific routing
type ServiceRouter struct {
	authRoutes       []string
	projectRoutes    []string
	simulationRoutes []string
}

// NewServiceRouter creates a new service router
func NewServiceRouter() *ServiceRouter {
	return &ServiceRouter{
		authRoutes: []string{
			// Public auth routes
			"/api/auth/login",
			"/api/auth/register",
			"/api/auth/refresh",
			"/api/auth/forgot-password",
			"/api/auth/reset-password",
			"/api/auth/verify-email",
			"/api/auth/resend-verification",
			"/api/auth/logout",
			"/api/auth/health",

			// User management routes
			"/api/user/profile",
			"/api/user/change-password",
			"/api/user/account",
			"/api/user/sessions",
			"/api/user/sessions/[^/]+",
			"/api/user/stats",

			// RBAC routes
			"/api/rbac/my-roles",
			"/api/rbac/my-permissions",

			// Admin routes
			"/api/admin/roles",
			"/api/admin/permissions",
			"/api/admin/users/assign-role",
			"/api/admin/users/remove-role",
			"/api/admin/users/[^/]+/roles",

			// Gateway-specific auth routes
			"/api/auth/validate",
			"/api/auth/profile",
			"/api/auth/permissions",
		},
		projectRoutes: []string{
			"/api/projects",
			"/api/projects/[0-9]+",
			"/api/projects/[0-9]+/share",
			"/api/projects/[0-9]+/permissions",
		},
		simulationRoutes: []string{
			"/api/simulations",
			"/api/simulations/[0-9]+",
			"/api/simulations/[0-9]+/start",
			"/api/simulations/[0-9]+/stop",
			"/api/simulations/[0-9]+/status",
			"/api/simulations/[0-9]+/results",
		},
	}
}

// GetServiceForPath determines which service should handle the path
func (sr *ServiceRouter) GetServiceForPath(path string) string {
	// Check auth routes
	for _, route := range sr.authRoutes {
		if matched, _ := regexp.MatchString("^"+route+"$", path); matched {
			return "auth"
		}
	}

	// Check project routes
	for _, route := range sr.projectRoutes {
		if matched, _ := regexp.MatchString("^"+route+"$", path); matched {
			return "project"
		}
	}

	// Check simulation routes
	for _, route := range sr.simulationRoutes {
		if matched, _ := regexp.MatchString("^"+route+"$", path); matched {
			return "simulation"
		}
	}

	return "unknown"
}

// IsProtectedRoute checks if a route requires authentication
func (sr *ServiceRouter) IsProtectedRoute(path string) bool {
	// Public auth routes (login, register, etc.)
	publicRoutes := []string{
		"/api/auth/login",
		"/api/auth/register",
		"/api/auth/refresh",
		"/api/auth/forgot-password",
		"/api/auth/reset-password",
		"/api/auth/verify-email",
		"/api/auth/resend-verification",
		"/api/auth/health",
	}

	// Auth routes that handle their own authentication
	authHandledRoutes := []string{
		"/api/auth/validate",
		"/api/auth/profile",
		"/api/auth/permissions",
	}

	for _, route := range publicRoutes {
		if matched, _ := regexp.MatchString("^"+route+"$", path); matched {
			return false
		}
	}

	// Auth routes handle their own authentication internally
	for _, route := range authHandledRoutes {
		if matched, _ := regexp.MatchString("^"+route+"$", path); matched {
			return false
		}
	}

	// All other API routes are protected
	return strings.HasPrefix(path, "/api/")
}
