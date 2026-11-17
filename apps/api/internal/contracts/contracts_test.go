package contracts_test

import (
	"blood/apps/api/internal/contracts"
	"blood/apps/api/internal/httpapi"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestImplementedHTTPMatchesDTO(t *testing.T) {
	for _, tc := range []struct {
		path       string
		readyError error
		status     int
		target     any
	}{
		{"/health/live", nil, 200, &contracts.Health{}},
		{"/health/ready", nil, 200, &contracts.Health{}},
		{"/health/ready", errors.New("секрет подключения"), 503, &contracts.Health{}},
		{"/api/v1/status", nil, 200, &contracts.Status{}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			httpapi.New(func(context.Context) error { return tc.readyError }).ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != tc.status {
				t.Fatalf("неожиданный HTTP status %d", response.Code)
			}
			decoder := json.NewDecoder(response.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(tc.target); err != nil {
				t.Fatal(err)
			}
			switch value := tc.target.(type) {
			case *contracts.Health:
				expected := "ok"
				if tc.status == 503 {
					expected = "not_ready"
				}
				if value.Status != expected {
					t.Fatalf("неожиданный status %s", value.Status)
				}
			case *contracts.Status:
				if value.Service != "blood-api" || value.Stage != "bootstrap" {
					t.Fatalf("неожиданный статус API: %#v", value)
				}
			}
		})
	}
}

func TestWireFixturesRoundTrip(t *testing.T) {
	data, err := os.ReadFile("../../../../docs/contracts/fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name   string          `json:"name"`
		Schema string          `json:"schema"`
		Value  json.RawMessage `json:"value"`
		Valid  bool            `json:"valid"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	clientTypes := map[string]func() any{
		contracts.SubscribeRoomsType:        func() any { return &contracts.SubscribeRooms{} },
		contracts.SubscribeRoomType:         func() any { return &contracts.SubscribeRoom{} },
		contracts.SubscribeConversationType: func() any { return &contracts.SubscribeConversation{} },
		contracts.SetReadyType:              func() any { return &contracts.SetReady{} },
		contracts.SetSeatType:               func() any { return &contracts.SetSeat{} },
		contracts.SendRoomMessageType:       func() any { return &contracts.SendRoomMessage{} },
		contracts.SendPrivateMessageType:    func() any { return &contracts.SendPrivateMessage{} },
		contracts.PingType:                  func() any { return &contracts.Ping{} },
	}
	serverTypes := map[string]func() any{
		contracts.AckType:                  func() any { return &contracts.Ack{} },
		contracts.ErrorType:                func() any { return &contracts.Error{} },
		contracts.RoomsSnapshotType:        func() any { return &contracts.RoomsSnapshot{} },
		contracts.RoomSnapshotType:         func() any { return &contracts.RoomSnapshot{} },
		contracts.SelfSnapshotType:         func() any { return &contracts.SelfSnapshot{} },
		contracts.ConversationSnapshotType: func() any { return &contracts.ConversationSnapshot{} },
		contracts.PongType:                 func() any { return &contracts.Pong{} },
	}
	for _, fixture := range fixtures {
		if !fixture.Valid {
			continue // Полную JSON Schema, включая границы и константы, проверяет Ajv.
		}
		t.Run(fixture.Name, func(t *testing.T) {
			var target any
			switch fixture.Schema {
			case "Health":
				target = &contracts.Health{}
			case "Status":
				target = &contracts.Status{}
			case "Credentials":
				target = &contracts.Credentials{}
			case "RoomView":
				target = &contracts.RoomView{}
			case "ClientMessage", "ServerMessage":
				var tag struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal(fixture.Value, &tag); err != nil {
					t.Fatal(err)
				}
				factory := clientTypes[tag.Type]
				if fixture.Schema == "ServerMessage" {
					factory = serverTypes[tag.Type]
				}
				if factory == nil {
					t.Fatalf("нет Go DTO для %s", tag.Type)
				}
				target = factory()
			default:
				t.Fatalf("нет Go DTO для схемы %s", fixture.Schema)
			}
			decoder := json.NewDecoder(bytes.NewReader(fixture.Value))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(target); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(target)
			if err != nil {
				t.Fatal(err)
			}
			var before, after any
			if err := json.Unmarshal(fixture.Value, &before); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &after); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("Go изменяет wire JSON: %s", encoded)
			}
		})
	}
}

func TestRoomViewRejectsHiddenFields(t *testing.T) {
	for _, secret := range []string{"actualRoleId", "grimoire", "privateMessages"} {
		decoder := json.NewDecoder(bytes.NewBufferString(`{"` + secret + `":{}}`))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&contracts.RoomView{}); err == nil {
			t.Fatalf("публичная проекция принимает %s", secret)
		}
	}
}
