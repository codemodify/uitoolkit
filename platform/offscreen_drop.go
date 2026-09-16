package platform

import "github.com/codemodify/paintengine2d"

// SimulateDrop queues a drop from another app on an offscreen surface (tests
// and screenshots): a drag over pos offering the data by type, then the
// drop. ReceiveDrop hands the data back.
func (o *Offscreen) SimulateDrop(pos paintengine2d.Point, data map[string][]byte) {
	var mimes []string
	for m := range data {
		mimes = append(mimes, m)
	}
	o.drop = data
	o.dropTaken = nil
	o.Inject(Event{Kind: EventDragMotion, Pos: pos, Mimes: mimes})
	o.Inject(Event{Kind: EventDrop, Pos: pos, Mimes: mimes})
}

// ReceiveDrop implements DropReceiver for simulated drops.
func (o *Offscreen) ReceiveDrop(mime string) ([]byte, bool) {
	d, ok := o.drop[mime]
	return d, ok
}

// FinishDrop implements DropReceiver; DropTaken reports how it ended. A
// drop that finished a drag of this surface's own also ends that drag.
func (o *Offscreen) FinishDrop(ok bool) {
	o.dropTaken = &ok
	o.drop = nil
	o.dropFinished(ok)
}

// DropTaken reports whether the last simulated drop was taken (nil: not
// finished yet).
func (o *Offscreen) DropTaken() *bool { return o.dropTaken }
