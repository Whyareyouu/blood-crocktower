package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"blood/apps/api/internal/access"
	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/contracts"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type socketOwner struct{ hash, userID string }
type socketHub struct {
	mu          sync.Mutex
	connections map[*websocket.Conn]socketOwner
}

func (h *socketHub) add(c *websocket.Conn, id auth.Identity) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.connections == nil {
		h.connections = make(map[*websocket.Conn]socketOwner)
	}
	count := 0
	for _, owner := range h.connections {
		if owner.userID == id.User.Id {
			count++
		}
	}
	if count >= 4 || len(h.connections) >= 256 {
		return false
	}
	h.connections[c] = socketOwner{id.TokenHash, id.User.Id}
	return true
}

func (h *socketHub) remove(c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections, c)
}
func (h *socketHub) revoke(hash string) {
	h.mu.Lock()
	var connections []*websocket.Conn
	for c, owner := range h.connections {
		if owner.hash == hash {
			connections = append(connections, c)
		}
	}
	h.mu.Unlock()
	for _, c := range connections {
		go func() { _ = c.Close(4401, "Сессия завершена") }()
	}
}

func (h *socketHub) closeAll() {
	h.mu.Lock()
	var connections []*websocket.Conn
	for c := range h.connections {
		connections = append(connections, c)
	}
	h.mu.Unlock()
	for _, c := range connections {
		go func() { _ = c.Close(1001, "Сервер остановлен") }()
	}
}

func (s *server) websocket(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != s.options.PublicOrigin || r.URL.RawQuery != "" {
		apiError(w, 403, contracts.ErrorCodeForbidden, "Недопустимый источник соединения.")
		return
	}
	protocolOK := false
	for _, protocol := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		if strings.TrimSpace(protocol) == "blood.v1" {
			protocolOK = true
		}
	}
	if !protocolOK {
		apiError(w, 426, contracts.ErrorCodeUnsupportedVersion, "Требуется подпротокол blood.v1.")
		return
	}
	id, ok := s.identity(w, r)
	if !ok {
		return
	}
	if !s.limits.allow("ws-connect:"+id.User.Id, time.Second, 10) {
		limited(w)
		return
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"blood.v1"}, OriginPatterns: []string{s.options.PublicOrigin}})
	if err != nil {
		return
	}
	defer c.CloseNow()
	if !s.sockets.add(c, id) {
		_ = c.Close(1008, "Слишком много соединений")
		return
	}
	defer s.sockets.remove(c)
	c.SetReadLimit(64 * 1024)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	token := cookieValue(r, sessionCookie)
	// Проверяет отзыв и срок даже у молчащего соединения и при другом API-процессе.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				check, stop := context.WithTimeout(ctx, 2*time.Second)
				_, err := s.options.Auth.Resolve(check, token)
				stop()
				if err != nil {
					_ = c.Close(4401, "Сессия завершена")
					cancel()
					return
				}
			}
		}
	}()
	for {
		readCtx, stop := context.WithTimeout(ctx, 60*time.Second)
		kind, data, err := c.Read(readCtx)
		stop()
		if err != nil {
			return
		}
		if kind != websocket.MessageText {
			_ = c.Close(1008, "Ожидается текстовый JSON")
			return
		}
		commandCtx, done := context.WithTimeout(ctx, 5*time.Second)
		identity, err := s.options.Auth.Resolve(commandCtx, token)
		if err != nil {
			done()
			_ = c.Close(4401, "Сессия завершена")
			return
		}
		if !s.limits.allow("ws-command:"+identity.User.Id, time.Second/5, 20) {
			done()
			_ = c.Close(1008, "Слишком много команд")
			return
		}
		var message struct {
			Version   int    `json:"version"`
			Type      string `json:"type"`
			RequestID string `json:"requestId"`
			Payload   struct {
				RoomID         string `json:"roomId"`
				ConversationID string `json:"conversationId"`
			} `json:"payload"`
		}
		if contracts.Validate("ClientMessage", data) != nil || json.Unmarshal(data, &message) != nil {
			err := wsjson.Write(commandCtx, c, contracts.Error{Version: 1, Type: "error", Payload: contracts.APIError{Code: contracts.ErrorCodeInvalidRequest, Message: "Некорректная команда."}})
			done()
			if err != nil {
				return
			}
			continue
		}
		var response any
		if message.Type == "ping" {
			response = contracts.Pong{Version: 1, Type: "pong", Payload: contracts.PongPayload{ServerTime: time.Now().UTC().Format(time.RFC3339Nano)}}
		} else {
			permissionErr := access.ErrDenied
			if s.options.Access != nil {
				switch message.Type {
				case "rooms.subscribe":
					permissionErr = nil
				case "room.subscribe", "room.ready", "room.seat", "room.message.send":
					permissionErr = s.options.Access.Room(commandCtx, identity.User.Id, message.Payload.RoomID, access.Participate)
				case "conversation.subscribe", "conversation.message.send":
					permissionErr = s.options.Access.Conversation(commandCtx, identity.User.Id, message.Payload.ConversationID)
				}
			}
			code, text := contracts.ErrorCodeInvalidRequest, "Операция будет доступна на этапе комнат и общения."
			if errors.Is(permissionErr, access.ErrDenied) {
				code, text = contracts.ErrorCodeForbidden, "Нет доступа к операции."
			} else if permissionErr != nil {
				code, text = contracts.ErrorCodeInternalError, "Сервис временно недоступен."
			}
			response = contracts.Error{Version: 1, Type: "error", RequestID: &message.RequestID, Payload: contracts.APIError{Code: code, Message: text}}
		}
		// Перед выдачей ответа повторно проверяем сессию; закрытие отзыва также
		// прерывает запись. Секретных проекций на этом этапе нет.
		_, err = s.options.Auth.Resolve(commandCtx, token)
		if err == nil {
			err = wsjson.Write(commandCtx, c, response)
		}
		done()
		if err != nil {
			return
		}
	}
}
