package main

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/fbegyn/website/cmd/server/internal/blog"
	"github.com/fbegyn/website/cmd/server/internal/middleware"
	"github.com/sebest/xff"
	"github.com/snabb/sitemap"
	"within.website/ln"
	"within.website/ln/ex"
	"within.website/ln/opname"
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
	ctx := ln.WithF(opname.With(context.Background(), "main"), ln.F{
		"port": port,
	})

	// Create the site
	s, err := Build()
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("Build"))
	}

	// Create the webmux and attach it to the website
	mux := http.NewServeMux()
	mux.Handle("/", s)

	// Enable logging and serve the website
	ln.Log(ctx, ln.Action("http_listening"))
	ln.FatalErr(ctx, http.ListenAndServe(":"+port, mux))
}

type Site struct {
	Posts blog.Entries
	About template.HTML

	mux   *http.ServeMux
	xffmw *xff.XFF
}

// Make so our site struct can serve http requests
func (s *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a context for the request
	ctx := opname.With(r.Context(), "site.ServeHTTP")
	ctx = ln.WithF(ctx, ln.F{
		"user_agent": r.Header.Get("User-Agent"),
	})
	r = r.WithContext(ctx)

	// Add a unique ID to each request
	middleware.RequestID(s.xffmw.Handler(ex.HTTPLog(s.mux))).ServeHTTP(w, r)
}

func Build() (*Site, error) {
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
		return nil, err
	}

	// Struct that represents the website
	s := &Site{
		mux:   http.NewServeMux(),
		xffmw: xffmw,
	}

	posts, err := blog.LoadEntriesDir("./blog/", "blog")
	if err != nil {
		return nil, err
	}
	s.Posts = posts

	for _, entry := range s.Posts {
		smap.Add(&sitemap.URL{
			Loc:        "https://francis.begyn.be/" + entry.Link,
			LastMod:    &entry.Date,
			ChangeFreq: sitemap.Monthly,
		})
	}

	// Add HTTP routes here
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			s.renderPageTemplate("error.html", "page not found: "+r.URL.Path).ServeHTTP(w, r)
			return
		}
		s.renderPageTemplate("index.html", nil).ServeHTTP(w, r)
	})

	s.mux.Handle("/about", s.renderPageTemplate("about.html", nil))
	s.mux.Handle("/blog", s.renderPageTemplate("blogindex.html", s.Posts))
	s.mux.Handle("/blog/", http.HandlerFunc(s.renderPost))

	// server static files
	s.mux.Handle("/static/", http.FileServer(http.Dir(".")))

	return s, nil
}
