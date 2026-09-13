package platform

type stubStatusItem struct {
	opts StatusItemOptions
}

func newStubStatusItem(opts StatusItemOptions) StatusItem {
	return &stubStatusItem{opts: opts}
}

func (s *stubStatusItem) SetIcon(StatusIcon) error       { return nil }
func (s *stubStatusItem) SetTooltip(string) error        { return nil }
func (s *stubStatusItem) SetTitle(string) error          { return nil }
func (s *stubStatusItem) SetMenu([]StatusMenuItem) error { return nil }
func (s *stubStatusItem) Notify(Notification) error      { return nil }
func (s *stubStatusItem) Close() error                   { return nil }
func (s *stubStatusItem) Backend() string                { return "stub" }
func (s *stubStatusItem) Alive() bool                    { return false }
