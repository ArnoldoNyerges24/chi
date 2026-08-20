package chi

import (
	"context"
	"net/http"
	"strings"
)

// ContextKey is a value for use with context.WithValue.
type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "chi context value " + k.name
}

var (
	// RouteCtxKey is the context.Context key to store the RouteContext.
	RouteCtxKey = &contextKey{"RouteContext"}
)

// Context is the context object for the chi router.
type Context struct {
	Routes Routes

	// RouteMethod is the HTTP method of the route.
	RouteMethod string

	// RoutePath is the routing path pattern for the particular request.
	RoutePath string

	// RoutePatterns is a slice of route patterns matched.
	RoutePatterns []string

	// URLParams is a slice of URL parameters matched.
	URLParams URLParams

	// routeParams stores the key-value pairs of URL parameters.
	routeParams routeParams
}

// NewRouteContext returns a new RouteContext object.
func NewRouteContext() *Context {
	return &Context{
		routeParams: routeParams{
			Keys:   make([]string, 0, 8),
			Values: make([]string, 0, 8),
		},
	}
}

// Reset cleans the RouteContext object for reuse.
func (ctx *Context) Reset() {
	ctx.Routes = nil
	ctx.RouteMethod = ""
	ctx.RoutePath = ""
	ctx.RoutePatterns = ctx.RoutePatterns[:0]
	ctx.URLParams.Keys = ctx.URLParams.Keys[:0]
	ctx.URLParams.Values = ctx.URLParams.Values[:0]
	ctx.routeParams.Keys = ctx.routeParams.Keys[:0]
	ctx.routeParams.Values = ctx.routeParams.Values[:0]
}

// URLParam returns the value of a URL parameter.
func (ctx *Context) URLParam(key string) string {
	for i := len(ctx.URLParams.Keys) - 1; i >= 0; i-- {
		if ctx.URLParams.Keys[i] == key {
			return ctx.URLParams.Values[i]
		}
	}
	return ""
}

// RoutePattern builds the routing path pattern for the particular request
// at the current point in execution.
func (ctx *Context) RoutePattern() string {
	if len(ctx.RoutePatterns) == 0 {
		return ""
	}

	var pattern string
	for _, p := range ctx.RoutePatterns {
		if p == "" {
			continue
		}
		if pattern == "" {
			pattern = p
			continue
		}
		if strings.HasSuffix(pattern, "/*") {
			pattern = pattern[:len(pattern)-2]
		} else if strings.HasSuffix(pattern, "*") {
			pattern = pattern[:len(pattern)-1]
		}

		if strings.HasSuffix(pattern, "/") && strings.HasPrefix(p, "/") {
			pattern += p[1:]
		} else if !strings.HasSuffix(pattern, "/") && !strings.HasPrefix(p, "/") {
			pattern += "/" + p
		} else {
			pattern += p
		}
	}
	return pattern
}

// RouteContext returns the RouteContext object for the request.
func RouteContext(ctx context.Context) *Context {
	val, _ := ctx.Value(RouteCtxKey).(*Context)
	return val
}

// URLParam returns the value of a URL parameter from the request context.
func URLParam(r *http.Request, key string) string {
	if rctx := RouteContext(r.Context()); rctx != nil {
		return rctx.URLParam(key)
	}
	return ""
}

// URLParamFromCtx returns the value of a URL parameter from the context.
func URLParamFromCtx(ctx context.Context, key string) string {
	if rctx := RouteContext(ctx); rctx != nil {
		return rctx.URLParam(key)
	}
	return ""
}

// ServerRouteContext returns the RouteContext object for the request.
// Deprecated: Use RouteContext instead.
func ServerRouteContext(ctx context.Context) *Context {
	return RouteContext(ctx)
}

// routeParams is a helper struct to store URL parameters.
type routeParams struct {
	Keys   []string
	Values []string
}

func (s *routeParams) Add(key, value string) {
	s.Keys = append(s.Keys, key)
	s.Values = append(s.Values, value)
}

// URLParams is a helper struct to store URL parameters.
type URLParams struct {
	Keys   []string
	Values []string
}

// Add adds a URL parameter to the URLParams object.
func (s *URLParams) Add(key, value string) {
	s.Keys = append(s.Keys, key)
	s.Values = append(s.Values, value)
}