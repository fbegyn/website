package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

type MultiplexKey string

func GetMultiplexCredentials(host string) ([2]string, error) {
	resp, err := http.Get(host + "/token")
	if err != nil {
		return [2]string{}, err
	}
	defer resp.Body.Close()

	var payload struct {
		Secret   string `json:"secret,omitempty"`
		SocketID string `json:"socketId"`
	}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		return [2]string{}, err
	}
	return [2]string{payload.Secret, payload.SocketID}, nil
}

func InjectMultiplexCredentials(creds [2]string, r *http.Request) *http.Request {
	temp := r.WithContext(context.WithValue(r.Context(), MultiplexKey("secret"), creds[0]))
	temp = temp.WithContext(context.WithValue(temp.Context(), MultiplexKey("socketID"), creds[1]))
	return temp
}

func MultiplexViewerToContext(r *http.Request) *http.Request {
	socketID := r.PathValue("socketID")
	if socketID != "" {
		temp := r.WithContext(context.WithValue(r.Context(), MultiplexKey("socketID"), socketID))
		return temp
	}
	return r
}
func MultiplexPresenterToContext(r *http.Request) *http.Request {
	socketID := r.PathValue("secret")
	if socketID != "" {
		temp := r.WithContext(context.WithValue(r.Context(), MultiplexKey("secret"), socketID))
		return temp
	}
	return r
}

func MultiplexCreateCredentials(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		credentials, err := GetMultiplexCredentials("http://localhost:8000")
		if err == nil {
			r = InjectMultiplexCredentials(credentials, r)
			next.ServeHTTP(w, r)
			return
		}
		slog.Error("failed to load the multiplex credentials from multiplex server", "error", err)
		http.Error(w, "failed to get token", http.StatusBadGateway)
	})
}
