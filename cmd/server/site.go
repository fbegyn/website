package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/fbegyn/website/cmd/server/internal"
	"github.com/fbegyn/website/cmd/server/internal/blog"
	"github.com/fbegyn/website/cmd/server/internal/middleware"
	"github.com/gorilla/feeds"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sebest/xff"
	"github.com/snabb/sitemap"
	"within.website/ln/ex"
)

// Site represents the website structure and data
type Site struct {
	Posts  blog.Entries
	Talks  blog.Talks
	Notes  blog.Notes
	About  template.HTML
	Drafts bool

	rssFeed *feeds.Feed

	mux   *http.ServeMux
	xffmw *xff.XFF
}

// Make so our site struct can serve http requests
func (s *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a context for the request
	ctx := context.Background()
	ctx = context.WithValue(ctx, internal.ContextKey("func"), "site.ServeHTTP")
	ctx = context.WithValue(ctx, internal.ContextKey("user_agent"), r.Header.Get("User-Agent"))
	r = r.WithContext(ctx)
	// Add a unique ID to each request
	middleware.RequestID(s.xffmw.Handler(ex.HTTPLog(s.mux))).ServeHTTP(w, r)
}

// Build renders the entire website
func Build(ctx context.Context, publishDrafts bool) (*Site, chan int, error) {
	ctx = context.WithValue(ctx, internal.ContextKey("func"), "Build")
	// Define sitemap for the website
	smap := sitemap.New()
	smap.Add(&sitemap.URL{
		Loc:        "https://francis.begyn.be/",
		ChangeFreq: "monthly",
		LastMod:    &arbDate,
	})

	// Handle X-Forwarde-For headers
	xffmw, err := xff.Default()
	if err != nil {
		return nil, nil, err
	}

	// Struct that represents the website
	s := &Site{
		mux:   http.NewServeMux(),
		xffmw: xffmw,
		rssFeed: &feeds.Feed{
			Title: "Francis Begyn's thoughts",
			Link: &feeds.Link{
				Href: "https://francis.begyn.be/blog",
			},
			Description: "A collection of my thoughts on the interwebs",
			Author: &feeds.Author{
				Name:  "Francis Begyn",
				Email: "francis@begyn.be",
			},
			Created:   rssTime,
			Copyright: "Copyright 2020-2025 Francis Begyn. Any and all opinions listed here are my own and not representative of my employers; future, past and present.",
		},
	}

	// load the blog entries from disk
	posts, err := blog.LoadEntriesDir("./blog/", "blog", publishDrafts)
	if err != nil {
		return nil, nil, err
	}
	s.Posts = posts
	// load talk entries from disk
	talks, err := blog.LoadTalksDir("./talks/", "talk", publishDrafts)
	if err != nil {
		return nil, nil, err
	}
	s.Talks = talks
	// load note entries from disk
	notes, err := blog.LoadNotesDir("./notes/", "note", publishDrafts)
	if err != nil {
		return nil, nil, err
	}
	s.Notes = notes

	for _, entry := range s.Posts {
		if entry.Draft {
			continue
		}
		s.rssFeed.Items = append(s.rssFeed.Items, &feeds.Item{
			Title: entry.Title,
			Link: &feeds.Link{
				Href: "https://francis.begyn.be/" + entry.Link,
			},
			Author: &feeds.Author{
				Name:  "Francis Begyn",
				Email: "francis@begyn.be",
			},
			Description: entry.Summary,
			Created:     entry.Date,
			Content:     string(entry.BodyHTML),
		})
		smap.Add(&sitemap.URL{
			Loc:        "https://francis.begyn.be/" + entry.Link,
			LastMod:    &entry.Date,
			ChangeFreq: sitemap.Monthly,
		})
	}

	// Add HTTP routes here
	s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			s.renderPageTemplate("error.html", "page not found: "+r.URL.Path).ServeHTTP(w, r)
			return
		}
		s.renderPageTemplate("index.html", nil).ServeHTTP(w, r)
	})

	slog.InfoContext(ctx, "spinning up background process channel", slog.String("action", "background_gen"))
	stop := make(chan int)

	s.mux.Handle("GET /metrics", promhttp.Handler())
	s.mux.Handle("GET /about", middleware.Metrics("about", s.renderPageTemplate("about.html", nil)))
	s.mux.Handle("GET /blog", middleware.Metrics("blog", s.renderPageTemplate("blogindex.html", s.Posts)))
	s.mux.Handle("GET /notes", middleware.Metrics("notes", s.renderPageTemplate("noteindex.html", s.Notes)))
	s.mux.Handle("GET /resume", middleware.Metrics("resume", s.renderPageFromCUE("resume.html", "resume.cue")))
	s.mux.Handle("GET /blog/rss", middleware.Metrics("rss", http.HandlerFunc(s.createFeed)))
	s.mux.Handle("GET /blog.rss", middleware.Metrics("rss", http.HandlerFunc(s.createFeed)))
	s.mux.Handle("GET /blog/", middleware.Metrics("post", http.HandlerFunc(s.renderPost)))
	s.mux.Handle("GET /note/", middleware.Metrics("notes", http.HandlerFunc(s.renderNote)))
	s.mux.Handle("GET /talks", middleware.Metrics("talk", s.renderPageTemplate("talks.html", s.Talks)))
	s.mux.Handle("GET /talks/{year}/{slug}", middleware.Metrics("talks", http.HandlerFunc(s.renderTalk)))

	handler := http.StripPrefix("/talk/", http.FileServer(http.Dir("static/pdf/talks")))
	s.mux.Handle("GET /talk/", handler)
	s.mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/favicon.ico")
	})

	s.mux.HandleFunc("GET /.well-known/cf-2fa-verify.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/text")
	})

	// server static files
	s.mux.Handle("GET /static/", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	s.mux.Handle("GET /static/talks/img/", http.StripPrefix("/static/talks/img/", http.FileServer(http.Dir("./static/talks/img"))))
	// handle talk content
	t := blog.TalkFS{}
	s.mux.Handle("GET /static/talks/", http.StripPrefix("/static/", TalkFSHandler(http.FileServerFS(t))))

	return s, stop, nil
}

type WebsiteResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *WebsiteResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func TalkFSHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		temp := &WebsiteResponseWriter{ResponseWriter: w}
		h.ServeHTTP(temp, r)
		if temp.status >= 400 {
			slog.ErrorContext(r.Context(), "failed to open TalkFS and serve talk", "status", temp.status)
		}
	}
}
