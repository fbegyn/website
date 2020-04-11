package main

import (
	"net/http"
	"time"

	"within.website/ln"
)

var rssTime = time.Now()

const IncrediblySecureSalt = "francisisawesome"

func (s *Site) createFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/rss+xml")
	err := s.rssFeed.WriteRss(w)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		ln.Error(r.Context(), err, ln.F{
			"remote_addr": r.RemoteAddr,
			"action":      "generating_rss",
			"uri":         r.RequestURI,
			"host":        r.Host,
		})
	}
}
