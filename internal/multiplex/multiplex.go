// Package multiplex is an in-process replacement for the standalone
// reveal.js multiplex server. It speaks a small custom protocol:
//
//   GET  /multiplex/token                 -> {secret, socketId}
//   WS   /multiplex/presenter             -> presenter connection;
//                                            receives JSON frames
//                                            {secret, socketId, state}
//   GET  /multiplex/{socketID}/events     -> SSE stream for viewers,
//                                            each event payload is JSON
//                                            {socketId, state}
//
// socketId is the hex-encoded SHA-256 of the secret, so any presenter
// frame whose secret hashes to the claimed socketId is accepted.
package multiplex

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// Hub fans out presenter state changes to subscribed viewer streams.
// The zero value is not usable; call New().
type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[*subscriber]struct{}
}

type subscriber struct {
	ch chan []byte
}

// Token is the credential pair handed out by /multiplex/token. The
// presenter keeps the secret; the socketId is the public identifier
// embedded in viewer URLs.
type Token struct {
	Secret   string `json:"secret"`
	SocketID string `json:"socketId"`
}

type presenterFrame struct {
	Secret   string          `json:"secret"`
	SocketID string          `json:"socketId"`
	State    json.RawMessage `json:"state"`
}

type viewerFrame struct {
	SocketID string          `json:"socketId"`
	State    json.RawMessage `json:"state"`
}

// New returns a ready-to-use Hub.
func New() *Hub {
	return &Hub{subs: make(map[string]map[*subscriber]struct{})}
}

// IssueToken returns a fresh {secret, socketId} pair. socketId is
// derived from secret as hex(sha256(secret)).
func (h *Hub) IssueToken() (Token, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Token{}, err
	}
	secret := hex.EncodeToString(raw[:])
	sum := sha256.Sum256([]byte(secret))
	return Token{Secret: secret, SocketID: hex.EncodeToString(sum[:])}, nil
}

// Register attaches the multiplex routes onto mux under /multiplex/*.
func (h *Hub) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /multiplex/token", h.TokenHandler)
	mux.HandleFunc("GET /multiplex/{socketID}/events", h.EventsHandler)
	mux.Handle("GET /multiplex/presenter", h.PresenterHandler())
}

// TokenHandler serves GET /multiplex/token.
func (h *Hub) TokenHandler(w http.ResponseWriter, r *http.Request) {
	tok, err := h.IssueToken()
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(tok)
}

// EventsHandler serves GET /multiplex/{socketID}/events as an SSE
// stream. Each subscriber gets a buffered channel; slow consumers
// have messages dropped rather than blocking the broadcast.
func (h *Hub) EventsHandler(w http.ResponseWriter, r *http.Request) {
	socketID := r.PathValue("socketID")
	if socketID == "" {
		http.Error(w, "missing socketID", http.StatusBadRequest)
		return
	}

	// http.ResponseController unwraps logging/xff/etc. wrappers so we
	// can flush even when the chain hides the underlying Flusher.
	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	sub := &subscriber{ch: make(chan []byte, 16)}
	h.subscribe(socketID, sub)
	defer h.unsubscribe(socketID, sub)

	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		return
	}

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-keepalive.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		case msg, ok := <-sub.ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", msg); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}

// PresenterHandler upgrades to WebSocket and listens for presenter
// state frames. A frame is accepted when sha256(secret) matches the
// claimed socketId; otherwise the frame is dropped.
func (h *Hub) PresenterHandler() http.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			var frame presenterFrame
			if err := websocket.JSON.Receive(ws, &frame); err != nil {
				return
			}
			sum := sha256.Sum256([]byte(frame.Secret))
			if hex.EncodeToString(sum[:]) != frame.SocketID {
				slog.Warn("multiplex: secret/socketId mismatch", "socketId", frame.SocketID)
				continue
			}
			payload, err := json.Marshal(viewerFrame{SocketID: frame.SocketID, State: frame.State})
			if err != nil {
				continue
			}
			h.fanout(frame.SocketID, payload)
		}
	})
}

func (h *Hub) subscribe(socketID string, s *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[socketID] == nil {
		h.subs[socketID] = make(map[*subscriber]struct{})
	}
	h.subs[socketID][s] = struct{}{}
}

func (h *Hub) unsubscribe(socketID string, s *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.subs[socketID]; ok {
		delete(subs, s)
		if len(subs) == 0 {
			delete(h.subs, socketID)
		}
	}
	close(s.ch)
}

func (h *Hub) fanout(socketID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for s := range h.subs[socketID] {
		select {
		case s.ch <- msg:
		default:
		}
	}
}
