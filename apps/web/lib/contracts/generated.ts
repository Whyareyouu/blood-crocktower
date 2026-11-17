// Сгенерировано из docs/contracts/protocol.schema.json. Не редактировать вручную.
export interface Health {
  status: "ok" | "not_ready";
}

export interface Status {
  service: "blood-api";
  stage: "accounts";
}

export type ErrorCode = "invalid_request" | "unauthenticated" | "forbidden" | "not_found" | "conflict" | "rate_limited" | "unsupported_version" | "internal_error";

export interface APIError {
  code: ErrorCode;
  message: string;
  requestId?: string;
}

export interface User {
  id: string;
  login: string;
}

export interface Credentials {
  login: string;
  password: string;
}

export interface Session {
  user: User;
  csrfToken: string;
}

export type Script = "trouble_brewing" | "bad_moon_rising" | "sects_and_violets";

export type Mode = "human_storyteller" | "program_storyteller";

export type RoomState = "waiting" | "running" | "paused" | "finished";

export interface CreateRoom {
  name: string;
  script: Script;
  mode: Mode;
  capacity: number;
}

export interface RoomSummary {
  id: string;
  name: string;
  script: Script;
  mode: Mode;
  creator: User;
  capacity: number;
  playerCount: number;
  state: RoomState;
  joinable: boolean;
}

export interface RoomList {
  rooms: Array<RoomSummary>;
}

export interface Participant {
  user: User;
  seat?: number;
  ready: boolean;
  connected: boolean;
  publicAlive: boolean;
  travellerRole?: string;
}

export interface RoomView {
  room: RoomSummary;
  participants: Array<Participant>;
  storytellerId?: string;
}

export interface SelfView {
  userId: string;
  shownRoleId?: string;
  shownAlignment?: "good" | "evil";
  information: Array<string>;
}

export interface Message {
  id: string;
  sender: User;
  text: string;
  sentAt: string;
}

export interface SendMessage {
  text: string;
}

export interface MessageList {
  messages: Array<Message>;
  nextCursor?: string;
}

export interface Conversation {
  id: string;
  roomId: string;
  members: Array<User>;
}

export interface ConversationList {
  conversations: Array<Conversation>;
}

export interface CreateConversation {
  participantId: string;
}

export type SubscribeRoomsPayload = Record<string, never>;

export interface SubscribeRoomPayload {
  roomId: string;
}

export interface SubscribeConversationPayload {
  conversationId: string;
}

export interface SetReadyPayload {
  roomId: string;
  ready: boolean;
}

export interface SetSeatPayload {
  roomId: string;
  seat: number;
}

export interface SendRoomMessagePayload {
  roomId: string;
  text: string;
}

export interface SendPrivateMessagePayload {
  conversationId: string;
  text: string;
}

export type PingPayload = Record<string, never>;

export type AckPayload = Record<string, never>;

export interface RoomSnapshotPayload {
  roomId: string;
  viewRevision: string;
  view: RoomView;
}

export interface SelfSnapshotPayload {
  roomId: string;
  viewRevision: string;
  view: SelfView;
}

export interface RoomsSnapshotPayload {
  viewRevision: string;
  view: RoomList;
}

export interface ConversationSnapshotPayload {
  conversationId: string;
  viewRevision: string;
  view: MessageList;
}

export interface PongPayload {
  serverTime: string;
}

export interface SubscribeRooms {
  version: 1;
  type: "rooms.subscribe";
  requestId: string;
  payload: SubscribeRoomsPayload;
}

export interface SubscribeRoom {
  version: 1;
  type: "room.subscribe";
  requestId: string;
  payload: SubscribeRoomPayload;
}

export interface SubscribeConversation {
  version: 1;
  type: "conversation.subscribe";
  requestId: string;
  payload: SubscribeConversationPayload;
}

export interface SetReady {
  version: 1;
  type: "room.ready";
  requestId: string;
  payload: SetReadyPayload;
}

export interface SetSeat {
  version: 1;
  type: "room.seat";
  requestId: string;
  payload: SetSeatPayload;
}

export interface SendRoomMessage {
  version: 1;
  type: "room.message.send";
  requestId: string;
  payload: SendRoomMessagePayload;
}

export interface SendPrivateMessage {
  version: 1;
  type: "conversation.message.send";
  requestId: string;
  payload: SendPrivateMessagePayload;
}

export interface Ping {
  version: 1;
  type: "ping";
  requestId: string;
  payload: PingPayload;
}

export interface Ack {
  version: 1;
  type: "ack";
  requestId: string;
  payload: AckPayload;
}

export interface Error {
  version: 1;
  type: "error";
  requestId?: string;
  payload: APIError;
}

export interface RoomsSnapshot {
  version: 1;
  type: "rooms.snapshot";
  payload: RoomsSnapshotPayload;
}

export interface RoomSnapshot {
  version: 1;
  type: "room.snapshot";
  payload: RoomSnapshotPayload;
}

export interface SelfSnapshot {
  version: 1;
  type: "self.snapshot";
  payload: SelfSnapshotPayload;
}

export interface ConversationSnapshot {
  version: 1;
  type: "conversation.snapshot";
  payload: ConversationSnapshotPayload;
}

export interface Pong {
  version: 1;
  type: "pong";
  payload: PongPayload;
}

export type ClientMessage = SubscribeRooms | SubscribeRoom | SubscribeConversation | SetReady | SetSeat | SendRoomMessage | SendPrivateMessage | Ping;

export type ServerMessage = Ack | Error | RoomsSnapshot | RoomSnapshot | SelfSnapshot | ConversationSnapshot | Pong;

export interface CSRFToken {
  csrfToken: string;
}
