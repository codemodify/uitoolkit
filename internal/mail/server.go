package mail

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Server is mailclientd: one Store, many Unix-socket JSON-RPC clients.
type Server struct {
	Store  Store
	Socket string

	mu      sync.Mutex
	conns   map[*rpcConn]struct{}
	serving bool
}

type rpcConn struct {
	net.Conn
	mu sync.Mutex
	w  *bufio.Writer
}

func (c *rpcConn) send(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return writeJSON(c.w, v)
}

// NewServer owns store and advertises socket in status.get.
func NewServer(store Store, socket string) *Server {
	if store == nil {
		store = NewDemoStore()
	}
	if socket == "" {
		socket = DefaultSocket()
	}
	return &Server{Store: store, Socket: socket, conns: map[*rpcConn]struct{}{}}
}

// ListenAndServe binds a Unix socket and serves until ctx is cancelled.
func ListenAndServe(ctx context.Context, socket string, store Store) error {
	if err := os.RemoveAll(socket); err != nil {
		return err
	}
	ln, err := net.Listen("unix", socket)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	srv := NewServer(store, socket)
	return srv.Serve(ln)
}

// Serve accepts connections on ln until the listener closes.
func (s *Server) Serve(ln net.Listener) error {
	s.mu.Lock()
	s.serving = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.serving = false
		s.mu.Unlock()
	}()
	for {
		c, err := ln.Accept()
		if err != nil {
			if s.ctxClosed(err) {
				return nil
			}
			return err
		}
		go s.handleConn(c)
	}
}

func (s *Server) ctxClosed(err error) bool {
	return err != nil && (isClosed(err) || !s.isServing())
}

func (s *Server) isServing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serving
}

func isClosed(err error) bool {
	if err == nil {
		return false
	}
	return err == net.ErrClosed || strings.Contains(err.Error(), "use of closed network connection")
}

func (s *Server) handleConn(raw net.Conn) {
	c := &rpcConn{Conn: raw, w: bufio.NewWriter(raw)}
	s.mu.Lock()
	s.conns[c] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.conns, c)
		s.mu.Unlock()
		_ = c.Close()
	}()
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = c.send(Response{JSONRPC: RPCVersion, Error: &RPCError{Code: -32700, Message: err.Error()}})
			continue
		}
		resp := s.dispatch(req)
		if req.ID == nil {
			continue
		}
		resp.JSONRPC = RPCVersion
		resp.ID = req.ID
		if err := c.send(resp); err != nil {
			return
		}
	}
}

func writeJSON(w *bufio.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	if err := w.WriteByte('\n'); err != nil {
		return err
	}
	return w.Flush()
}

func (s *Server) dispatch(req Request) Response {
	var (
		result any
		err    error
	)
	switch req.Method {
	case MethodPing:
		result = map[string]string{"pong": "ok"}
	case MethodStatusGet:
		result, err = s.status()
	case MethodAccountsList:
		result = s.Store.Accounts()
	case MethodFoldersList:
		var p folderListParams
		p, err = decodeParams[folderListParams](req.Params)
		if err == nil {
			result = s.Store.ListFolders(p.AccountID)
		}
	case MethodFoldersGet:
		var p folderGetParams
		p, err = decodeParams[folderGetParams](req.Params)
		if err == nil {
			f, ok := s.Store.GetFolder(p.ID)
			if !ok {
				err = fmt.Errorf("mail: no folder %s", p.ID)
			} else {
				result = f
			}
		}
	case MethodFoldersCreate:
		var p folderCreateParams
		p, err = decodeParams[folderCreateParams](req.Params)
		if err == nil {
			result, err = s.Store.CreateFolder(p.AccountID, p.Name, p.Parent)
			if err == nil {
				s.broadcast(EventChanged, eventParams{FolderID: result.(Folder).ID, Reason: "create-folder"})
			}
		}
	case MethodMessagesList:
		var p messagesListParams
		p, err = decodeParams[messagesListParams](req.Params)
		if err == nil {
			all := s.Store.ListMessages(p.FolderID)
			if p.Filter != nil {
				var hit []Message
				for _, m := range all {
					if p.Filter.Match(m) {
						hit = append(hit, m)
					}
				}
				result = hit
			} else {
				result = all
			}
		}
	case MethodMessagesGet:
		var p messageIDParams
		p, err = decodeParams[messageIDParams](req.Params)
		if err == nil {
			m, ok := s.Store.GetMessage(p.ID)
			if !ok {
				err = fmt.Errorf("mail: no message %s", p.ID)
			} else {
				result = m
			}
		}
	case MethodMessagesSearch:
		var p searchParams
		p, err = decodeParams[searchParams](req.Params)
		if err == nil {
			result = s.Store.Search(SearchQuery{AccountID: p.AccountID, Folder: p.FolderID, Filter: p.Filter})
		}
	case MethodMessagesFlags:
		var p setFlagsParams
		p, err = decodeParams[setFlagsParams](req.Params)
		if err == nil {
			err = s.Store.SetFlags(p.ID, p.Patch.to())
			if err == nil {
				if m, ok := s.Store.GetMessage(p.ID); ok {
					s.broadcast(EventChanged, eventParams{FolderID: m.Folder, Reason: "flags"})
				}
			}
		}
	case MethodMessagesMove:
		var p moveParams
		p, err = decodeParams[moveParams](req.Params)
		if err == nil {
			err = s.Store.Move(p.IDs, p.Dest)
			if err == nil {
				s.broadcast(EventChanged, eventParams{FolderID: p.Dest, Reason: "move"})
			}
		}
	case MethodMessagesDelete:
		var p deleteParams
		p, err = decodeParams[deleteParams](req.Params)
		if err == nil {
			err = s.Store.Delete(p.IDs)
			if err == nil {
				s.broadcast(EventChanged, eventParams{Reason: "delete"})
			}
		}
	case MethodMessagesAppend:
		var p appendParams
		p, err = decodeParams[appendParams](req.Params)
		if err == nil {
			var id MessageID
			id, err = s.Store.Append(p.FolderID, p.Message)
			if err == nil {
				result = appendResult{ID: id}
				s.broadcast(EventChanged, eventParams{FolderID: p.FolderID, Reason: "append"})
			}
		}
	case MethodMessagesUpdate:
		var p updateParams
		p, err = decodeParams[updateParams](req.Params)
		if err == nil {
			err = s.Store.Update(p.ID, p.Message)
			if err == nil {
				s.broadcast(EventChanged, eventParams{Reason: "update"})
			}
		}
	case MethodMessagesFetch:
		var p fetchParams
		p, err = decodeParams[fetchParams](req.Params)
		if err == nil {
			var n int
			n, err = s.Store.Fetch(p.AccountID)
			if err == nil {
				result = fetchResult{Count: n}
				s.broadcast(EventFetched, eventParams{AccountID: p.AccountID, Count: n})
			}
		}
	case MethodUnreadGet:
		var p unreadParams
		p, err = decodeParams[unreadParams](req.Params)
		if err == nil {
			if p.FolderID == "" {
				result = unreadResult{Count: s.Store.UnreadTotal()}
			} else {
				result = unreadResult{Count: s.Store.Unread(p.FolderID)}
			}
		}
	case MethodComposeSend:
		var p composeParams
		p, err = decodeParams[composeParams](req.Params)
		if err == nil {
			result, err = s.send(p)
		}
	case MethodComposeDraft:
		var p composeParams
		p, err = decodeParams[composeParams](req.Params)
		if err == nil {
			result, err = s.saveDraft(p)
		}
	case MethodIdentitiesList:
		var p identityListParams
		p, err = decodeParams[identityListParams](req.Params)
		if err == nil {
			result = s.Store.Identities(p.AccountID)
		}
	case MethodIdentitiesPut:
		var id Identity
		id, err = decodeParams[Identity](req.Params)
		if err == nil {
			result, err = s.Store.PutIdentity(id)
			if err == nil {
				s.broadcast(EventChanged, eventParams{Reason: "identity"})
			}
		}
	case MethodIdentitiesDel:
		var p identityIDParams
		p, err = decodeParams[identityIDParams](req.Params)
		if err == nil {
			err = s.Store.DeleteIdentity(p.ID)
		}
	case MethodTagsList:
		result = s.Store.ListTags()
	case MethodTagsPut:
		var t Tag
		t, err = decodeParams[Tag](req.Params)
		if err == nil {
			result, err = s.Store.PutTag(t)
		}
	case MethodFoldersVirtual:
		result = s.Store.VirtualFolders()
	case MethodFiltersList:
		result = s.Store.ListRules()
	case MethodFiltersPut:
		var r FilterRule
		r, err = decodeParams[FilterRule](req.Params)
		if err == nil {
			result, err = s.Store.PutRule(r)
		}
	case MethodFiltersDel:
		var p ruleIDParams
		p, err = decodeParams[ruleIDParams](req.Params)
		if err == nil {
			err = s.Store.DeleteRule(p.ID)
		}
	case MethodFiltersApply:
		var p applyRulesParams
		p, err = decodeParams[applyRulesParams](req.Params)
		if err == nil {
			var n int
			n, err = s.Store.ApplyRules(p.FolderID)
			if err == nil {
				result = applyResult{Count: n}
				s.broadcast(EventChanged, eventParams{FolderID: p.FolderID, Reason: "filters"})
			}
		}
	case MethodMessagesPart:
		var p partParams
		p, err = decodeParams[partParams](req.Params)
		if err == nil {
			result, err = s.Store.GetPart(p.ID, p.PartID)
		}
	case MethodMessagesOpen:
		var p partParams
		p, err = decodeParams[partParams](req.Params)
		if err == nil {
			result, err = s.Store.OpenPart(p.ID, p.PartID)
		}
	case MethodSyncRun:
		var p fetchParams
		p, err = decodeParams[fetchParams](req.Params)
		if err == nil {
			var r SyncResult
			r, err = s.Store.Sync(p.AccountID)
			if err == nil {
				result = r
				s.broadcast(EventSynced, eventParams{AccountID: p.AccountID, Count: r.New, Reason: "sync"})
				s.broadcast(EventFetched, eventParams{AccountID: p.AccountID, Count: r.New})
			}
		}
	default:
		err = fmt.Errorf("unknown method %s", req.Method)
	}
	if err != nil {
		return Response{Error: &RPCError{Code: -32000, Message: err.Error()}}
	}
	raw, jerr := json.Marshal(result)
	if jerr != nil {
		return Response{Error: &RPCError{Code: -32603, Message: jerr.Error()}}
	}
	return Response{Result: raw}
}

func (s *Server) status() (DaemonStatus, error) {
	st := DaemonStatus{
		Backend:  s.Store.Backend(),
		Socket:   s.Socket,
		Online:   true,
		Accounts: len(s.Store.Accounts()),
	}
	if err := s.Store.Health(); err != nil {
		st.Online = false
		st.Health = err.Error()
	}
	return st, nil
}

func (s *Server) send(p composeParams) (appendResult, error) {
	accountID := p.AccountID
	var ident Identity
	if p.IdentityID != "" {
		for _, id := range s.Store.Identities("") {
			if id.ID == p.IdentityID {
				ident = id
				if id.AccountID != "" {
					accountID = id.AccountID
				}
				break
			}
		}
	}
	if ident.ID == "" {
		ids := s.Store.Identities(accountID)
		for _, id := range ids {
			if id.Default || ident.ID == "" {
				ident = id
			}
		}
	}
	msg := p.Message
	if msg.From == "" && ident.Address != "" {
		msg.From = ident.DisplayFrom()
	}
	if ident.Signature != "" && !strings.Contains(msg.Body, ident.Signature) {
		msg.Body = strings.TrimRight(msg.Body, "\n") + "\n\n-- \n" + ident.Signature + "\n"
	}
	msg.IdentityID = ident.ID
	msg.Read = true
	var files []AttachedFile
	for _, path := range p.AttachPaths {
		b, err := os.ReadFile(path)
		if err != nil {
			return appendResult{}, fmt.Errorf("attach %s: %w", path, err)
		}
		name := path
		if i := strings.LastIndex(path, "/"); i >= 0 {
			name = path[i+1:]
		}
		files = append(files, AttachedFile{Name: name, MIME: guessMIME(name), Data: b})
		msg.HasAttach = true
		msg.Attachments = append(msg.Attachments, name)
	}
	if ls, ok := s.Store.(*LocalStore); ok && len(ls.cfg.Accounts) > 0 {
		id, err := ls.SendViaSMTP(accountID, ident.ID, msg, files)
		if err != nil {
			return appendResult{}, err
		}
		if p.ID != "" {
			_ = s.Store.Delete([]MessageID{p.ID})
		}
		s.broadcast(EventChanged, eventParams{Reason: "send"})
		return appendResult{ID: id}, nil
	}
	sent, ok := specialFolder(s.Store, accountID, FolderSent)
	if !ok {
		return appendResult{}, fmt.Errorf("mail: no Sent folder for %s", accountID)
	}
	id, err := s.Store.Append(sent.ID, msg)
	if err != nil {
		return appendResult{}, err
	}
	if p.ID != "" {
		_ = s.Store.Delete([]MessageID{p.ID})
	}
	s.broadcast(EventChanged, eventParams{FolderID: sent.ID, Reason: "send"})
	return appendResult{ID: id}, nil
}

func (s *Server) saveDraft(p composeParams) (appendResult, error) {
	drafts, ok := specialFolder(s.Store, p.AccountID, FolderDrafts)
	if !ok {
		return appendResult{}, fmt.Errorf("mail: no Drafts folder for %s", p.AccountID)
	}
	msg := p.Message
	msg.Read = true
	if p.ID != "" {
		if err := s.Store.Update(p.ID, msg); err != nil {
			return appendResult{}, err
		}
		s.broadcast(EventChanged, eventParams{FolderID: drafts.ID, Reason: "draft"})
		return appendResult{ID: p.ID}, nil
	}
	id, err := s.Store.Append(drafts.ID, msg)
	if err != nil {
		return appendResult{}, err
	}
	s.broadcast(EventChanged, eventParams{FolderID: drafts.ID, Reason: "draft"})
	return appendResult{ID: id}, nil
}

func (s *Server) broadcast(method string, params eventParams) {
	raw, err := json.Marshal(params)
	if err != nil {
		return
	}
	note := Request{JSONRPC: RPCVersion, Method: method, Params: raw}
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.conns {
		_ = c.send(note)
	}
}

func specialFolder(store Store, accountID string, kind FolderKind) (Folder, bool) {
	for _, f := range store.ListFolders(accountID) {
		if f.Kind == kind && f.Parent == "" {
			return f, true
		}
	}
	return Folder{}, false
}
