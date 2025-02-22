package multiplex

import (
	"net/http"
	"path/filepath"
)

func ContentHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if mimeType := detectContentType(r.URL.Path); mimeType != "" {
		// 	w.Header().Set("Content-Type", mimeType)
		// }

		if r.URL.Path != "/" {
			h.ServeHTTP(w, r)
			return
		}

		// tmpl, err := template.New("slide template").Parse(slideTemplate)
		// if err != nil {
		// 	log.Printf("error:%v", err)
		// 	http.NotFound(w, r)
		// 	return
		// }

		// if err := tmpl.Execute(w, params); err != nil {
		// 	log.Fatalf("error:%v", err)
		// }
	})
}

func AssetsHandler(prefix string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//if mimeType := detectContentType(r.URL.Path); mimeType != "" {
		//	w.Header().Set("Content-Type", mimeType)
		//}

		if prefix != "" {
			r.URL.Path = filepath.Join(prefix, r.URL.Path)
			r.URL.RawPath = filepath.Join(prefix, r.URL.RawPath)
		}

		h.ServeHTTP(w, r)
	})
}
