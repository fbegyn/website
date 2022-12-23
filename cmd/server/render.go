package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fbegyn/website/cmd/server/internal"
	"github.com/fbegyn/website/cmd/server/internal/addons"
	"github.com/fbegyn/website/cmd/server/internal/blog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/russross/blackfriday"
	gogpt "github.com/sashabaranov/go-gpt3"
	"within.website/ln"
	"within.website/ln/opname"
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

	s.renderPageTemplate("blogpost.html", struct {
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

func GenGPTPost() blog.Entry {
	rsp, err := http.Get(os.Getenv("PROMPT_URL"))
	if err != nil {
		return blog.Entry{
			Title: "Woops",
			Body:  "# Something went wrong\n\nWe might have hit a rate limit. Try again in 1 minute or wait till the site owner fixes it.",
			Tags:  []string{"gpt", "blog", "error"},
			Date:  time.Now(),
		}
	}
	topic, err := io.ReadAll(rsp.Body)
	if err != nil {
		return blog.Entry{
			Title: "Woops",
			Body:  "# Something went wrong\n\nWe might have hit a rate limit. Try again in 1 minute or wait till the site owner fixes it.",
			Tags:  []string{"gpt", "blog", "error"},
			Date:  time.Now(),
		}
	}
	prompt := "Write a paragraph about " + string(topic) + " in markdown"
	c := gogpt.NewClient(os.Getenv("OPENAI_KEY"))
	ctx := context.Background()
	req := gogpt.CompletionRequest{
		Model:       "text-davinci-003",
		Temperature: 0.9,
		MaxTokens:   4000 - len(prompt),
		Prompt:      prompt,
	}
	resp, err := c.CreateCompletion(ctx, req)
	if err != nil {
		return blog.Entry{
			Title: "Woops",
			Body:  "# Something went wrong\n\nWe might have hit a rate limit. Try again in 1 minute or wait till the site owner fixes it.",
			Tags:  []string{"gpt", "blog", "error"},
			Date:  time.Now(),
		}
	}
	body := "# " + string(topic) + "\n" + resp.Choices[0].Text
	return blog.Entry{
		Title:   string(topic),
		Summary: "This is just a blog post generated by OpenAI models, you don't care about a summary",
		Body:    body,
		Tags:    []string{"gpt", "blog"},
		Date:    time.Now(),
	}
}

func (s *Site) randomGPTPost(w http.ResponseWriter, r *http.Request, buffer <-chan blog.Entry) {
	if r.RequestURI == "/blog/" {
		http.Redirect(w, r, "/blog", http.StatusSeeOther)
		return
	}

	p := <-buffer

	var tags string
	if len(p.Tags) != 0 {
		for _, t := range p.Tags {
			tags = tags + " #" + strings.ReplaceAll(t, "-", "")
		}
	}

	w.Header().Add("Cache-Control", "no-cache")
	s.renderPageTemplate("gptpost.html", struct {
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
		BodyHTML: template.HTML(blackfriday.Run([]byte(p.Body))),
		Date:     internal.IOS13Detri(p.Date),
		Tags:     tags,
		Prism:    true,
	}).ServeHTTP(w, r)
	postViews.With(prometheus.Labels{"post": filepath.Base(p.Link)}).Inc()
}

func (s *Site) officeStatus(w http.ResponseWriter, r *http.Request, deskHost, hassHost string) {
	height, err := addons.GetDeskHeight(deskHost)
	if err != nil {
		fmt.Println(err)
	}
	s.renderPageTemplate("office.html", struct {
		OfficeTemp     int
		OfficeHumidity int
		DeskHeight     int
		DeskDur        int
		DeskHeightUnit string
	}{
		DeskHeight:     height,
		DeskHeightUnit: "cm",
	}).ServeHTTP(w, r)
}
