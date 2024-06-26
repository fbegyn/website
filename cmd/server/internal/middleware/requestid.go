package middleware

import (
	"context"
	"net/http"

	"github.com/celrenheit/sandflake"
	"github.com/fbegyn/website/cmd/server/internal"
)

// RequestID appends a unique (sandflake) request ID to each request's
// X-Request-Id header field, much like Heroku's router does.
func RequestID(next http.Handler) http.Handler {
	var g sandflake.Generator
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := g.Next().String()

		if rid := r.Header.Get("X-Request-Id"); rid != "" {
			id = rid + "," + id
		}

		ctx := context.WithValue(r.Context(), internal.ContextKey("request_id"), id)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-Id", id)
		r.Header.Set("X-Request-Id", id)

		next.ServeHTTP(w, r)
	})
}
