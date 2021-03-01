package context

import (
	"context"
	"net/http"
)

var pathVars = contextKey("pathVars")

// Vars parses path variables and returns a map.
func Vars(r *http.Request) map[string]string {
	vars, ok := r.Context().Value(pathVars).(map[string]string)
	if ok {
		return vars
	}

	return nil
}

// WithPathVars writes params into given r and returns a new http.Request.
func WithPathVars(r *http.Request, params map[string]string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), pathVars, params))
}

type contextKey string

func (c contextKey) String() string {
	return "rest/internal/context key: " + string(c)
}
