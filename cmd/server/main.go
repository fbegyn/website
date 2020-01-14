package main

import (
	"log"
	"net/http"
	"os"
)

var port = os.Getenv("SERVER_PORT")

func main() {
	if port == "" {
		port = "3114"
	}

	s, err := Build()
	if err != nil {
		log.Fatalln(err)
	}

	mux := http.NewServeMux()
	mux.handle("/", s)
}

type Site struct {
	mux *http.ServeMux
}

func Build() (*Site, error) {
	s := &Site{
		mux: http.NewServeMux(),
	}
	// Add HTTP routes here
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			s.renderTemplatePage("error.html", "can't find "+r.URL.Path).ServeHTTP(w, r)
			return
		}

		s.renderTemplatePage("index.html", nil).ServeHTTP(w, r)
	})
	return s, nil
}
