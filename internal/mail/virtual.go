package mail

import "strings"

// Synthetic account / folder ids for virtual views.
const (
	AccountUnified = "unified"
	AccountTags    = "tags"

	FolderUnifiedInbox   FolderID = "virtual/inbox"
	FolderUnifiedUnread  FolderID = "virtual/unread"
	FolderUnifiedStarred FolderID = "virtual/starred"
)

// IsVirtual reports Unified / tag folder ids.
func IsVirtual(id FolderID) bool {
	s := string(id)
	return strings.HasPrefix(s, "virtual/") || strings.HasPrefix(s, "tag/")
}

// TagFolderID is the virtual folder for a named tag.
func TagFolderID(name string) FolderID {
	return FolderID("tag/" + name)
}

// TagNameFromFolder returns the tag for tag/<name> folders.
func TagNameFromFolder(id FolderID) string {
	s := string(id)
	if strings.HasPrefix(s, "tag/") {
		return s[len("tag/"):]
	}
	return ""
}

func defaultVirtualFolders() []Folder {
	return []Folder{
		{ID: FolderUnifiedInbox, AccountID: AccountUnified, Name: "Unified Inbox", Kind: FolderInbox, Virtual: true, MatchKind: FolderInbox},
		{ID: FolderUnifiedUnread, AccountID: AccountUnified, Name: "Unread", Kind: FolderCustom, Virtual: true},
		{ID: FolderUnifiedStarred, AccountID: AccountUnified, Name: "Starred", Kind: FolderCustom, Virtual: true},
	}
}

func virtualFolderByID(id FolderID) (Folder, bool) {
	if name := TagNameFromFolder(id); name != "" {
		return Folder{ID: id, AccountID: AccountTags, Name: name, Kind: FolderCustom, Virtual: true, Tag: name}, true
	}
	for _, f := range defaultVirtualFolders() {
		if f.ID == id {
			return f, true
		}
	}
	return Folder{}, false
}

func matchVirtual(id FolderID, m Message, folder Folder, kind FolderKind) bool {
	switch id {
	case FolderUnifiedInbox:
		return kind == FolderInbox && !folder.Virtual
	case FolderUnifiedUnread:
		return !m.Read
	case FolderUnifiedStarred:
		return m.Starred
	}
	if tag := TagNameFromFolder(id); tag != "" {
		return hasTag(m.Tags, tag)
	}
	return false
}
