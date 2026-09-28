package settingsapp

import (
	"fmt"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// RepoURL is where uitoolkit lives. It is here rather than inlined in
// the sentence that shows it because the About dialog and its test both
// have to say the same thing.
const RepoURL = "https://github.com/codemodify/uitoolkit"

// LicenseName is the licence in LICENSE, named the way the file names
// itself.
const LicenseName = "The Free License"

// aboutButton is the button in front of Apply at the foot of the
// window, and it opens [showAbout].
//
// It is a dialog and not a page. Settings had an About page once, one
// of four behind a sidebar, and it went when the four became one:
// a page is somewhere you can be, and nobody wants to be in About —
// they want to look something up and leave. A dialog is the shape of
// that. It is also why this is not a line of small print under the
// preview: the version, the licence and the repository are read about
// once a year each, and a page that shows them all the time is paying
// for them all the time out of the pane the preview is the point of.
//
// It stands in front of Apply rather than after it because Apply is the
// one thing on this page that writes anything, and the button that
// writes is the last button in the row — that is where every dialog on
// every desktop puts it, and a row that puts something after it invites
// the click that was meant for it. About is also the one button here
// that changes nothing at all, so it is the furthest from the one that
// changes everything.
func (s *settingsState) aboutButton() *widgets.Button {
	b := widgets.NewButton("About", func() { s.showAbout() })
	b.Tip = "What uitoolkit is, which version this is, and what this window is running on."
	b.SetAccessibleName("About uitoolkit")
	b.SetAccessibleDescription(b.Tip)
	return b
}

// aboutFacts is the paragraph that makes this dialog worth opening
// twice, and every number in it is asked for at the moment it is shown
// rather than written down here.
//
// An About box that recites constants is a file somebody could have
// read instead. This one answers the questions a person actually has in
// front of a toolkit that claims to draw everything itself: how many
// looks does it really carry, how many of them are code rather than
// colour lists, and which of the two window backends and which of the
// two paint devices is this window — the one I am looking at — using
// right now. Those four answers differ between two machines, between
// X11 and Wayland, and between a session where EGL started and one
// where it did not, which is exactly why they are worth printing.
func (s *settingsState) aboutFacts() string {
	packs, engines := len(style.ListBuiltinThemes()), len(style.EngineIDs())
	device := "the CPU rasterizer"
	if s.win.PaintBackend() == style.RendererGPU {
		device = "the GPU, through EGL/GLES"
	}
	backend := s.a.BackendName()
	if strings.TrimSpace(backend) == "" {
		backend = "offscreen"
	}
	return fmt.Sprintf(
		"This copy carries %d theme packs, drawn by %d engines — the engines are the code that gives an era its shapes, "+
			"and the packs are what they are given to draw. The window you are reading is on the %s backend, painting on %s, at %g× scale. "+
			"Everything in it, this card included, went through the same widgets the page behind it is configuring: the preview back there is a "+
			"live application, not a screenshot.",
		packs, engines, backend, device, s.win.Scale())
}

// showAbout opens the About card.
//
// It is an [widgets.Overlay] with a [widgets.Panel] in it rather than a
// [widgets.ShowMessageBox], and the difference is the link. A message
// box takes a string and a stock button row; this card has a repository
// address that has to be clickable and reachable by Tab — a URL nobody
// can follow is a URL nobody reads — and a version line that wants to
// be a heading rather than the first sentence of a paragraph. The
// machinery is the toolkit's own either way: promptExportName, a hand's
// width up this file, builds its card the same way.
func (s *settingsState) showAbout() {
	host := s.applyBtn
	if host == nil {
		return
	}
	var overlay *widgets.Overlay
	close := widgets.NewButton("Close", func() { widget.DismissOverlay(overlay) })
	close.Primary = true

	// Every platform the toolkit has a backend for is named, and that
	// is deliberate rather than tidy. This sentence said "X11 and
	// Wayland" for a while after the Win32 and AppKit backends landed,
	// so on a Mac the dialog contradicted its own next paragraph —
	// which names the backend the window is actually on.
	// TestAboutDialogSaysWhatItIsAbout holds it to all four or none.
	what := widgets.NewLabel(
		"A desktop toolkit written in Go and nothing else. Every pixel in this window is drawn by paintengine2d — " +
			"there is no Qt, no GTK, no Skia and no browser underneath it — and the windows themselves come straight " +
			"from the platform: X11 and Wayland on Linux, Win32 on Windows, AppKit on macOS.")
	what.Wrap = true

	facts := widgets.NewLabel(s.aboutFacts())
	facts.Wrap = true

	// The licence in one line, in the words the file uses. Anything
	// longer would be a licence nobody reads in a dialog nobody reads.
	lic := widgets.NewLabel(LicenseName + " — free to use, no restrictions. The full text is in LICENSE.")
	lic.Wrap = true

	link := widgets.NewLinkButton("github.com/codemodify/uitoolkit", RepoURL)
	link.SetAccessibleName("Repository: github.com/codemodify/uitoolkit")

	head := widgets.NewTitle("uitoolkit " + uitoolkit.Version)
	// The version is a heading here and a line of output from
	// `uitoolkit-settings -version` there, and they are the same
	// constant; there is nowhere else in the command that states it.
	head.SetAccessibleName("uitoolkit version " + uitoolkit.Version)

	card := widgets.NewPanel("About uitoolkit",
		head, what, facts, lic, link,
		widgets.NewButtonBox().AddButton(close, widgets.RoleAccept),
	)
	card.Window = true
	card.Raised = true
	card.OnClose = func() { widget.DismissOverlay(overlay) }

	overlay = widgets.NewOverlay(card)
	overlay.Modal = true
	overlay.InitialFocus = close
	// Wide enough that the two paragraphs read as paragraphs rather than
	// as a column of four-word lines, and it still fits the 720x580
	// minimum: the overlay caps a card at 92% of the window's width.
	overlay.MinCardW = 460
	overlay.MinCardH = 300
	widget.ShowOverlay(host, overlay)
}
