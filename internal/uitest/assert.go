package uitest

import (
	"fmt"
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// OverlapError is two rects that must stay exclusive.
type OverlapError struct {
	A, B paintengine2d.Rect
}

func (e OverlapError) Error() string {
	return fmt.Sprintf("rects overlap A=%+v B=%+v", e.A, e.B)
}

// CheckExclusive reports an error when a and b overlap (and are non-empty).
func CheckExclusive(a, b paintengine2d.Rect) error {
	if a.Empty() || b.Empty() {
		return nil
	}
	if a.Overlaps(b) {
		return OverlapError{A: a, B: b}
	}
	return nil
}

// CheckClamped reports when offset is outside [0, max].
func CheckClamped(offset, max float32) error {
	if offset < -0.01 {
		return fmt.Errorf("scroll offset %v < 0", offset)
	}
	if max < 0 {
		return fmt.Errorf("max scroll %v < 0", max)
	}
	if offset > max+0.01 {
		return fmt.Errorf("scroll offset %v > max %v", offset, max)
	}
	return nil
}

// CheckThumbInTrack reports when a non-empty thumb escapes its track.
func CheckThumbInTrack(track, thumb paintengine2d.Rect) error {
	if thumb.Empty() {
		return nil
	}
	if track.Empty() {
		return fmt.Errorf("thumb %+v without track", thumb)
	}
	inset := thumb.Inset(0.5)
	if !track.Overlaps(inset) {
		return fmt.Errorf("thumb %+v misses track %+v", thumb, track)
	}
	if thumb.Min.X < track.Min.X-1 || thumb.Max.X > track.Max.X+1 {
		return fmt.Errorf("thumb x out of track thumb=%+v track=%+v", thumb, track)
	}
	if thumb.Min.Y < track.Min.Y-1 || thumb.Max.Y > track.Max.Y+1 {
		return fmt.Errorf("thumb y out of track thumb=%+v track=%+v", thumb, track)
	}
	return nil
}

// TreeInvariants walks root and checks splitter exclusivity, scroll clamp,
// and overflow thumbs. Used by the widget suite and the app driver.
func TreeInvariants(root widget.Component) []error {
	var errs []error
	widget.Walk(root, func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.Splitter:
			if err := CheckExclusive(v.PaneA(), v.PaneB()); err != nil {
				errs = append(errs, fmt.Errorf("splitter: %w", err))
			}
		case *widgets.ListView:
			if err := CheckClamped(v.OffsetY, v.MaxOffset()); err != nil {
				errs = append(errs, fmt.Errorf("list: %w", err))
			}
			if err := CheckThumbInTrack(v.ScrollTrack()); err != nil {
				errs = append(errs, fmt.Errorf("list thumb: %w", err))
			}
			if v.MaxOffset() > 0 {
				lo, hi := v.VisibleRange()
				if lo >= hi {
					errs = append(errs, fmt.Errorf("list blank window lo=%d hi=%d offset=%v max=%v", lo, hi, v.OffsetY, v.MaxOffset()))
				}
			}
		case *widgets.TableView:
			if err := CheckClamped(v.OffsetY, v.MaxOffset()); err != nil {
				errs = append(errs, fmt.Errorf("table: %w", err))
			}
			if err := CheckThumbInTrack(v.ScrollTrack()); err != nil {
				errs = append(errs, fmt.Errorf("table thumb: %w", err))
			}
			if v.MaxOffset() > 0 {
				lo, hi := v.VisibleRange()
				if lo >= hi {
					errs = append(errs, fmt.Errorf("table blank window lo=%d hi=%d offset=%v max=%v", lo, hi, v.OffsetY, v.MaxOffset()))
				}
			}
			if v.OffsetY <= 0.5 && v.RowCount > 0 {
				row0 := v.RowBounds(0)
				hh := v.HeaderHeight()
				if !row0.Empty() && math.Abs(float64(row0.Min.Y-hh)) > 2 {
					errs = append(errs, fmt.Errorf("table header gap: row0.y=%v headerH=%v", row0.Min.Y, hh))
				}
			}
		case *widgets.TreeView:
			if err := CheckClamped(v.OffsetY, v.MaxOffset()); err != nil {
				errs = append(errs, fmt.Errorf("tree: %w", err))
			}
			if err := CheckThumbInTrack(v.ScrollTrack()); err != nil {
				errs = append(errs, fmt.Errorf("tree thumb: %w", err))
			}
			if v.MaxOffset() > 0 {
				lo, hi := v.VisibleRange()
				if lo >= hi {
					errs = append(errs, fmt.Errorf("tree blank window lo=%d hi=%d", lo, hi))
				}
			}
		case *widgets.CardList:
			if err := CheckClamped(v.OffsetY, v.MaxOffset()); err != nil {
				errs = append(errs, fmt.Errorf("cards: %w", err))
			}
			if err := CheckThumbInTrack(v.ScrollTrack()); err != nil {
				errs = append(errs, fmt.Errorf("cards thumb: %w", err))
			}
			if v.MaxOffset() > 0 {
				lo, hi := v.VisibleRange()
				if lo >= hi {
					errs = append(errs, fmt.Errorf("cards blank window lo=%d hi=%d", lo, hi))
				}
			}
		case *widgets.ScrollView:
			if err := CheckClamped(v.OffsetY, v.MaxOffset()); err != nil {
				errs = append(errs, fmt.Errorf("scrollview: %w", err))
			}
			if err := CheckThumbInTrack(v.ScrollTrack()); err != nil {
				errs = append(errs, fmt.Errorf("scrollview thumb: %w", err))
			}
		case *widgets.TextArea:
			if err := CheckClamped(v.OffsetY(), v.MaxOffset()); err != nil {
				errs = append(errs, fmt.Errorf("textarea: %w", err))
			}
		case *widgets.PopupMenu:
			if err := CheckMenuFitsItems(v); err != nil {
				errs = append(errs, fmt.Errorf("menu: %w", err))
			}
		}
	})
	return errs
}

// CheckMenuFitsItems reports when a popup is narrower than its labels or
// crops rows without offering scroll.
func CheckMenuFitsItems(p *widgets.PopupMenu) error {
	if p == nil || len(p.Items) == 0 {
		return nil
	}
	if err := CheckClamped(p.OffsetY, p.MaxOffset()); err != nil {
		return err
	}
	if err := CheckThumbInTrack(p.ScrollTrack()); err != nil {
		return fmt.Errorf("thumb: %w", err)
	}
	box := p.LocalBounds()
	cs := p.ContentSize()
	if box.Dx()+1 < cs.X {
		return fmt.Errorf("menu width %v < content %v (labels truncated)", box.Dx(), cs.X)
	}
	if cs.Y > box.Dy()+1 && p.MaxOffset() <= 0 {
		return fmt.Errorf("menu height %v < content %v and no scroll", box.Dy(), cs.Y)
	}
	ch := style.MenuChromeFor(p.Look())
	f := p.Look().Font()
	for i, it := range p.Items {
		if it == nil || it.Separator {
			continue
		}
		r := p.ItemBounds(i)
		if r.Empty() {
			return fmt.Errorf("item %d %q empty bounds", i, it.Text)
		}
		label, _, _ := widgets.ParseMnemonic(it.Text)
		lb := p.LabelBounds(i)
		adv := float32(0)
		if f != nil {
			adv = f.Advance(label)
			if ink := f.InkWidth(label); ink > adv {
				adv = ink
			}
		}
		if adv > 0 && lb.Dx()+0.5 < adv {
			return fmt.Errorf("label column %v < %q advance %v", lb.Dx(), label, adv)
		}
		textMax := lb.Min.X + adv
		if adv > 0 && textMax > r.Max.X+0.5 {
			return fmt.Errorf("label %q text maxX %v exceeds item %+v", label, textMax, r)
		}
		if adv > 0 && textMax > box.Max.X+0.5 {
			return fmt.Errorf("label %q text maxX %v exceeds menu %+v", label, textMax, box)
		}
		need := adv + ch.PadL + ch.CheckCol() + ch.ItemPad + ch.PadR
		if adv > 0 && box.Dx()+0.5 < need {
			return fmt.Errorf("menu width %v < %q advance+pad %v", box.Dx(), label, need)
		}
		if sb := p.ShortcutBounds(i); !sb.Empty() {
			if err := CheckExclusive(lb.Inset(0.5), sb.Inset(0.5)); err != nil {
				return fmt.Errorf("label/shortcut overlap %q %q: %w", label, it.Shortcut, err)
			}
			if f != nil {
				if adv := f.Advance(it.Shortcut); adv > 0 && sb.Dx()+0.5 < adv {
					return fmt.Errorf("shortcut column %v < %q advance %v", sb.Dx(), it.Shortcut, adv)
				}
			}
			if sb.Max.X > box.Max.X+1 {
				return fmt.Errorf("shortcut %q escapes menu %+v", it.Shortcut, sb)
			}
		}
		if p.MaxOffset() <= 0 {
			if r.Min.Y < box.Min.Y-1 || r.Max.Y > box.Max.Y+1 {
				return fmt.Errorf("item %d %q %+v clipped by menu %+v", i, label, r, box)
			}
		}
	}
	if p.MaxOffset() <= 0 {
		last := lastMenuItem(p)
		if last >= 0 {
			r := p.ItemBounds(last)
			if r.Max.Y > box.Max.Y+1 {
				return fmt.Errorf("last item clipped maxY=%v menu=%v", r.Max.Y, box.Max.Y)
			}
		}
	}
	return nil
}

func lastMenuItem(p *widgets.PopupMenu) int {
	for i := len(p.Items) - 1; i >= 0; i-- {
		it := p.Items[i]
		if it != nil && !it.Separator {
			return i
		}
	}
	return -1
}

// ColorNear reports whether img at (x,y) is within slop of want (0–255).
func ColorNear(img *paintengine2d.Image, x, y, wr, wg, wb, slop int) bool {
	if img == nil || x < 0 || y < 0 || x >= img.Width || y >= img.Height {
		return false
	}
	r, g, b, _ := img.PremulAt(x, y)
	return abs(int(r)-wr) <= slop && abs(int(g)-wg) <= slop && abs(int(b)-wb) <= slop
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// RGB8 is Palette color as 8-bit channels (opaque).
func RGB8(c paintengine2d.Color) (r, g, b int) {
	return int(c.R*255 + 0.5), int(c.G*255 + 0.5), int(c.B*255 + 0.5)
}

// CountNonColor counts pixels in r whose RGB is farther than slop from col.
func CountNonColor(img *paintengine2d.Image, r paintengine2d.Rect, col paintengine2d.Color, slop int) int {
	if img == nil || r.Empty() {
		return 0
	}
	wr, wg, wb := RGB8(col)
	x0 := clampInt(int(r.Min.X), 0, img.Width)
	y0 := clampInt(int(r.Min.Y), 0, img.Height)
	x1 := clampInt(int(r.Max.X), 0, img.Width)
	y1 := clampInt(int(r.Max.Y), 0, img.Height)
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			cr, cg, cb, _ := img.PremulAt(x, y)
			if abs(int(cr)-wr) > slop || abs(int(cg)-wg) > slop || abs(int(cb)-wb) > slop {
				n++
			}
		}
	}
	return n
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// FieldColor is the Look's list/table body fill (blank-list detector).
func FieldColor(lk style.LookAndFeel) paintengine2d.Color {
	if lk == nil {
		return style.DarkLook().Palette().Field
	}
	return lk.Palette().Field
}

// HeaderFill is the Look's table header fill (SurfaceAlt).
func HeaderFill(lk style.LookAndFeel) paintengine2d.Color {
	if lk == nil {
		return style.DarkLook().Palette().SurfaceAlt
	}
	return lk.Palette().SurfaceAlt
}
