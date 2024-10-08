package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/fbegyn/website/cmd/server/internal"
	"github.com/fbegyn/website/cmd/server/internal/blog"
	"github.com/fbegyn/website/cmd/server/internal/middleware"
	"github.com/gorilla/feeds"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sebest/xff"
	"github.com/snabb/sitemap"
	"within.website/ln/ex"
)

var (
	port    = os.Getenv("SERVER_PORT")
	arbDate = time.Date(2020, time.January, 9, 0, 0, 0, 0, time.UTC)
)

func main() {
	// Port selection for the webste
	if port == "" {
		port = "3114"
	}

	// create a context in which the website can run and add logging
	ctx := context.Background()
	logger := slog.Default()
	slog.SetDefault(logger)
	logger = slog.Default().With(slog.String("func", "main"))

	// Create the site
	s, _, err := Build(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to build website", slog.Any("err", err), slog.String("action", "build"))
		os.Exit(1)
	}

	// Create the webmux and attach it to the website
	mux := http.NewServeMux()
	mux.Handle("/", s)

	// Enable logging and serve the website
	logger.InfoContext(ctx, "starting webserver", slog.String("action", "http_listen"))
	if err = http.ListenAndServe(":"+port, mux); err != nil {
		logger.ErrorContext(ctx, "web server shut down unexpectantly", slog.Any("err", err))
	} else {
		logger.InfoContext(ctx, "web server shut down clean")
	}
}

// Site represents the website structure and data
type Site struct {
	Posts blog.Entries
	Talks blog.Talks
	About template.HTML

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
func Build(ctx context.Context) (*Site, chan int, error) {
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
			Copyright: "Copyright 2020 Francis Begyn. Any and all opinions listed here are my own and not representative of my employers; future, past and present.",
		},
	}

	// load the blog entries from disk
	posts, err := blog.LoadEntriesDir("./blog/", "blog")
	if err != nil {
		return nil, nil, err
	}
	s.Posts = posts
	// load talk entries from disk
	talks, err := blog.LoadTalksDir("./talks/", "talk")
	if err != nil {
		return nil, nil, err
	}
	s.Talks = talks

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
	s.mux.Handle("GET /blog/rss", middleware.Metrics("rss", http.HandlerFunc(s.createFeed)))
	s.mux.Handle("GET /blog.rss", middleware.Metrics("rss", http.HandlerFunc(s.createFeed)))
	s.mux.Handle("GET /blog/", middleware.Metrics("post", http.HandlerFunc(s.renderPost)))
	s.mux.Handle("GET /talks", middleware.Metrics("talk", s.renderPageTemplate("talks.html", s.Talks)))

	handler := http.StripPrefix("/talk/", http.FileServer(http.Dir("static/pdf/talks")))
	s.mux.Handle("GET /talk/", handler)
	s.mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/favicon.ico")
	})

	s.mux.HandleFunc("GET /.well-known/cf-2fa-verify.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/text")
	})

	// server static files
	s.mux.Handle("GET /static/", http.FileServer(http.Dir(".")))

	return s, stop, nil
}
