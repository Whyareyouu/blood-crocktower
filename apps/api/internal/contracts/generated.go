// Сгенерировано из docs/contracts/protocol.schema.json. Не редактировать вручную.
package contracts

type Health struct {
	Status string `json:"status"`
}

type Status struct {
	Service string `json:"service"`
	Stage   string `json:"stage"`
}

type ErrorCode string

const (
	ErrorCodeInvalidRequest     ErrorCode = "invalid_request"
	ErrorCodeUnauthenticated    ErrorCode = "unauthenticated"
	ErrorCodeForbidden          ErrorCode = "forbidden"
	ErrorCodeNotFound           ErrorCode = "not_found"
	ErrorCodeConflict           ErrorCode = "conflict"
	ErrorCodeRateLimited        ErrorCode = "rate_limited"
	ErrorCodeUnsupportedVersion ErrorCode = "unsupported_version"
	ErrorCodeInternalError      ErrorCode = "internal_error"
)

type APIError struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	RequestID *string   `json:"requestId,omitempty"`
}

type User struct {
	Id    string `json:"id"`
	Login string `json:"login"`
}

type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Session struct {
	User      User   `json:"user"`
	CsrfToken string `json:"csrfToken"`
}

type Script string

const (
	ScriptTroubleBrewing  Script = "trouble_brewing"
	ScriptBadMoonRising   Script = "bad_moon_rising"
	ScriptSectsAndViolets Script = "sects_and_violets"
)

type Mode string

const (
	ModeHumanStoryteller   Mode = "human_storyteller"
	ModeProgramStoryteller Mode = "program_storyteller"
)

type RoomState string

const (
	RoomStateWaiting  RoomState = "waiting"
	RoomStateRunning  RoomState = "running"
	RoomStatePaused   RoomState = "paused"
	RoomStateFinished RoomState = "finished"
)

type CreateRoom struct {
	Name     string `json:"name"`
	Script   Script `json:"script"`
	Mode     Mode   `json:"mode"`
	Capacity int    `json:"capacity"`
}

type RoomSummary struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Script      Script    `json:"script"`
	Mode        Mode      `json:"mode"`
	Creator     User      `json:"creator"`
	Capacity    int       `json:"capacity"`
	PlayerCount int       `json:"playerCount"`
	State       RoomState `json:"state"`
	Joinable    bool      `json:"joinable"`
}

type RoomList struct {
	Rooms []RoomSummary `json:"rooms"`
}

type Participant struct {
	User          User    `json:"user"`
	Seat          *int    `json:"seat,omitempty"`
	Ready         bool    `json:"ready"`
	Connected     bool    `json:"connected"`
	PublicAlive   bool    `json:"publicAlive"`
	TravellerRole *string `json:"travellerRole,omitempty"`
}

type RoomView struct {
	Room          RoomSummary   `json:"room"`
	Participants  []Participant `json:"participants"`
	StorytellerID *string       `json:"storytellerId,omitempty"`
}

type SelfView struct {
	UserID         string   `json:"userId"`
	ShownRoleID    *string  `json:"shownRoleId,omitempty"`
	ShownAlignment *string  `json:"shownAlignment,omitempty"`
	Information    []string `json:"information"`
}

type Message struct {
	Id     string `json:"id"`
	Sender User   `json:"sender"`
	Text   string `json:"text"`
	SentAt string `json:"sentAt"`
}

type SendMessage struct {
	Text string `json:"text"`
}

type MessageList struct {
	Messages   []Message `json:"messages"`
	NextCursor *string   `json:"nextCursor,omitempty"`
}

type Conversation struct {
	Id      string `json:"id"`
	RoomID  string `json:"roomId"`
	Members []User `json:"members"`
}

type ConversationList struct {
	Conversations []Conversation `json:"conversations"`
}

type CreateConversation struct {
	ParticipantID string `json:"participantId"`
}

type SubscribeRoomsPayload struct {
}

type SubscribeRoomPayload struct {
	RoomID string `json:"roomId"`
}

type SubscribeConversationPayload struct {
	ConversationID string `json:"conversationId"`
}

type SetReadyPayload struct {
	RoomID string `json:"roomId"`
	Ready  bool   `json:"ready"`
}

type SetSeatPayload struct {
	RoomID string `json:"roomId"`
	Seat   int    `json:"seat"`
}

type SendRoomMessagePayload struct {
	RoomID string `json:"roomId"`
	Text   string `json:"text"`
}

type SendPrivateMessagePayload struct {
	ConversationID string `json:"conversationId"`
	Text           string `json:"text"`
}

type PingPayload struct {
}

type AckPayload struct {
}

type RoomSnapshotPayload struct {
	RoomID       string   `json:"roomId"`
	ViewRevision string   `json:"viewRevision"`
	View         RoomView `json:"view"`
}

type SelfSnapshotPayload struct {
	RoomID       string   `json:"roomId"`
	ViewRevision string   `json:"viewRevision"`
	View         SelfView `json:"view"`
}

type RoomsSnapshotPayload struct {
	ViewRevision string   `json:"viewRevision"`
	View         RoomList `json:"view"`
}

type ConversationSnapshotPayload struct {
	ConversationID string      `json:"conversationId"`
	ViewRevision   string      `json:"viewRevision"`
	View           MessageList `json:"view"`
}

type PongPayload struct {
	ServerTime string `json:"serverTime"`
}

type SubscribeRooms struct {
	Version   int                   `json:"version"`
	Type      string                `json:"type"`
	RequestID string                `json:"requestId"`
	Payload   SubscribeRoomsPayload `json:"payload"`
}

const SubscribeRoomsType = "rooms.subscribe"

type SubscribeRoom struct {
	Version   int                  `json:"version"`
	Type      string               `json:"type"`
	RequestID string               `json:"requestId"`
	Payload   SubscribeRoomPayload `json:"payload"`
}

const SubscribeRoomType = "room.subscribe"

type SubscribeConversation struct {
	Version   int                          `json:"version"`
	Type      string                       `json:"type"`
	RequestID string                       `json:"requestId"`
	Payload   SubscribeConversationPayload `json:"payload"`
}

const SubscribeConversationType = "conversation.subscribe"

type SetReady struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	RequestID string          `json:"requestId"`
	Payload   SetReadyPayload `json:"payload"`
}

const SetReadyType = "room.ready"

type SetSeat struct {
	Version   int            `json:"version"`
	Type      string         `json:"type"`
	RequestID string         `json:"requestId"`
	Payload   SetSeatPayload `json:"payload"`
}

const SetSeatType = "room.seat"

type SendRoomMessage struct {
	Version   int                    `json:"version"`
	Type      string                 `json:"type"`
	RequestID string                 `json:"requestId"`
	Payload   SendRoomMessagePayload `json:"payload"`
}

const SendRoomMessageType = "room.message.send"

type SendPrivateMessage struct {
	Version   int                       `json:"version"`
	Type      string                    `json:"type"`
	RequestID string                    `json:"requestId"`
	Payload   SendPrivateMessagePayload `json:"payload"`
}

const SendPrivateMessageType = "conversation.message.send"

type Ping struct {
	Version   int         `json:"version"`
	Type      string      `json:"type"`
	RequestID string      `json:"requestId"`
	Payload   PingPayload `json:"payload"`
}

const PingType = "ping"

type Ack struct {
	Version   int        `json:"version"`
	Type      string     `json:"type"`
	RequestID string     `json:"requestId"`
	Payload   AckPayload `json:"payload"`
}

const AckType = "ack"

type Error struct {
	Version   int      `json:"version"`
	Type      string   `json:"type"`
	RequestID *string  `json:"requestId,omitempty"`
	Payload   APIError `json:"payload"`
}

const ErrorType = "error"

type RoomsSnapshot struct {
	Version int                  `json:"version"`
	Type    string               `json:"type"`
	Payload RoomsSnapshotPayload `json:"payload"`
}

const RoomsSnapshotType = "rooms.snapshot"

type RoomSnapshot struct {
	Version int                 `json:"version"`
	Type    string              `json:"type"`
	Payload RoomSnapshotPayload `json:"payload"`
}

const RoomSnapshotType = "room.snapshot"

type SelfSnapshot struct {
	Version int                 `json:"version"`
	Type    string              `json:"type"`
	Payload SelfSnapshotPayload `json:"payload"`
}

const SelfSnapshotType = "self.snapshot"

type ConversationSnapshot struct {
	Version int                         `json:"version"`
	Type    string                      `json:"type"`
	Payload ConversationSnapshotPayload `json:"payload"`
}

const ConversationSnapshotType = "conversation.snapshot"

type Pong struct {
	Version int         `json:"version"`
	Type    string      `json:"type"`
	Payload PongPayload `json:"payload"`
}

const PongType = "pong"

// ClientMessage объединяет допустимые направления сообщения. Декодирование и проверка констант обязательны в будущем транспорте.
type ClientMessage interface{ isClientMessage() }

func (SubscribeRooms) isClientMessage()        {}
func (SubscribeRoom) isClientMessage()         {}
func (SubscribeConversation) isClientMessage() {}
func (SetReady) isClientMessage()              {}
func (SetSeat) isClientMessage()               {}
func (SendRoomMessage) isClientMessage()       {}
func (SendPrivateMessage) isClientMessage()    {}
func (Ping) isClientMessage()                  {}

// ServerMessage объединяет допустимые направления сообщения. Декодирование и проверка констант обязательны в будущем транспорте.
type ServerMessage interface{ isServerMessage() }

func (Ack) isServerMessage()                  {}
func (Error) isServerMessage()                {}
func (RoomsSnapshot) isServerMessage()        {}
func (RoomSnapshot) isServerMessage()         {}
func (SelfSnapshot) isServerMessage()         {}
func (ConversationSnapshot) isServerMessage() {}
func (Pong) isServerMessage()                 {}

type CSRFToken struct {
	CsrfToken string `json:"csrfToken"`
}
