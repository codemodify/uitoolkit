package widget

// Tipper reports delayed hover help text.
type Tipper interface {
	Tooltip() string
}

// TooltipText walks from c to the root and returns the first non-empty tip.
func TooltipText(c Component) string {
	for n := c; n != nil; n = n.Parent() {
		if t, ok := n.(Tipper); ok {
			if s := t.Tooltip(); s != "" {
				return s
			}
		}
	}
	return ""
}

// TooltipHost is implemented by app.Window: a non-interactive hover bubble.
type TooltipHost interface {
	Host
	SetTooltip(c Component)
	Tooltip() Component
	HideTooltip()
}

// ShowTooltip places bubble on the window tooltip layer.
func ShowTooltip(from, bubble Component) bool {
	if from == nil || bubble == nil {
		return false
	}
	h := from.Host()
	if h == nil {
		return false
	}
	th, ok := h.(TooltipHost)
	if !ok {
		return false
	}
	bubble.SetHost(h)
	th.SetTooltip(bubble)
	return true
}

// HideTooltip dismisses the hover bubble, if any.
func HideTooltip(from Component) {
	if from == nil {
		return
	}
	h := from.Host()
	if h == nil {
		return
	}
	if th, ok := h.(TooltipHost); ok {
		th.HideTooltip()
	}
}
