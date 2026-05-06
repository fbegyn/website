package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/fbegyn/website/cmd/talk/internal/multiplex"
)

type MultiplexKey string

// InjectMultiplexCredentials stuffs a freshly issued {secret, socketId}
// pair into the request context so downstream handlers can render them
// into the presenter template.
func InjectMultiplexCredentials(tok multiplex.Token, r *http.Request) *http.Request {
	r = r.WithContext(context.WithValue(r.Context(), MultiplexKey("secret"), tok.Secret))
	r = r.WithContext(context.WithValue(r.Context(), MultiplexKey("socketID"), tok.SocketID))
	return r
}

func MultiplexViewerToContext(r *http.Request) *http.Request {
	socketID := r.PathValue("socketID")
	if socketID != "" {
		return r.WithContext(context.WithValue(r.Context(), MultiplexKey("socketID"), socketID))
	}
	return r
}

func MultiplexPresenterToContext(r *http.Request) *http.Request {
	secret := r.PathValue("secret")
	if secret != "" {
		return r.WithContext(context.WithValue(r.Context(), MultiplexKey("secret"), secret))
	}
	return r
}

// MultiplexCreateCredentials issues a fresh token from the in-process
// Hub and injects it into the request context before delegating.
func MultiplexCreateCredentials(hub *multiplex.Hub, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok, err := hub.IssueToken()
		if err != nil {
			slog.Error("failed to issue multiplex token", "error", err)
			http.Error(w, "failed to issue token", http.StatusInternalServerError)
			return
		}
		r = InjectMultiplexCredentials(tok, r)
		next.ServeHTTP(w, r)
	}
}
