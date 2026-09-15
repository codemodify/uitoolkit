#!/usr/bin/env python3
"""Reads the demo app through AT-SPI, the way a screen reader does.

Run by smoke.sh inside a private D-Bus session: it never touches the
desktop's accessibility bus."""
import sys
import time

import gi

gi.require_version("Atspi", "2.0")
from gi.repository import Atspi  # noqa: E402

R = Atspi.Role
problems = []


def check(ok, what):
    print(("ok   " if ok else "FAIL ") + what)
    if not ok:
        problems.append(what)


desk = Atspi.get_desktop(0)
app = None
for _ in range(100):
    for i in range(desk.get_child_count()):
        c = desk.get_child_at_index(i)
        if c is not None and c.get_name() == "demo":
            app = c
    if app:
        break
    time.sleep(0.1)
check(app is not None, "the app is registered on the desktop")
if app is None:
    sys.exit(1)
check(app.get_role() == R.APPLICATION, "the app's role is application")
check(app.get_toolkit_name() == "uitoolkit", "toolkit name")
win = app.get_child_at_index(0)
check(win.get_role() == R.FRAME and win.get_name() == "Gallery", "the window: frame 'Gallery'")

nodes = []


def walk(o, depth=0):
    nodes.append(o)
    for i in range(o.get_child_count()):
        walk(o.get_child_at_index(i), depth + 1)


walk(win)
roles = {o.get_role() for o in nodes}
print(len(nodes), "objects")
for r in (R.PUSH_BUTTON, R.CHECK_BOX, R.MENU_BAR, R.PAGE_TAB_LIST, R.PAGE_TAB, R.SLIDER, R.SPIN_BUTTON,
          R.ENTRY, R.PROGRESS_BAR, R.COMBO_BOX, R.STATUS_BAR):
    check(r in roles, "has a " + Atspi.role_get_name(r))

buttons = [o for o in nodes if o.get_role() == R.PUSH_BUTTON]
check(all(b.get_name() for b in buttons), "every push button has a name")
# Parent links agree with the children lists.
b = buttons[0]
parent = b.get_parent()
check(any(parent.get_child_at_index(i) == b for i in range(parent.get_child_count())), "parent and index agree")
st = b.get_state_set()
check(st.contains(Atspi.StateType.ENABLED) and st.contains(Atspi.StateType.SHOWING), "a button is enabled and showing")
ext = b.get_extents(Atspi.CoordType.WINDOW)
check(ext.width > 0 and ext.height > 0, "a button has a box (%dx%d)" % (ext.width, ext.height))
check(Atspi.Action.get_n_actions(b) >= 1 and Atspi.Action.get_action_name(b, 0) == "click", "a button clicks")

slider = next(o for o in nodes if o.get_role() == R.SLIDER)
cur, top = Atspi.Value.get_current_value(slider), Atspi.Value.get_maximum_value(slider)
check(top == 100 and 0 <= cur <= 100, "the slider reads %s of %s" % (cur, top))
check(slider.get_name() == "Volume", "the slider is named Volume")

entry = next(o for o in nodes if o.get_role() == R.ENTRY)
check(Atspi.Text.get_text(entry, 0, -1) == "Ada Lovelace", "the entry's text reads 'Ada Lovelace'")
check(Atspi.Text.get_character_count(entry) == 12, "the entry counts 12 characters")

# Act through AT-SPI: tick the check box whose name we know.
boxes = [o for o in nodes if o.get_role() == R.CHECK_BOX]
box = boxes[0]
before = box.get_state_set().contains(Atspi.StateType.CHECKED)
Atspi.Action.do_action(box, 0)
time.sleep(0.3)
after = box.get_state_set().contains(Atspi.StateType.CHECKED)
check(before != after, "toggling '%s' through AT-SPI changes its state" % box.get_name())

# Tabs: select the second one through its action.
tabs = [o for o in nodes if o.get_role() == R.PAGE_TAB]
Atspi.Action.do_action(tabs[1], 0)
time.sleep(0.3)
check(tabs[1].get_state_set().contains(Atspi.StateType.SELECTED), "a tab selected through AT-SPI")

# Focus: grabbing the entry's focus announces it, as Orca expects.
from gi.repository import GLib  # noqa: E402

got = []


def on_focus(ev):
    if ev.detail1 == 1:
        got.append(ev.source)


listener = Atspi.EventListener.new(on_focus)
listener.register("object:state-changed:focused")
Atspi.Component.grab_focus(entry)
ctx = GLib.MainContext.default()
for _ in range(100):
    ctx.iteration(False)
    if got:
        break
    time.sleep(0.02)
check(bool(got) and got[-1].get_role() == R.ENTRY, "focusing the entry announces it (object:state-changed:focused)")
check(entry.get_state_set().contains(Atspi.StateType.FOCUSED), "the entry reports focused")

print("%d problems" % len(problems))
sys.exit(1 if problems else 0)
