package mail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/ianaindex"
)

var wordDec = &mime.WordDecoder{
	CharsetReader: charsetReader,
}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	charset = strings.TrimSpace(charset)
	if charset == "" || strings.EqualFold(charset, "utf-8") || strings.EqualFold(charset, "us-ascii") {
		return input, nil
	}
	if e, err := ianaindex.MIME.Encoding(charset); err == nil && e != nil {
		return e.NewDecoder().Reader(input), nil
	}
	if e, err := htmlindex.Get(charset); err == nil && e != nil {
		return e.NewDecoder().Reader(input), nil
	}
	return input, nil
}

func decodeRFC2047(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	out, err := wordDec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return out
}

// ParseRFC822 fills a Message from a raw message. Attachments stay as Parts.
func ParseRFC822(raw []byte, folder FolderID, accountID string) (Message, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return Message{}, err
	}
	h := msg.Header
	out := Message{
		Folder:    folder,
		AccountID: accountID,
		From:      decodeRFC2047(h.Get("From")),
		To:        decodeRFC2047(h.Get("To")),
		Cc:        decodeRFC2047(h.Get("Cc")),
		Bcc:       decodeRFC2047(h.Get("Bcc")),
		Subject:      decodeRFC2047(h.Get("Subject")),
		Size:         len(raw),
		RFCMessageID: h.Get("Message-Id"),
		InReplyTo:    h.Get("In-Reply-To"),
		References:   h.Get("References"),
	}
	out.ThreadID = ThreadIDOf(out)
	if t, err := mail.ParseDate(h.Get("Date")); err == nil {
		out.Date = t
	}
	ct := h.Get("Content-Type")
	media, params, _ := mime.ParseMediaType(ct)
	if media == "" {
		media = "text/plain"
	}
	body, _ := io.ReadAll(msg.Body)
	if strings.HasPrefix(strings.ToLower(media), "multipart/") {
		parseMultipart(&out, media, params["boundary"], body, "1")
	} else {
		decoded := decodeTransfer(body, h.Get("Content-Transfer-Encoding"))
		decoded = decodeCharset(decoded, params["charset"])
		applyTextPart(&out, media, decoded, params["name"], false, "1")
		out.Parts = append(out.Parts, Part{
			ID: "1", MIMEType: media, Size: len(decoded), Charset: params["charset"],
			Filename: params["name"],
		})
	}
	if out.Snippet == "" {
		out.Snippet = snippetOf(out.Body)
	}
	return out, nil
}

func parseMultipart(out *Message, media, boundary string, body []byte, prefix string) {
	if boundary == "" {
		out.Body = string(body)
		return
	}
	r := multipart.NewReader(bytes.NewReader(body), boundary)
	i := 0
	for {
		p, err := r.NextPart()
		if err != nil {
			break
		}
		i++
		id := fmt.Sprintf("%s.%d", prefix, i)
		if prefix == "1" && !strings.Contains(prefix, ".") {
			id = fmt.Sprintf("%d", i)
		}
		pct := p.Header.Get("Content-Type")
		pmedia, params, _ := mime.ParseMediaType(pct)
		if pmedia == "" {
			pmedia = "text/plain"
		}
		raw, _ := io.ReadAll(p)
		disp, dparams, _ := mime.ParseMediaType(p.Header.Get("Content-Disposition"))
		filename := dparams["filename"]
		if filename == "" {
			filename = params["name"]
		}
		filename = decodeRFC2047(filename)
		if strings.HasPrefix(strings.ToLower(pmedia), "multipart/") {
			parseMultipart(out, pmedia, params["boundary"], raw, id)
			continue
		}
		decoded := decodeTransfer(raw, p.Header.Get("Content-Transfer-Encoding"))
		inline := !strings.EqualFold(disp, "attachment")
		if strings.HasPrefix(strings.ToLower(pmedia), "text/") {
			decoded = decodeCharset(decoded, params["charset"])
			applyTextPart(out, pmedia, decoded, filename, strings.EqualFold(disp, "attachment"), id)
		} else if filename != "" || strings.EqualFold(disp, "attachment") {
			out.HasAttach = true
			if filename != "" && !containsStr(out.Attachments, filename) {
				out.Attachments = append(out.Attachments, filename)
			}
		}
		out.Parts = append(out.Parts, Part{
			ID: id, MIMEType: pmedia, Filename: filename, Size: len(decoded),
			Charset: params["charset"], Inline: inline && filename == "",
		})
	}
}

func applyTextPart(out *Message, media string, data []byte, filename string, attach bool, id string) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	low := strings.ToLower(media)
	if attach && filename != "" {
		out.HasAttach = true
		if !containsStr(out.Attachments, filename) {
			out.Attachments = append(out.Attachments, filename)
		}
		return
	}
	if strings.Contains(low, "html") {
		if out.HTML == "" {
			out.HTML = SanitizeHTML(text)
		}
		if out.Body == "" {
			out.Body = HTMLToText(text)
		}
		return
	}
	if out.Body == "" {
		out.Body = text
	}
}

func decodeTransfer(b []byte, enc string) []byte {
	enc = strings.ToLower(strings.TrimSpace(enc))
	switch enc {
	case "base64":
		s := bytes.Join(bytes.Fields(b), nil)
		if dec, err := decodeBase64(s); err == nil {
			return dec
		}
	case "quoted-printable":
		if dec, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(b))); err == nil {
			return dec
		}
	}
	return b
}

func decodeBase64(s []byte) ([]byte, error) {
	out := make([]byte, base64.StdEncoding.DecodedLen(len(s)))
	n, err := base64.StdEncoding.Decode(out, s)
	return out[:n], err
}

func decodeCharset(b []byte, charset string) []byte {
	if utf8.Valid(b) && (charset == "" || strings.EqualFold(charset, "utf-8") || strings.EqualFold(charset, "us-ascii")) {
		return b
	}
	r, err := charsetReader(charset, bytes.NewReader(b))
	if err != nil {
		return b
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return b
	}
	return out
}

func snippetOf(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	runes := []rune(s)
	if len(runes) > 140 {
		return string(runes[:140]) + "…"
	}
	return s
}

func containsStr(in []string, s string) bool {
	for _, x := range in {
		if x == s {
			return true
		}
	}
	return false
}

// DisplayBody is the text-only message view: prefer text/plain, else strip
// HTML tags. There is no HTML engine — the UI never paints markup.
func DisplayBody(m Message) string {
	if m.Body != "" && !looksLikeHTML(m.Body) {
		return m.Body
	}
	if m.HTML != "" {
		return HTMLToText(m.HTML)
	}
	if m.Body != "" {
		return HTMLToText(m.Body)
	}
	return ""
}

func looksLikeHTML(s string) bool {
	low := strings.ToLower(s)
	return strings.Contains(low, "<html") || strings.Contains(low, "<body") ||
		strings.Contains(low, "<div") || strings.Contains(low, "<p>") ||
		strings.Contains(low, "<br") || strings.Contains(low, "<span") ||
		strings.Contains(low, "<table") || strings.Contains(low, "<!doctype")
}

func plainMessage(m Message) Message {
	m.Body = DisplayBody(m)
	m.HTML = ""
	if m.Snippet == "" {
		m.Snippet = snippetOf(m.Body)
	}
	return m
}

// HTMLToText is a conservative tag stripper (no JS execution).
func HTMLToText(html string) string {
	html = stripBlocks(html, "script")
	html = stripBlocks(html, "style")
	var b strings.Builder
	inTag := false
	for i := 0; i < len(html); i++ {
		c := html[i]
		if c == '<' {
			inTag = true
			name := tagName(html[i:])
			switch name {
			case "br", "p", "div", "tr", "li", "h1", "h2", "h3", "h4":
				b.WriteByte('\n')
			}
			continue
		}
		if inTag {
			if c == '>' {
				inTag = false
			}
			continue
		}
		b.WriteByte(c)
	}
	s := strings.ReplaceAll(b.String(), "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	return strings.TrimSpace(s)
}

// SanitizeHTML drops script/style/iframe and on* handlers. Used only when
// caching a MIME part; the message view always uses DisplayBody / HTMLToText.
func SanitizeHTML(html string) string {
	html = stripBlocks(html, "script")
	html = stripBlocks(html, "style")
	html = stripBlocks(html, "iframe")
	html = stripBlocks(html, "object")
	var b strings.Builder
	inTag := false
	var tag strings.Builder
	for i := 0; i < len(html); i++ {
		c := html[i]
		if c == '<' {
			inTag = true
			tag.Reset()
			tag.WriteByte(c)
			continue
		}
		if inTag {
			tag.WriteByte(c)
			if c == '>' {
				inTag = false
				cleaned := stripEventAttrs(tag.String())
				if cleaned != "" {
					b.WriteString(cleaned)
				}
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func stripBlocks(s, tag string) string {
	low := strings.ToLower(s)
	open := "<" + tag
	close := "</" + tag + ">"
	var b strings.Builder
	for {
		i := strings.Index(low, open)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		restLow := low[i:]
		j := strings.Index(restLow, close)
		if j < 0 {
			return b.String()
		}
		skip := j + len(close)
		s = s[i+skip:]
		low = low[i+skip:]
	}
}

func tagName(s string) string {
	if len(s) < 2 || s[0] != '<' {
		return ""
	}
	i := 1
	if i < len(s) && s[i] == '/' {
		i++
	}
	start := i
	for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	return strings.ToLower(s[start:i])
}

func stripEventAttrs(tag string) string {
	low := strings.ToLower(tag)
	if strings.HasPrefix(low, "<script") || strings.HasPrefix(low, "<iframe") {
		return ""
	}
	var b strings.Builder
	i := 0
	for i < len(tag) {
		if i+3 < len(tag) && (tag[i] == ' ' || tag[i] == '\t') && (tag[i+1] == 'o' || tag[i+1] == 'O') && (tag[i+2] == 'n' || tag[i+2] == 'N') {
			j := i + 3
			for j < len(tag) && tag[j] != '=' && tag[j] != '>' && tag[j] != ' ' {
				j++
			}
			if j < len(tag) && tag[j] == '=' {
				j++
				if j < len(tag) && (tag[j] == '"' || tag[j] == '\'') {
					q := tag[j]
					j++
					for j < len(tag) && tag[j] != q {
						j++
					}
					if j < len(tag) {
						j++
					}
				} else {
					for j < len(tag) && tag[j] != ' ' && tag[j] != '>' {
						j++
					}
				}
				i = j
				continue
			}
		}
		b.WriteByte(tag[i])
		i++
	}
	return b.String()
}

// BuildRFC822 writes a text or multipart message for SMTP / IMAP APPEND.
func BuildRFC822(msg Message, ident Identity, files []AttachedFile) []byte {
	from := strings.TrimSpace(msg.From)
	if from == "" && ident.Address != "" {
		from = ident.DisplayFrom()
	}
	body := msg.Body
	if ident.Signature != "" && !strings.Contains(body, ident.Signature) {
		body = strings.TrimRight(body, "\n") + "\n\n-- \n" + ident.Signature + "\n"
	}
	var b bytes.Buffer
	hdr := func(k, v string) {
		if strings.TrimSpace(v) == "" {
			return
		}
		fmt.Fprintf(&b, "%s: %s\r\n", k, v)
	}
	hdr("From", from)
	hdr("To", msg.To)
	hdr("Cc", msg.Cc)
	hdr("Bcc", msg.Bcc)
	hdr("Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	date := msg.Date
	if date.IsZero() {
		date = time.Now()
	}
	hdr("Date", date.Format(time.RFC1123Z))
	mid := strings.TrimSpace(msg.RFCMessageID)
	if mid == "" {
		mid = fmt.Sprintf("<%d.%s@uitoolkit>", date.UnixNano(), slug(ident.Address))
	}
	hdr("Message-ID", mid)
	hdr("In-Reply-To", msg.InReplyTo)
	hdr("References", msg.References)
	hdr("MIME-Version", "1.0")
	if len(files) == 0 {
		hdr("Content-Type", `text/plain; charset="utf-8"`)
		hdr("Content-Transfer-Encoding", "8bit")
		b.WriteString("\r\n")
		b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
		if !strings.HasSuffix(body, "\n") {
			b.WriteString("\r\n")
		}
		return b.Bytes()
	}
	boundary := fmt.Sprintf("uitk-%d", date.UnixNano())
	hdr("Content-Type", `multipart/mixed; boundary="`+boundary+`"`)
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "--%s\r\n", boundary)
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=\"utf-8\"\r\n")
	fmt.Fprintf(&b, "Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	b.WriteString("\r\n")
	for _, f := range files {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		ct := f.MIME
		if ct == "" {
			ct = "application/octet-stream"
		}
		name := mime.QEncoding.Encode("utf-8", f.Name)
		fmt.Fprintf(&b, "Content-Type: %s; name=\"%s\"\r\n", ct, name)
		fmt.Fprintf(&b, "Content-Disposition: attachment; filename=\"%s\"\r\n", name)
		fmt.Fprintf(&b, "Content-Transfer-Encoding: base64\r\n\r\n")
		enc := base64.StdEncoding.EncodeToString(f.Data)
		for i := 0; i < len(enc); i += 76 {
			end := i + 76
			if end > len(enc) {
				end = len(enc)
			}
			b.WriteString(enc[i:end])
			b.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes()
}

// AttachedFile is a compose attachment (path already read by the daemon).
type AttachedFile struct {
	Name string
	MIME string
	Data []byte
}

func guessMIME(name string) string {
	ext := strings.ToLower(name)
	if i := strings.LastIndex(ext, "."); i >= 0 {
		if t := mime.TypeByExtension(ext[i:]); t != "" {
			return t
		}
	}
	return "application/octet-stream"
}

