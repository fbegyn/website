package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/fbegyn/website/cmd/server/internal"
	"github.com/fbegyn/website/cmd/server/internal/blog"
	"within.website/ln"
	"within.website/ln/opname"
)

func (s *Site) renderPageTemplate(templateFile string, data interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := opname.With(r.Context(), "renderTemplate")

		var t *template.Template
		var err error
		t, err = template.ParseFiles("templates/base.html", "templates/"+templateFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			ln.Error(ctx, err, ln.F{
				"action": "renderPageTemplate",
				"page":   templateFile,
			})
			fmt.Fprintf(w, "error: %v", err)
		}

		w.Header().Add("Cache-Control", "max-age=86400")

		err = t.Execute(w, data)
		if err != nil {
			ln.FatalErr(ctx, err, ln.F{
				"action": "executeTemplate",
				"page":   templateFile,
			})
		}
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

	s.renderPageTemplate("blogpost.html", struct {
		Title             string
		Link              string
		BodyHTML          template.HTML
		Date              string
		Series, SeriesTag string
		Tags              string
	}{
		Title:    p.Title,
		Link:     p.Link,
		BodyHTML: p.BodyHTML,
		Date:     internal.IOS13Detri(p.Date),
		Tags:     tags,
	}).ServeHTTP(w, r)
}
