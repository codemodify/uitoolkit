package mail

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client is the mailclientui side of the Unix JSON-RPC. No IMAP here.
type Client struct {
	Socket string

	mu      sync.Mutex
	conn    net.Conn
	w       *bufio.Writer
	pending map[uint64]chan Response
	nextID  atomic.Uint64
	onEvent func(Event)
	closed  atomic.Bool
}

// Dial connects to mailclientd at socket.
func Dial(socket string) (*Client, error) {
	if socket == "" {
		socket = DefaultSocket()
	}
	c, err := net.DialTimeout("unix", socket, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("mailclientui: dial %s: %w (is mailclientd running?)", socket, err)
	}
	cli := &Client{
		Socket:  socket,
		conn:    c,
		w:       bufio.NewWriter(c),
		pending: map[uint64]chan Response{},
	}
	go cli.readLoop()
	return cli, nil
}

// DialWait retries Dial until timeout.
func DialWait(socket string, wait time.Duration) (*Client, error) {
	deadline := time.Now().Add(wait)
	var last error
	for time.Now().Before(deadline) {
		cli, err := Dial(socket)
		if err == nil {
			return cli, nil
		}
		last = err
		time.Sleep(25 * time.Millisecond)
	}
	if last == nil {
		last = fmt.Errorf("mailclientui: timeout dialing %s", socket)
	}
	return nil, last
}

// Close drops the socket.
func (c *Client) Close() error {
	c.closed.Store(true)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// OnEvent registers a handler for mail.changed / mail.fetched.
func (c *Client) OnEvent(fn func(Event)) {
	c.mu.Lock()
	c.onEvent = fn
	c.mu.Unlock()
}

func (c *Client) readLoop() {
	sc := bufio.NewScanner(c.conn)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var peek map[string]json.RawMessage
		if err := json.Unmarshal(line, &peek); err != nil {
			continue
		}
		if _, hasErr := peek["error"]; hasErr || len(peek["result"]) > 0 || len(peek["id"]) > 0 && peek["method"] == nil {
			var resp Response
			if err := json.Unmarshal(line, &resp); err != nil {
				continue
			}
			c.deliver(resp)
			continue
		}
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		if req.Method == "" {
			continue
		}
		ev := Event{Method: req.Method}
		var p eventParams
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &p)
			ev.Folder = p.FolderID
			ev.Reason = p.Reason
			ev.Count = p.Count
		}
		c.mu.Lock()
		fn := c.onEvent
		c.mu.Unlock()
		if fn != nil {
			fn(ev)
		}
	}
}

func (c *Client) deliver(resp Response) {
	id, ok := jsonNumber(resp.ID)
	if !ok {
		return
	}
	c.mu.Lock()
	ch := c.pending[id]
	delete(c.pending, id)
	c.mu.Unlock()
	if ch != nil {
		ch <- resp
	}
}

func jsonNumber(v any) (uint64, bool) {
	switch n := v.(type) {
	case float64:
		return uint64(n), true
	case json.Number:
		u, err := n.Int64()
		return uint64(u), err == nil
	case int:
		return uint64(n), true
	case int64:
		return uint64(n), true
	case uint64:
		return n, true
	default:
		return 0, false
	}
}

func (c *Client) call(method string, params any, result any) error {
	id := c.nextID.Add(1)
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req := Request{JSONRPC: RPCVersion, ID: id, Method: method}
	if params != nil {
		req.Params = raw
	}
	ch := make(chan Response, 1)
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("mailclientui: not connected")
	}
	c.pending[id] = ch
	if err := writeJSON(c.w, req); err != nil {
		delete(c.pending, id)
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	select {
	case resp := <-ch:
		if resp.Error != nil {
			return resp.Error
		}
		if result == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
			return nil
		}
		return json.Unmarshal(resp.Result, result)
	case <-timer.C:
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return fmt.Errorf("mailclientui: timeout on %s", method)
	}
}

func (c *Client) Ping() error {
	var out map[string]string
	return c.call(MethodPing, nil, &out)
}

func (c *Client) Status() (DaemonStatus, error) {
	var st DaemonStatus
	err := c.call(MethodStatusGet, nil, &st)
	return st, err
}

func (c *Client) Accounts() ([]Account, error) {
	var out []Account
	err := c.call(MethodAccountsList, nil, &out)
	return out, err
}

func (c *Client) ListFolders(accountID string) ([]Folder, error) {
	var out []Folder
	err := c.call(MethodFoldersList, folderListParams{AccountID: accountID}, &out)
	return out, err
}

func (c *Client) GetFolder(id FolderID) (Folder, bool, error) {
	var f Folder
	err := c.call(MethodFoldersGet, folderGetParams{ID: id}, &f)
	if err != nil {
		return Folder{}, false, err
	}
	return f, f.ID != "", nil
}

func (c *Client) CreateFolder(accountID, name string, parent FolderID) (Folder, error) {
	var f Folder
	err := c.call(MethodFoldersCreate, folderCreateParams{AccountID: accountID, Name: name, Parent: parent}, &f)
	return f, err
}

func (c *Client) ListMessages(folder FolderID, filter Filter) ([]Message, error) {
	var out []Message
	p := messagesListParams{FolderID: folder}
	if !filterEmpty(filter) {
		p.Filter = &filter
	}
	err := c.call(MethodMessagesList, p, &out)
	if out == nil {
		out = []Message{}
	}
	return out, err
}

func (c *Client) GetMessage(id MessageID) (Message, bool, error) {
	var m Message
	err := c.call(MethodMessagesGet, messageIDParams{ID: id}, &m)
	if err != nil {
		return Message{}, false, err
	}
	return m, m.ID != "", nil
}

func (c *Client) Search(q SearchQuery) ([]Message, error) {
	var out []Message
	err := c.call(MethodMessagesSearch, searchParams{AccountID: q.AccountID, FolderID: q.Folder, Filter: q.Filter}, &out)
	if out == nil {
		out = []Message{}
	}
	return out, err
}

func (c *Client) SetFlags(id MessageID, patch FlagPatch) error {
	return c.call(MethodMessagesFlags, setFlagsParams{ID: id, Patch: patchToWire(patch)}, nil)
}

func (c *Client) Move(ids []MessageID, dest FolderID) error {
	return c.call(MethodMessagesMove, moveParams{IDs: ids, Dest: dest}, nil)
}

func (c *Client) Delete(ids []MessageID) error {
	return c.call(MethodMessagesDelete, deleteParams{IDs: ids}, nil)
}

func (c *Client) Append(folder FolderID, msg Message) (MessageID, error) {
	var r appendResult
	err := c.call(MethodMessagesAppend, appendParams{FolderID: folder, Message: msg}, &r)
	return r.ID, err
}

func (c *Client) Update(id MessageID, msg Message) error {
	return c.call(MethodMessagesUpdate, updateParams{ID: id, Message: msg}, nil)
}

func (c *Client) Fetch(accountID string) (int, error) {
	var r fetchResult
	err := c.call(MethodMessagesFetch, fetchParams{AccountID: accountID}, &r)
	return r.Count, err
}

func (c *Client) Unread(folder FolderID) (int, error) {
	var r unreadResult
	err := c.call(MethodUnreadGet, unreadParams{FolderID: folder}, &r)
	return r.Count, err
}

func (c *Client) UnreadTotal() (int, error) {
	var r unreadResult
	err := c.call(MethodUnreadGet, unreadParams{}, &r)
	return r.Count, err
}

func (c *Client) Send(accountID string, msg Message, draftID MessageID) (MessageID, error) {
	var r appendResult
	err := c.call(MethodComposeSend, composeParams{AccountID: accountID, Message: msg, ID: draftID}, &r)
	return r.ID, err
}

func (c *Client) SaveDraft(accountID string, msg Message, draftID MessageID) (MessageID, error) {
	var r appendResult
	err := c.call(MethodComposeDraft, composeParams{AccountID: accountID, Message: msg, ID: draftID}, &r)
	return r.ID, err
}

func filterEmpty(f Filter) bool {
	return f.Query == "" && !f.Unread && !f.Starred && !f.Attachment && f.Tag == "" &&
		!f.Sender && !f.Recipients && !f.SubjectOnly && !f.Body
}
