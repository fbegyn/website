package main

import (
	"context"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fbegyn/website/cmd/server/internal/blog"
	"github.com/fbegyn/website/cmd/server/internal/front"
	"github.com/fbegyn/website/cmd/server/internal/middleware"
	"github.com/gorilla/feeds"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sebest/xff"
	"github.com/snabb/sitemap"
	"within.website/ln"
	"within.website/ln/ex"
	"within.website/ln/opname"
)

var (
	port          = os.Getenv("SERVER_PORT")
	arbDate       = time.Date(2020, time.January, 9, 0, 0, 0, 0, time.UTC)
	fileDownloads = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "file_downloads",
			Help: "Number of times a file is downloaded",
		}, []string{"file"},
	)
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
	s, err, _ := Build(ctx)
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

// Site represents the website structure and data
type Site struct {
	Posts blog.Entries
	Talks []Talk
	About template.HTML

	rssFeed *feeds.Feed

	mux   *http.ServeMux
	xffmw *xff.XFF
}

type Talk struct {
	Title string
	Date  string
	Link  string
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

// Build renders the entire website
func Build(ctx context.Context) (*Site, error, chan int) {
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
		return nil, err, nil
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

	posts, err := blog.LoadEntriesDir("./blog/", "blog")
	if err != nil {
		return nil, err, nil
	}
	s.Posts = posts

	// TODO: work in progress to host PDFs on the site of talks/workshops
	err = filepath.Walk("./talks", func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return nil
		}

		var t Talk
		_, err = front.Unmarshal(content, &t)
		if err != nil {
			return err
		}
		s.Talks = append(s.Talks, t)
		return nil
	})

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
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			s.renderPageTemplate("error.html", "page not found: "+r.URL.Path).ServeHTTP(w, r)
			return
		}
		s.renderPageTemplate("index.html", nil).ServeHTTP(w, r)
	})

	ln.Log(ctx, ln.Action("background_gen"), ln.Info("spinnign up background process channel"))
	stop := make(chan int)

	s.mux.Handle("/metrics", promhttp.Handler())
	s.mux.Handle("/about", middleware.Metrics("about", s.renderPageTemplate("about.html", nil)))
	s.mux.Handle("/blog", middleware.Metrics("blog", s.renderPageTemplate("blogindex.html", s.Posts)))
	s.mux.Handle("/blog/rss", middleware.Metrics("rss", http.HandlerFunc(s.createFeed)))
	s.mux.Handle("/blog.rss", middleware.Metrics("rss", http.HandlerFunc(s.createFeed)))
	s.mux.Handle("/blog/", middleware.Metrics("post", http.HandlerFunc(s.renderPost)))
	s.mux.Handle("/talks", middleware.Metrics("talk", s.renderPageTemplate("talks.html", s.Talks)))

	handler := http.StripPrefix("/talk/", http.FileServer(http.Dir("static/pdf/talks")))
	s.mux.Handle("/talk/", handler)
	//s.mux.HandleFunc("/francis_begyn_cv_eng.pdf", func(w http.ResponseWriter, r *http.Request) {
	//	fileDownloads.With(prometheus.Labels{"file": "francis_begyn_cv_eng.pdf"}).Inc()
	//	http.ServeFile(w, r, "./cv/francis_begyn_cv_eng.pdf")
	//})
	//s.mux.HandleFunc("/francis_begyn_cv_nl.pdf", func(w http.ResponseWriter, r *http.Request) {
	//	fileDownloads.With(prometheus.Labels{"file": "francis_begyn_cv_nl.pdf"}).Inc()
	//	http.ServeFile(w, r, "./cv/francis_begyn_cv_nl.pdf")
	//})
	s.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/favicon.ico")
	})

	s.mux.HandleFunc("/.well-known/cf-2fa-verify.txt", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/text")
	})

	// server static files
	s.mux.Handle("/static/", http.FileServer(http.Dir(".")))

	return s, nil, stop
}
