package multiplex

import (
	"log/slog"
	"net/http"

	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"golang.org/x/crypto/bcrypt"
)

type MultiplexData struct {
	Secret string `json:"secret,omitempty"`
	SocketID string `json:"socketId"`
	State map[string]interface{} `json:"state"`
}

func SocketIOSetup() *socketio.Server {
	var allowOriginFunc = func(r *http.Request) bool {
		return true
	}

	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: allowOriginFunc,
			},
			&websocket.Transport{
				CheckOrigin: allowOriginFunc,
			},
		},
	})

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		slog.Info("connected", "socket id", s.ID())

		return nil
	})

	server.OnEvent("/", "multiplex-statechanged", func(s socketio.Conn, data MultiplexData) {
		if err := bcrypt.CompareHashAndPassword([]byte(data.SocketID), []byte(data.Secret)); err != nil {
			return
		}

		data.Secret = ""
		server.BroadcastToNamespace("/", data.SocketID, data)
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		slog.Info("socket error", "error", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		slog.Info("closed", "reason", reason)
	})

	return server
}
