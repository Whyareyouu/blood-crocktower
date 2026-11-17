// Package access содержит единую серверную политику доступа HTTP и WebSocket.
package access

import (
	"context"
	"errors"

	"blood/apps/api/internal/auth"
	"blood/apps/api/internal/contracts"
	"gorm.io/gorm"
)

var ErrDenied = errors.New("нет доступа")

// Эти записи задают членство и полномочия. Управление комнатами добавляется
// на этапе 3; идентификаторы владельца и Рассказчика клиент изменить не может.
type Room struct {
	ID            string           `gorm:"primaryKey;size:128"`
	Name          string           `gorm:"size:80;not null"`
	CreatorID     string           `gorm:"size:64;not null;index"`
	Creator       auth.User        `gorm:"constraint:OnDelete:RESTRICT"`
	StorytellerID *string          `gorm:"size:64"`
	Storyteller   *auth.User       `gorm:"constraint:OnDelete:RESTRICT"`
	Mode          contracts.Mode   `gorm:"size:32;not null"`
	Script        contracts.Script `gorm:"size:32;not null"`
	Capacity      int
	State         contracts.RoomState `gorm:"size:16;not null"`
}

type Membership struct {
	RoomID string    `gorm:"primaryKey;size:128"`
	Room   Room      `gorm:"constraint:OnDelete:CASCADE"`
	UserID string    `gorm:"primaryKey;size:64"`
	User   auth.User `gorm:"constraint:OnDelete:CASCADE"`
	// У Рассказчика нет игровой роли. Создатель в программном режиме — игрок.
	Player bool `gorm:"not null"`
}

type Conversation struct {
	ID     string `gorm:"primaryKey;size:128"`
	RoomID string `gorm:"size:128;not null"`
	Room   Room   `gorm:"constraint:OnDelete:CASCADE"`
}

type ConversationMember struct {
	ConversationID string       `gorm:"primaryKey;size:128"`
	Conversation   Conversation `gorm:"constraint:OnDelete:CASCADE"`
	UserID         string       `gorm:"primaryKey;size:64"`
	User           auth.User    `gorm:"constraint:OnDelete:CASCADE"`
}

type Permission string

const (
	Participate  Permission = "participate"
	ManageRoom   Permission = "manage_room"
	ReadGrimoire Permission = "read_grimoire"
)

type Policy struct{ db *gorm.DB }

func New(db *gorm.DB) *Policy { return &Policy{db: db} }

func (p *Policy) Room(ctx context.Context, userID, roomID string, permission Permission) error {
	// GORM пропускает нулевые поля struct в Where: пустой ID не должен
	// превращать проверку конкретного ресурса в поиск любого членства.
	if userID == "" || roomID == "" {
		return ErrDenied
	}
	var member Membership
	err := p.db.WithContext(ctx).Preload("Room").Where(&Membership{RoomID: roomID, UserID: userID}).Take(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrDenied
	}
	if err != nil {
		return err
	}
	switch permission {
	case Participate:
		return nil
	case ManageRoom:
		if member.Room.CreatorID == userID {
			return nil
		}
	case ReadGrimoire:
		if member.Room.Mode == contracts.ModeHumanStoryteller && member.Room.StorytellerID != nil && *member.Room.StorytellerID == userID && !member.Player {
			return nil
		}
	}
	return ErrDenied
}

func (p *Policy) Conversation(ctx context.Context, userID, conversationID string) error {
	if userID == "" || conversationID == "" {
		return ErrDenied
	}
	var member ConversationMember
	err := p.db.WithContext(ctx).Preload("Conversation").Where(&ConversationMember{ConversationID: conversationID, UserID: userID}).Take(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrDenied
	}
	if err != nil {
		return err
	}
	return p.Room(ctx, userID, member.Conversation.RoomID, Participate)
}

// MyRooms восстанавливает только собственное членство после входа на любом
// устройстве. Никакие сведения о ролях и сообщениях в DTO не попадают.
func (p *Policy) MyRooms(ctx context.Context, userID string) (contracts.RoomList, error) {
	if userID == "" {
		return contracts.RoomList{Rooms: []contracts.RoomSummary{}}, ErrDenied
	}
	var memberships []Membership
	err := p.db.WithContext(ctx).Preload("Room.Creator").Where(&Membership{UserID: userID}).Order("room_id").Find(&memberships).Error
	result := contracts.RoomList{Rooms: []contracts.RoomSummary{}}
	if err != nil {
		return result, err
	}
	for _, m := range memberships {
		var count int64
		if err := p.db.WithContext(ctx).Model(&Membership{}).Where(&Membership{RoomID: m.RoomID}).Where(map[string]any{"player": true}).Count(&count).Error; err != nil {
			return result, err
		}
		r := m.Room
		result.Rooms = append(result.Rooms, contracts.RoomSummary{Id: r.ID, Name: r.Name, Script: r.Script, Mode: r.Mode,
			Creator: contracts.User{Id: r.Creator.ID, Login: r.Creator.Login}, Capacity: r.Capacity, PlayerCount: int(count), State: r.State,
			Joinable: r.State == contracts.RoomStateWaiting && count < int64(r.Capacity)})
	}
	return result, nil
}
