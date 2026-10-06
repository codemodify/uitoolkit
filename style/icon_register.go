package style

import (
	"fmt"
	"sync"

	"github.com/codemodify/paintengine2d"
)

// An icon an application ships itself.
//
// The toolkit's rule is that a mark is an icon and never a character, because
// no bundled face carries arrows, checks or chevrons. It ships 56 typed ids
// and 80 stems, and an application whose vocabulary is wider than that — a
// mail client with a *server*, a *compact folder*, a *plain text* body, a
// *turn this filter off* — had nowhere to put one: IconByStem answers only
// typed names and shipped stems, AddSearchPath art can only stand in for a
// stem the toolkit already knows, and a stem-only id draws the missing-icon
// mark in the drawn sets, which every pack uses unless the person picks
// otherwise. So the application drew the mark itself, outside the icon
// system, and lost the sets, the sizes and the tinting with it.

// IconDrawer draws one icon inside b, in col.
//
// It is handed the box the icon is to fill and the ink it is to be drawn in.
// The toolkit's own vectors are drawn for a 24x24 box and scaled from there,
// strokes included, so a drawer written against that box behaves at every
// size and scale the toolkit draws at; nothing requires it, and a drawer may
// read b for itself.
type IconDrawer func(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color)

// appIconBase is where an application's registered icons start, above the
// typed ids and above the stem-only ones.
const appIconBase ToolIcon = 1 << 13

var appIcons struct {
	mu     sync.RWMutex
	stems  []string
	draw   []IconDrawer
	byStem map[string]ToolIcon
}

// RegisterIcon gives the toolkit a vector for a mark it does not ship, under
// a stem of the application's own, and answers the [ToolIcon] to use for it —
// in Button.Icon, MenuItem.Icon, TableColumn.Icon, Label.Icon,
// [DrawToolIcon], anywhere a toolkit icon goes.
//
// The stem is what a *file* icon set matches on, so the icon the application
// draws is still the fallback and not the last word: a person whose icon set
// ships "server.svg" gets theirs, exactly as for every stem the toolkit
// ships. That is the whole reason this takes a stem rather than handing back
// an opaque id.
//
// Registering a stem that is already registered replaces its drawing and
// answers the same id, so a library may register at init and an application
// may override it afterwards without the first id going stale. Registering
// one the toolkit already ships — a typed name or a shipped stem — is
// refused: those have the toolkit's own drawings in every set, and quietly
// replacing them would make one application's icons differ from every other's
// for no reason the person could see.
//
// It is safe to call from any goroutine, and the usual place is once at
// start-up.
func RegisterIcon(stem string, draw IconDrawer) (ToolIcon, error) {
	if stem == "" {
		return IconNone, fmt.Errorf("style: RegisterIcon needs a stem")
	}
	if draw == nil {
		return IconNone, fmt.Errorf("style: RegisterIcon(%q) needs a drawing", stem)
	}
	if _, ok := ToolIconByName(stem); ok {
		return IconNone, fmt.Errorf("style: %q is one of the toolkit's own icons", stem)
	}
	for _, s := range shippedIconStems {
		if s == stem {
			return IconNone, fmt.Errorf("style: %q is one of the toolkit's own stems", stem)
		}
	}
	appIcons.mu.Lock()
	defer appIcons.mu.Unlock()
	if appIcons.byStem == nil {
		appIcons.byStem = map[string]ToolIcon{}
	}
	if icon, ok := appIcons.byStem[stem]; ok {
		appIcons.draw[int(icon-appIconBase)] = draw
		return icon, nil
	}
	icon := appIconBase + ToolIcon(len(appIcons.stems))
	appIcons.stems = append(appIcons.stems, stem)
	appIcons.draw = append(appIcons.draw, draw)
	appIcons.byStem[stem] = icon
	return icon, nil
}

// RegisteredIcon is the icon registered under a stem, and whether there is
// one.
func RegisteredIcon(stem string) (ToolIcon, bool) {
	appIcons.mu.RLock()
	defer appIcons.mu.RUnlock()
	icon, ok := appIcons.byStem[stem]
	return icon, ok
}

// appIcon reports whether icon is one an application registered.
func appIcon(icon ToolIcon) bool {
	if icon < appIconBase {
		return false
	}
	appIcons.mu.RLock()
	defer appIcons.mu.RUnlock()
	return int(icon-appIconBase) < len(appIcons.stems)
}

// appIconStem is the stem icon was registered under, or "".
func appIconStem(icon ToolIcon) string {
	if icon < appIconBase {
		return ""
	}
	appIcons.mu.RLock()
	defer appIcons.mu.RUnlock()
	if i := int(icon - appIconBase); i >= 0 && i < len(appIcons.stems) {
		return appIcons.stems[i]
	}
	return ""
}

// drawAppIcon draws a registered icon, scaled the way the toolkit's own are,
// and reports whether it did.
func drawAppIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon, col paintengine2d.Color) bool {
	if icon < appIconBase || ctx == nil || b.Empty() {
		return false
	}
	appIcons.mu.RLock()
	var draw IconDrawer
	if i := int(icon - appIconBase); i >= 0 && i < len(appIcons.draw) {
		draw = appIcons.draw[i]
	}
	appIcons.mu.RUnlock()
	if draw == nil {
		return false
	}
	drawScaledIcon(ctx, b, func(ctx *paintengine2d.Context, db paintengine2d.Rect) { draw(ctx, db, col) })
	return true
}
