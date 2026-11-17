package access_test

import (
	"context"
	"errors"
	"testing"

	"blood/apps/api/internal/access"
	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/contracts"
	"blood/apps/api/internal/testutil"
)

func TestRoomAndConversationPermissions(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	for _, id := range []string{"creator", "player", "storyteller", "outsider"} {
		if err := db.Create(&auth.User{ID: id, Login: id, PasswordHash: "fixture"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	storyteller := "storyteller"
	room := access.Room{ID: "room", Name: "Комната", CreatorID: "creator", StorytellerID: &storyteller, Mode: contracts.ModeHumanStoryteller, Script: contracts.ScriptTroubleBrewing, Capacity: 10, State: contracts.RoomStateWaiting}
	if err := db.Create(&room).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"creator", "player", "storyteller"} {
		if err := db.Create(&access.Membership{RoomID: room.ID, UserID: id, Player: id != "storyteller"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	policy := access.New(db)
	if err := policy.Room(ctx, "player", "", access.Participate); !errors.Is(err, access.ErrDenied) {
		t.Fatal("пустой ID комнаты дал доступ к произвольному членству")
	}
	if err := policy.Room(ctx, "", room.ID, access.Participate); !errors.Is(err, access.ErrDenied) {
		t.Fatal("пустой ID пользователя дал доступ")
	}
	if _, err := policy.MyRooms(ctx, ""); !errors.Is(err, access.ErrDenied) {
		t.Fatal("пустой ID пользователя раскрыл комнаты")
	}
	for _, tc := range []struct {
		user       string
		permission access.Permission
		allowed    bool
	}{
		{"creator", access.Participate, true}, {"creator", access.ManageRoom, true}, {"creator", access.ReadGrimoire, false},
		{"player", access.Participate, true}, {"player", access.ManageRoom, false}, {"player", access.ReadGrimoire, false},
		{"storyteller", access.Participate, true}, {"storyteller", access.ManageRoom, false}, {"storyteller", access.ReadGrimoire, true},
		{"outsider", access.Participate, false}, {"outsider", access.ManageRoom, false}, {"outsider", access.ReadGrimoire, false},
	} {
		err := policy.Room(ctx, tc.user, room.ID, tc.permission)
		if tc.allowed && err != nil || !tc.allowed && !errors.Is(err, access.ErrDenied) {
			t.Fatalf("%s %s: %v", tc.user, tc.permission, err)
		}
	}
	if err := db.Create(&access.Conversation{ID: "private", RoomID: room.ID}).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"player", "storyteller"} {
		if err := db.Create(&access.ConversationMember{ConversationID: "private", UserID: id}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"creator", "outsider"} {
		if err := policy.Conversation(ctx, id, "private"); !errors.Is(err, access.ErrDenied) {
			t.Fatal("создатель или посторонний получил чужую переписку")
		}
	}
	if err := policy.Conversation(ctx, "player", "private"); err != nil {
		t.Fatal(err)
	}
	if err := policy.Conversation(ctx, "player", ""); !errors.Is(err, access.ErrDenied) {
		t.Fatal("пустой ID беседы дал доступ к произвольной переписке")
	}
	if err := db.Model(&access.Membership{}).Where(&access.Membership{RoomID: room.ID, UserID: storyteller}).Update("player", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := policy.Room(ctx, storyteller, room.ID, access.ReadGrimoire); !errors.Is(err, access.ErrDenied) {
		t.Fatal("играющий Рассказчик получил Гримуар")
	}
	if err := db.Model(&access.Room{}).Where(&access.Room{ID: room.ID}).Update("mode", contracts.ModeProgramStoryteller).Error; err != nil {
		t.Fatal(err)
	}
	if err := policy.Room(ctx, storyteller, room.ID, access.ReadGrimoire); !errors.Is(err, access.ErrDenied) {
		t.Fatal("программный режим открыл Гримуар человеку")
	}
	rooms, err := policy.MyRooms(ctx, "player")
	if err != nil || len(rooms.Rooms) != 1 || rooms.Rooms[0].Id != room.ID {
		t.Fatal("собственное членство не восстановлено")
	}
	rooms, err = policy.MyRooms(ctx, "outsider")
	if err != nil || len(rooms.Rooms) != 0 {
		t.Fatal("раскрыто чужое членство")
	}
	if err := db.Where(&access.Membership{RoomID: room.ID, UserID: "player"}).Delete(&access.Membership{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := policy.Conversation(ctx, "player", "private"); !errors.Is(err, access.ErrDenied) {
		t.Fatal("бывший участник сохранил доступ")
	}
}
