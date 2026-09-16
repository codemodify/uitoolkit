package mail

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// Dragging an attachment out of the reading pane, which is how a mail
// client hands a file to a file manager, an editor, or another mail
// client's compose window.
//
// The other side of a drag wants a path, not bytes, so the attachment is
// written out when the drag starts. That fetch is synchronous: the drag
// has to know whether there is a file at all before it offers one, and a
// drag that begins by promising a path it cannot produce is worse than no
// drag. Attachments are small and the store is usually local; a slow one
// costs the moment between the press and the drag taking hold.

// attachScratch is where dragged attachments are written, made once per
// process.
var attachScratch struct {
	sync.Once
	dir string
}

func attachScratchDir() string {
	attachScratch.Do(func() {
		if dir, err := os.MkdirTemp("", "uitk-mail-drag-"); err == nil {
			attachScratch.dir = dir
		}
	})
	return attachScratch.dir
}

// dragAttachment is the drag of attachment i: its file, written out, and
// offered as a uri-list the way every file drag is.
func (s *session) dragAttachment(i int) *widget.Drag {
	m, ok := s.primary()
	if !ok || i < 0 || i >= len(s.attNames) {
		return nil
	}
	dir := attachScratchDir()
	if dir == "" {
		return nil
	}
	data, err := s.attachmentBytes(m, i)
	if err != nil {
		s.mark("Drag: " + err.Error())
		return nil
	}
	name := attachFileName(s.attNames[i])
	// One directory per attachment: two messages with a scan.pdf each
	// must not write over one another mid-drag.
	sub := filepath.Join(dir, safeDirName(m.ID, i))
	if err := os.MkdirAll(sub, 0o700); err != nil {
		s.mark("Drag: " + err.Error())
		return nil
	}
	path := filepath.Join(sub, name)
	if err := writeFileAtomic(path, data, 0o600); err != nil {
		s.mark("Drag: " + err.Error())
		return nil
	}
	d := widget.DragFiles(path)
	if d == nil {
		return nil
	}
	// An attachment is a copy: the message keeps it whatever the target
	// does, so offering a move would promise something untrue.
	d.Actions, d.Preferred = platform.DragCopy, platform.DragCopy
	if s.win != nil {
		d.Image, d.Hotspot = widget.DragLabel(s.win.Look(), name, s.win.Scale())
	}
	d.Done = func(action platform.DragAction) {
		if action == platform.DragNone {
			s.mark("Drag cancelled")
			return
		}
		s.mark("Dragged " + name)
	}
	return d
}

// safeDirName is a directory name for one message's attachment, with
// nothing in it a path could be traversed with.
func safeDirName(id MessageID, i int) string {
	out := make([]rune, 0, len(id)+4)
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
		if len(out) >= 48 {
			break
		}
	}
	return string(out) + "-" + string(rune('a'+i%26))
}

// DragAt drags the attachment out of its row (widget.DragSource).
func (h *attachHit) DragAt(paintengine2d.Point) *widget.Drag {
	if h.Drag == nil {
		return nil
	}
	return h.Drag()
}
