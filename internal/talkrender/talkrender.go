// Package talkrender renders the reveal.js talk page. The renderer
// chooses between plain / presenter / viewer mode based on the
// multiplex creds present in the request context (presenter) or the
// {socketID} path value (viewer); without either it renders the plain
// talk view.
package talkrender

import (
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fbegyn/website/internal/blog"
	"github.com/fbegyn/website/internal/middleware"
)

// Find returns the talk matching "talks/<year>/<slug>", or false when
// no such talk exists.
func Find(talks blog.Talks, year, slug string) (blog.Talk, bool) {
	target := "talks/" + year + "/" + slug
	for _, t := range talks {
		if t.Slug == target {
			return t, true
		}
	}
	return blog.Talk{}, false
}

// Render writes the talk page for p to w.
func Render(w http.ResponseWriter, r *http.Request, p blog.Talk) {
	var data templateData
	if v := r.Context().Value(middleware.MultiplexKey("secret")); v != nil {
		data.MSecret, _ = v.(string)
	}
	if v := r.Context().Value(middleware.MultiplexKey("socketID")); v != nil {
		data.MSocketID, _ = v.(string)
	}
	if data.MSocketID == "" {
		r = middleware.MultiplexViewerToContext(r)
		if v := r.Context().Value(middleware.MultiplexKey("socketID")); v != nil {
			data.MSocketID, _ = v.(string)
		}
	}

	switch {
	case data.MSecret != "" && data.MSocketID != "":
		data.Mode = "presenter"
	case data.MSocketID != "":
		data.Mode = "viewer"
	default:
		data.Mode = "plain"
	}

	data.Title = p.Title
	data.Slug = p.Slug
	data.Path = p.Path
	data.MURL = OriginURL(r)
	if data.Mode == "presenter" {
		viewerPath := strings.Replace(r.URL.Path, "/presenter/", "/", 1) + "/" + data.MSocketID
		data.ViewerURL = data.MURL + strings.TrimPrefix(viewerPath, "/")
	}

	t, err := template.ParseFiles("templates/talks/base.html", "templates/talks/talk.html")
	if err != nil {
		slog.Error("failed to parse talk template", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Cache-Control", "max-age=86400")
	if err := t.Execute(w, data); err != nil {
		slog.Error("failed to execute talk template", "error", err)
	}
}

// OriginURL returns "<scheme>://<host>/" for r, honouring
// X-Forwarded-Proto when it's present (we sit behind xff.Default so
// that header is trusted at this point in the chain).
func OriginURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return scheme + "://" + r.Host + "/"
}

type templateData struct {
	Title     string
	Slug      string
	Path      string
	Mode      string
	MSecret   string
	MSocketID string
	MURL      string
	ViewerURL string
}
