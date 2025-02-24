package main

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/fbegyn/website/cmd/server/internal"
	"github.com/fbegyn/website/cmd/server/internal/blog"
	"github.com/fbegyn/website/cmd/server/internal/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pageViews = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "page_views",
			Help: "number of views for a page (excluding posts)",
		}, []string{"page"},
	)
	postViews = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "post_views",
			Help: "number of views per post",
		}, []string{"post"},
	)
	talkViews = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "talk_views",
			Help: "number of views per talk",
		}, []string{"talk"},
	)
)

func (s *Site) renderPageTemplate(templateFile string, data interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var t *template.Template
		var err error
		t, err = template.ParseFiles("templates/base.html", "templates/"+templateFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error(
				"failed to render page",
				"error", err,
				"action", "renderPageTemplate",
				"page", templateFile,
			)
			fmt.Fprintf(w, "error: %v", err)
		}

		w.Header().Add("Cache-Control", "max-age=86400")

		err = t.Execute(w, data)
		if err != nil {
			slog.Error(
				"failed to execute template",
				"error", err,
				"action", "executeTemplate",
				"page", templateFile,
			)
		}
		pageViews.With(prometheus.Labels{"page": templateFile}).Inc()
	})
}

func (s *Site) renderPost(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/blog/" {
		http.Redirect(w, r, "/blog", http.StatusSeeOther)
		return
	}

	cmp := r.URL.Path[1:]
	var p blog.Entry
	var found bool
	for _, pst := range s.Posts {
		if pst.Link == cmp {
			p = pst
			found = true
		}
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		s.renderPageTemplate("error.html", "no such post found: "+r.RequestURI).ServeHTTP(w, r)
		return
	}

	var tags string

	if len(p.Tags) != 0 {
		for _, t := range p.Tags {
			tags = tags + " #" + strings.ReplaceAll(t, "-", "")
		}
	}

	s.renderPageTemplate("entries/entry.html", struct {
		Title             string
		Link              string
		BodyHTML          template.HTML
		Date              string
		Series, SeriesTag string
		Tags              string
		Prism             bool
	}{
		Title:    p.Title,
		Link:     p.Link,
		BodyHTML: p.BodyHTML,
		Date:     internal.IOS13Detri(p.Date),
		Tags:     tags,
		Prism:    true,
	}).ServeHTTP(w, r)
	postViews.With(prometheus.Labels{"post": filepath.Base(p.Link)}).Inc()
}

func (s *Site) renderTalk(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "talks/" {
		http.Redirect(w, r, "/talk", http.StatusSeeOther)
		return
	}

	var tmplData struct {
		Title     string
		Date      string
		Slug      string
		Path      string
		MSecret   string
		MSocketID string
		MURL      string
		ViewerURL string
	}

	cmp := r.PathValue("slug")
	year := r.PathValue("year")
	var p blog.Talk
	var found bool
	for _, pst := range s.Talks {
		if pst.Slug == ("talks/" + year + "/" + cmp) {
			p = pst
			found = true
		}
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		s.renderPageTemplate("error.html", "no such talk found: "+r.RequestURI).ServeHTTP(w, r)
		return
	}

	// lookup role type
	presenter, viewer := false, false
	secret := r.Context().Value(middleware.MultiplexKey("secret"))
	if secret == nil {
		r = middleware.MultiplexPresenterToContext(r)
		secret = r.Context().Value(middleware.MultiplexKey("secret"))
	}
	if secret != nil {
		tmplData.MSecret = secret.(string)
		presenter = true
	}
	socketID := r.Context().Value(middleware.MultiplexKey("socketID"))
	if socketID == nil {
		r = middleware.MultiplexViewerToContext(r)
		socketID = r.Context().Value(middleware.MultiplexKey("socketID"))
	}
	if socketID != nil {
		tmplData.MSocketID = socketID.(string)
		tmplData.ViewerURL = strings.Replace(r.URL.String(), "/presenter/", "/", 1) + "/" + socketID.(string)
		viewer = true
	}

	var base string
	if presenter {
		base = "talks/presenter.html"
	} else if viewer {
		base = "talks/viewer.html"
	} else {
		base = "talks/talk.html"
	}

	tmplData.Title = p.Title
	tmplData.Slug = p.Slug
	tmplData.Date = internal.IOS13Detri(p.Date)
	tmplData.Path = p.Path
	tmplData.Title = p.Title
	tmplData.MURL = "https://francis.begyn.be/-/multiplex/socket"

	s.renderTalkTemplate(base, tmplData).ServeHTTP(w, r)
	talkViews.With(prometheus.Labels{"talk": filepath.Base(p.Slug)}).Inc()
}

func (s *Site) renderTalkTemplate(templateFile string, data interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var t *template.Template
		var err error
		t, err = template.ParseFiles("templates/talks/base.html", "templates/"+templateFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error(
				"failed to render page",
				"error", err,
				"action", "renderPageTemplate",
				"page", templateFile,
			)
			fmt.Fprintf(w, "error: %v", err)
		}
		w.Header().Add("Cache-Control", "max-age=86400")

		err = t.Execute(w, data)
		if err != nil {
			slog.Error(
				"failed to execute template",
				"error", err,
				"action", "executeTemplate",
				"page", templateFile,
			)
		}
		pageViews.With(prometheus.Labels{"page": templateFile}).Inc()
	})
}

type Resume struct {
	Education    []Education
	Jobs         []Job
	Volunteering []Job
}

type Person struct {
	Name     string
	Github   string
	Mastodon string
	Twitter  string
	Address  string
}

type Education struct {
	Name        string
	Institution string
	Link        string
	Period      string
	Description string
}

type Job struct {
	Title       string
	Company     string
	Link        string
	Period      string
	Description string
}

func (s *Site) renderPageFromCUE(templateFile string, cueFile string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := cuecontext.New()
		instance := load.Instances([]string{"cue/" + cueFile, "cue/lib/education.cue", "cue/lib/job.cue"}, nil)
		resume := ctx.BuildInstance(instance[0])
		fmt.Println(resume)
		test := Resume{}
		resume.Decode(&test)
		var t *template.Template
		var err error
		t, err = template.ParseFiles("templates/base.html", "templates/"+templateFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error(
				"failed to render page",
				"error", err,
				"action", "renderPageTemplate",
				"page", templateFile,
			)
			fmt.Fprintf(w, "error: %v", err)
		}

		w.Header().Add("Cache-Control", "max-age=86400")

		err = t.Execute(w, test)
		if err != nil {
			slog.Error(
				"failed to execute template",
				"error", err,
				"action", "executeTemplate",
				"page", templateFile,
			)
		}
		pageViews.With(prometheus.Labels{"page": templateFile}).Inc()
	})
}

func (s *Site) renderNote(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/notes/" {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
		return
	}

	cmp := r.URL.Path[1:]
	var p blog.Note
	var found bool
	for _, pst := range s.Notes {
		if pst.Link == cmp {
			p = pst
			found = true
		}
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		s.renderPageTemplate("error.html", "no such note found: "+r.RequestURI).ServeHTTP(w, r)
		return
	}

	var tags string

	if len(p.Tags) != 0 {
		for _, t := range p.Tags {
			tags = tags + " #" + strings.ReplaceAll(t, "-", "")
		}
	}

	s.renderPageTemplate("note.html", struct {
		Title             string
		Link              string
		BodyHTML          template.HTML
		Date              string
		Series, SeriesTag string
		Tags              string
		Prism             bool
	}{
		Title:    p.Title,
		Link:     p.Link,
		BodyHTML: p.BodyHTML,
		Date:     internal.IOS13Detri(p.Date),
		Tags:     tags,
		Prism:    true,
	}).ServeHTTP(w, r)
	postViews.With(prometheus.Labels{"note": filepath.Base(p.Link)}).Inc()
}
