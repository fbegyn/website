package main

import (
	"fmt"
	"html/template"
	"net/http"

	"within.website/ln"
	"within.website/ln/opname"
)

func (s *Site) renderTemplatePage(templateFname string, data interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := opname.With(r.Context(), "renderTemplatePage")
		fetag := "W/" + internal.Hash(templateFname, etag) + "-1"

		f := ln.F{"etag": fetag, "if_none_match": r.Header.Get("If-None-Match")}

		if r.Header.Get("If-None-Match") == fetag {
			http.Error(w, "Cached data OK", http.StatusNotModified)
			ln.Log(ctx, f, ln.Info("Cache hit"))
			return
		}

		var t *template.Template
		var err error

		t, err = template.ParseFiles("templates/base.html", "templates/"+templateFname)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			ln.Error(ctx, err, ln.F{"action": "renderTemplatePage", "page": templateFname})
			fmt.Fprintf(w, "error: %v", err)
		}

		w.Header().Set("ETag", fetag)
		w.Header().Set("Cache-Control", "max-age=432000")

		err = t.Execute(w, data)
		if err != nil {
			panic(err)
		}
	})
}
