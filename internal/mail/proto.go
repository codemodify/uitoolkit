package mail

import "encoding/json"

// JSON-RPC 2.0 over a Unix socket, one JSON object per line (NDJSON).
// mailclientd is the server; mailclientui is a client. Notifications
// (events) have no "id".
//
// Methods:
//
//	ping
//	status.get
//	accounts.list
//	folders.list        {accountId}
//	folders.get         {id}
//	folders.create      {accountId, name, parent?}
//	messages.list       {folderId, filter?}
//	messages.get        {id}
//	messages.search     {accountId?, folderId?, filter}
//	messages.setFlags   {id, patch}
//	messages.move       {ids, dest}
//	messages.delete     {ids}
//	messages.append     {folderId, message}
//	messages.update     {id, message}
//	messages.fetch      {accountId}     // Get Messages
//	unread.get          {folderId?}     // omit folderId → total
//	compose.send        {accountId, message}
//	compose.saveDraft   {accountId, message, id?}
//
// Events (server → client, no id):
//
//	mail.changed        {folderId, reason}
//	mail.fetched        {accountId, count}

const RPCVersion = "2.0"

// Method names.
const (
	MethodPing           = "ping"
	MethodStatusGet      = "status.get"
	MethodAccountsList   = "accounts.list"
	MethodFoldersList    = "folders.list"
	MethodFoldersGet     = "folders.get"
	MethodFoldersCreate  = "folders.create"
	MethodMessagesList   = "messages.list"
	MethodMessagesGet    = "messages.get"
	MethodMessagesSearch = "messages.search"
	MethodMessagesFlags  = "messages.setFlags"
	MethodMessagesMove   = "messages.move"
	MethodMessagesDelete = "messages.delete"
	MethodMessagesAppend = "messages.append"
	MethodMessagesUpdate = "messages.update"
	MethodMessagesFetch  = "messages.fetch"
	MethodUnreadGet      = "unread.get"
	MethodComposeSend    = "compose.send"
	MethodComposeDraft   = "compose.saveDraft"

	EventChanged = "mail.changed"
	EventFetched = "mail.fetched"
)

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Event is a server notification (no id).
type Event struct {
	Method string
	Reason string
	Folder FolderID
	Count  int
}

// Wire params.

type folderListParams struct {
	AccountID string `json:"accountId"`
}

type folderGetParams struct {
	ID FolderID `json:"id"`
}

type folderCreateParams struct {
	AccountID string   `json:"accountId"`
	Name      string   `json:"name"`
	Parent    FolderID `json:"parent,omitempty"`
}

type messagesListParams struct {
	FolderID FolderID `json:"folderId"`
	Filter   *Filter  `json:"filter,omitempty"`
}

type messageIDParams struct {
	ID MessageID `json:"id"`
}

type searchParams struct {
	AccountID string   `json:"accountId,omitempty"`
	FolderID  FolderID `json:"folderId,omitempty"`
	Filter    Filter   `json:"filter"`
}

type setFlagsParams struct {
	ID    MessageID    `json:"id"`
	Patch wireFlagPatch `json:"patch"`
}

type wireFlagPatch struct {
	Read    *bool     `json:"read,omitempty"`
	Starred *bool     `json:"starred,omitempty"`
	Tags    *[]string `json:"tags,omitempty"`
}

func (w wireFlagPatch) to() FlagPatch {
	return FlagPatch{Read: w.Read, Starred: w.Starred, Tags: w.Tags}
}

func patchToWire(p FlagPatch) wireFlagPatch {
	return wireFlagPatch{Read: p.Read, Starred: p.Starred, Tags: p.Tags}
}

type moveParams struct {
	IDs  []MessageID `json:"ids"`
	Dest FolderID    `json:"dest"`
}

type deleteParams struct {
	IDs []MessageID `json:"ids"`
}

type appendParams struct {
	FolderID FolderID `json:"folderId"`
	Message  Message  `json:"message"`
}

type updateParams struct {
	ID      MessageID `json:"id"`
	Message Message   `json:"message"`
}

type fetchParams struct {
	AccountID string `json:"accountId"`
}

type unreadParams struct {
	FolderID FolderID `json:"folderId,omitempty"`
}

type composeParams struct {
	AccountID string    `json:"accountId"`
	Message   Message   `json:"message"`
	ID        MessageID `json:"id,omitempty"`
}

type unreadResult struct {
	Count int `json:"count"`
}

type fetchResult struct {
	Count int `json:"count"`
}

type appendResult struct {
	ID MessageID `json:"id"`
}

type eventParams struct {
	FolderID  FolderID `json:"folderId,omitempty"`
	Reason    string   `json:"reason,omitempty"`
	AccountID string   `json:"accountId,omitempty"`
	Count     int      `json:"count,omitempty"`
}

func decodeParams[T any](raw json.RawMessage) (T, error) {
	var v T
	if len(raw) == 0 {
		return v, nil
	}
	err := json.Unmarshal(raw, &v)
	return v, err
}
