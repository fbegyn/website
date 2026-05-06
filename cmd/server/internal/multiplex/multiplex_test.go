package multiplex

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestIssueTokenSocketIDIsHashOfSecret(t *testing.T) {
	h := New()
	tok, err := h.IssueToken()
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if tok.Secret == "" || tok.SocketID == "" {
		t.Fatalf("empty token: %+v", tok)
	}
	sum := sha256.Sum256([]byte(tok.Secret))
	if got, want := tok.SocketID, hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("socketID = %q, want %q", got, want)
	}
}

func TestTokenHandlerReturnsJSON(t *testing.T) {
	h := New()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/multiplex/token", nil)
	h.TokenHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var tok Token
	if err := json.NewDecoder(rec.Body).Decode(&tok); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tok.Secret == "" || tok.SocketID == "" {
		t.Fatalf("empty token: %+v", tok)
	}
}

func TestPresenterFanoutToViewers(t *testing.T) {
	h := New()
	tok, err := h.IssueToken()
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/multiplex/presenter"
	ws, err := websocket.Dial(wsURL, "", srv.URL)
	if err != nil {
		t.Fatalf("dial presenter: %v", err)
	}
	defer ws.Close()

	// Subscribe a viewer over SSE.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/multiplex/"+tok.SocketID+"/events", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("dial viewer: %v", err)
	}
	defer resp.Body.Close()

	// Drain the initial ": connected" comment so the next data: frame
	// is the broadcast we expect.
	br := bufio.NewReader(resp.Body)
	if _, err := readEvent(br); err != nil {
		t.Fatalf("read initial: %v", err)
	}

	// Give the subscribe a moment so the broadcast lands after.
	time.Sleep(20 * time.Millisecond)

	frame := presenterFrame{
		Secret:   tok.Secret,
		SocketID: tok.SocketID,
		State:    json.RawMessage(`{"indexh":2,"indexv":0}`),
	}
	if err := websocket.JSON.Send(ws, frame); err != nil {
		t.Fatalf("send: %v", err)
	}

	got, err := readEvent(br)
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	var payload viewerFrame
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("decode payload %q: %v", got, err)
	}
	if payload.SocketID != tok.SocketID {
		t.Fatalf("socketID = %q want %q", payload.SocketID, tok.SocketID)
	}
	if string(payload.State) != `{"indexh":2,"indexv":0}` {
		t.Fatalf("state = %s", payload.State)
	}
}

func TestPresenterRejectsWrongSecret(t *testing.T) {
	h := New()
	tok, _ := h.IssueToken()

	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/multiplex/presenter"
	ws, err := websocket.Dial(wsURL, "", srv.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/multiplex/"+tok.SocketID+"/events", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)
	_, _ = readEvent(br)
	time.Sleep(20 * time.Millisecond)

	bad := presenterFrame{
		Secret:   "not-the-real-secret",
		SocketID: tok.SocketID,
		State:    json.RawMessage(`{"indexh":1}`),
	}
	if err := websocket.JSON.Send(ws, bad); err != nil {
		t.Fatal(err)
	}

	// No frame should arrive within the deadline.
	type result struct {
		s   string
		err error
	}
	done := make(chan result, 1)
	go func() {
		s, err := readEvent(br)
		done <- result{s, err}
	}()
	select {
	case r := <-done:
		t.Fatalf("unexpected event %q (err=%v)", r.s, r.err)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestUnsubscribeOnViewerDisconnect(t *testing.T) {
	h := New()
	tok, _ := h.IssueToken()

	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/multiplex/"+tok.SocketID+"/events", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(resp.Body)
	_, _ = readEvent(br)

	h.mu.RLock()
	got := len(h.subs[tok.SocketID])
	h.mu.RUnlock()
	if got != 1 {
		t.Fatalf("subs after connect = %d, want 1", got)
	}

	resp.Body.Close()

	// Allow the SSE handler goroutine to notice disconnect and clean up.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		h.mu.RLock()
		got = len(h.subs[tok.SocketID])
		h.mu.RUnlock()
		if got == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("subs after disconnect = %d, want 0", got)
}

// readEvent reads bytes until a blank-line terminator, skipping
// SSE comment lines (those starting with ':'). Returns the joined
// data: payload.
func readEvent(br *bufio.Reader) (string, error) {
	var data strings.Builder
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF && data.Len() > 0 {
				return data.String(), nil
			}
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if data.Len() == 0 {
				continue
			}
			return data.String(), nil
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data.WriteString(strings.TrimPrefix(line, "data: "))
		}
	}
}
