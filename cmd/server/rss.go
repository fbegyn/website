package main

import (
	"log/slog"
	"net/http"
	"time"
)

var rssTime = time.Now()

func (s *Site) createFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/rss+xml")
	err := s.rssFeed.WriteRss(w)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		slog.Error(
			"failed to write rss feed",
			"error", err,
			"action", "generateRSS",
			"uri", r.RequestURI,
			"host", r.Host,
		)
	}
}
