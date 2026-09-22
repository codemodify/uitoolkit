package widgets_test

import (
	"fmt"
	"testing"

	"github.com/codemodify/uitoolkit/richtext"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

const richSample = `<h1>Release notes</h1>
<p>This build brings <b>bold</b>, <i>italic</i>, <u>underlined</u> and <s>struck</s> text, <code>monospace</code>,
<span style="font-size:20px">larger</span> and <span style="color:#c0392b">coloured</span> words, a
<span style="background-color:#ffe066">highlight</span> and <a href="https://example.com">a link</a>.</p>
<h2>Lists</h2>
<ul><li>Bullets<ul><li>that nest</li><li>as deep as you like</li></ul></li><li>and come back</li></ul>
<ol><li>Numbered</li><li>items<ol><li>with letters under them</li></ol></li></ol>
<p style="text-align:center">A centred paragraph that wraps onto a second line when the editor is narrow enough to need it.</p>`

func TestRichTextShots(t *testing.T) {
	for _, pack := range breadthLooks {
		for _, scale := range []float32{1, 1.75} {
			if testing.Short() && scale != 1 {
				continue
			}
			name := fmt.Sprintf("richtext-%s@%g", pack, scale)
			t.Run(name, func(t *testing.T) {
				var ed *widgets.RichText
				a, win := shotWindow(t, pack, scale, 560, 460, func() widget.Component {
					ed = widgets.NewRichTextHTML(richSample)
					bar := widgets.NewRichTextBar(ed)
					col := widgets.NewColumn(bar, ed).WithGap(6).WithPad(10)
					col.AddFlex(ed, 1)
					return col
				})
				defer win.Close()
				win.RequestFocus(ed)
				// Select "bold, italic" so the selection shows too.
				d := ed.Document()
				d.SetSelection(richtext.Pos{Block: 1, Off: 17}, richtext.Pos{Block: 1, Off: 29})
				a.PumpOnce()
				writeShot(t, win, name)
			})
		}
	}
}
