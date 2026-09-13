#!/usr/bin/env bash
# Render shipped PNG icon packs from pinned upstream SVGs (rsvg-convert).
# Not required at runtime — the PNGs in this directory are committed.
#
# Usage: ./icons/render.sh
# Needs: curl, tar, unzip, rsvg-convert (librsvg2-bin), python3
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
CACHE="${TMPDIR:-/tmp}/uitoolkit-icon-src"
mkdir -p "$CACHE"

if ! command -v rsvg-convert >/dev/null 2>&1; then
	echo "rsvg-convert is required (librsvg2-bin)" >&2
	exit 1
fi
if ! command -v python3 >/dev/null 2>&1; then
	echo "python3 is required" >&2
	exit 1
fi

fetch() {
	local url="$1" dest="$2"
	if [[ -s "$dest" ]]; then
		return 0
	fi
	local delay=1 i
	for i in 1 2 3 4 5; do
		if curl -fL --retry 3 --retry-delay 1 "$url" -o "$dest"; then
			if [[ -s "$dest" ]]; then
				return 0
			fi
		fi
		sleep "$delay"
		delay=$((delay * 2))
	done
	echo "download failed: $url" >&2
	return 1
}

# Pins (2026-09). Archives are cached under $CACHE.
LUCIDE_VER="1.45.0"
LUCIDE_ZIP="$CACHE/lucide-icons-${LUCIDE_VER}.zip"
LUCIDE_DIR="$CACHE/lucide-${LUCIDE_VER}/icons"
LUCIDE_URL="https://github.com/lucide-icons/lucide/releases/download/${LUCIDE_VER}/lucide-icons-${LUCIDE_VER}.zip"
LUCIDE_LICENSE_URL="https://raw.githubusercontent.com/lucide-icons/lucide/${LUCIDE_VER}/LICENSE"

PHOS_VER="v2.0.8"
PHOS_TGZ="$CACHE/phosphor-${PHOS_VER}.tar.gz"
PHOS_DIR="$CACHE/core-2.0.8/assets/regular"
PHOS_URL="https://github.com/phosphor-icons/core/archive/refs/tags/${PHOS_VER}.tar.gz"
PHOS_LICENSE="$CACHE/core-2.0.8/LICENSE"

TABLER_VER="v3.46.0"
TABLER_TGZ="$CACHE/tabler-${TABLER_VER}.tar.gz"
TABLER_DIR="$CACHE/tabler-icons-3.46.0/icons/outline"
TABLER_URL="https://github.com/tabler/tabler-icons/archive/refs/tags/${TABLER_VER}.tar.gz"
TABLER_LICENSE="$CACHE/tabler-icons-3.46.0/LICENSE"

HERO_VER="v2.2.0"
HERO_TGZ="$CACHE/heroicons-${HERO_VER}.tar.gz"
HERO_DIR="$CACHE/heroicons-2.2.0/optimized/24/outline"
HERO_URL="https://github.com/tailwindlabs/heroicons/archive/refs/tags/${HERO_VER}.tar.gz"
HERO_LICENSE="$CACHE/heroicons-2.2.0/LICENSE"

MAT_VER="v0.47.2"
MAT_TGZ="$CACHE/material-symbols-${MAT_VER}.tar.gz"
MAT_DIR="$CACHE/material-symbols-0.47.2/svg/400/outlined"
MAT_URL="https://github.com/marella/material-symbols/archive/refs/tags/${MAT_VER}.tar.gz"
MAT_LICENSE="$CACHE/material-symbols-0.47.2/LICENSE"

echo "fetching pinned archives…"
fetch "$LUCIDE_URL" "$LUCIDE_ZIP"
fetch "$LUCIDE_LICENSE_URL" "$CACHE/lucide-LICENSE"
fetch "$PHOS_URL" "$PHOS_TGZ"
fetch "$TABLER_URL" "$TABLER_TGZ"
fetch "$HERO_URL" "$HERO_TGZ"
fetch "$MAT_URL" "$MAT_TGZ"

if [[ ! -d "$LUCIDE_DIR" ]]; then
	mkdir -p "$CACHE/lucide-${LUCIDE_VER}"
	unzip -q -o "$LUCIDE_ZIP" -d "$CACHE/lucide-${LUCIDE_VER}"
fi
if [[ ! -d "$PHOS_DIR" ]]; then
	tar -xzf "$PHOS_TGZ" -C "$CACHE"
fi
if [[ ! -d "$TABLER_DIR" ]]; then
	tar -xzf "$TABLER_TGZ" -C "$CACHE"
fi
if [[ ! -d "$HERO_DIR" ]]; then
	tar -xzf "$HERO_TGZ" -C "$CACHE"
fi
if [[ ! -d "$MAT_DIR" ]]; then
	tar -xzf "$MAT_TGZ" -C "$CACHE"
fi

# dest lucide phosphor tabler heroicons material
# Official upstream names only — no hand-drawn SVGs.
MAPFILE="$CACHE/uitoolkit-icon-map.tsv"
cat >"$MAPFILE" <<'MAP'
new	file-plus	file-plus	file-plus	document-plus	note_add
open	folder-open	folder-open	folder-open	folder-open	folder_open
save	save	floppy-disk	device-floppy	arrow-down-on-square	save
cut	scissors	scissors	scissors	scissors	content_cut
copy	copy	copy	copy	document-duplicate	content_copy
paste	clipboard-paste	clipboard-text	clipboard	clipboard-document	content_paste
undo	undo-2	arrow-counter-clockwise	arrow-back-up	arrow-uturn-left	undo
redo	redo-2	arrow-clockwise	arrow-forward-up	arrow-uturn-right	redo
search	search	magnifying-glass	search	magnifying-glass	search
filter	list-filter	funnel	filter	funnel	filter_list
settings	settings	gear	settings	cog-6-tooth	settings
preferences	sliders-horizontal	sliders	adjustments-horizontal	adjustments-horizontal	tune
quit	log-out	sign-out	logout	arrow-right-on-rectangle	logout
close	x	x	x	x-mark	close
more	ellipsis	dots-three	dots	ellipsis-horizontal	more_horiz
chevron-up	chevron-up	caret-up	chevron-up	chevron-up	keyboard_arrow_up
chevron-down	chevron-down	caret-down	chevron-down	chevron-down	keyboard_arrow_down
chevron-left	chevron-left	caret-left	chevron-left	chevron-left	chevron_left
chevron-right	chevron-right	caret-right	chevron-right	chevron-right	chevron_right
arrow-up	arrow-up	arrow-up	arrow-up	arrow-up	arrow_upward
arrow-down	arrow-down	arrow-down	arrow-down	arrow-down	arrow_downward
arrow-left	arrow-left	arrow-left	arrow-left	arrow-left	arrow_back
arrow-right	arrow-right	arrow-right	arrow-right	arrow-right	arrow_forward
mail	mail	envelope	mail	envelope	mail
mail-open	mail-open	envelope-open	mail-opened	envelope-open	drafts
inbox	inbox	tray	inbox	inbox	inbox
send	send	paper-plane-tilt	send	paper-airplane	send
reply	reply	arrow-u-up-left	arrow-back	arrow-uturn-left	reply
reply-all	reply-all	arrow-bend-double-up-left	arrow-back-up-double	share	reply_all
forward	forward	arrow-bend-up-right	mail-forward	forward	forward
attach	paperclip	paperclip	paperclip	paper-clip	attach_file
download	download	download-simple	download	arrow-down-tray	download
upload	upload	upload-simple	upload	arrow-up-tray	upload
trash	trash	trash	trash	trash	delete
archive	archive	archive	archive	archive-box	archive
junk	ban	prohibit	ban	no-symbol	report
tag	tag	tag	tag	tag	label
star	star	star	star	star	star
flag	flag	flag	flag	flag	flag
pen	pen	pencil-simple	pencil	pencil	ink_pen
pencil	pencil	pencil	pencil	pencil	stylus_pencil
compose	square-pen	note-pencil	pencil-plus	pencil-square	edit_note
fetch	download	download-simple	download	arrow-down-tray	download
sync	refresh-cw	arrows-clockwise	refresh	arrow-path	sync
folder	folder	folder	folder	folder	folder
folder-plus	folder-plus	folder-plus	folder-plus	folder-plus	create_new_folder
user	user	user	user	user	person
users	users	users	users	users	group
bell	bell	bell	bell	bell	notifications
bell-off	bell-off	bell-slash	bell-off	bell-slash	notifications_off
eye	eye	eye	eye	eye	visibility
eye-off	eye-off	eye-slash	eye-off	eye-slash	visibility_off
lock	lock	lock	lock	lock-closed	lock
unlock	lock-open	lock-open	lock-open	lock-open	lock_open
check	check	check	check	check	check
x	x	x	x	x-mark	close
plus	plus	plus	plus	plus	add
minus	minus	minus	minus	minus	remove
edit	pencil	pencil-simple	edit	pencil	edit
trash-2	trash	trash	trash	trash	delete
info	info	info	info-circle	information-circle	info
warning	triangle-alert	warning	alert-triangle	exclamation-triangle	warning
error	circle-x	x-circle	circle-x	x-circle	error
question	circle-question-mark	question	help	question-mark-circle	help
help	circle-question-mark	question	help	question-mark-circle	help
home	house	house	home	home	home
calendar	calendar	calendar	calendar	calendar	calendar_today
clock	clock	clock	clock	clock	schedule
link	link	link	link	link	link
external-link	external-link	arrow-square-out	external-link	arrow-top-right-on-square	open_in_new
list	list	list	list	list-bullet	list
layout	layout-dashboard	layout	layout-dashboard	squares-2x2	dashboard
columns	columns-2	columns	layout-columns	view-columns	view_column
rows	rows-2	rows	layout-rows	bars-3	table_rows
table	table	table	table	table-cells	table_chart
cards	layout-grid	squares-four	layout-grid	squares-2x2	grid_view
sun	sun	sun	sun	sun	light_mode
moon	moon	moon	moon	moon	dark_mode
MAP

python3 - "$ROOT" "$MAPFILE" "$LUCIDE_DIR" "$PHOS_DIR" "$TABLER_DIR" "$HERO_DIR" "$MAT_DIR" \
	"$CACHE/lucide-LICENSE" "$PHOS_LICENSE" "$TABLER_LICENSE" "$HERO_LICENSE" "$MAT_LICENSE" \
	"$LUCIDE_VER" "$PHOS_VER" "$TABLER_VER" "$HERO_VER" "$MAT_VER" <<'PY'
import pathlib, shutil, subprocess, sys

root = pathlib.Path(sys.argv[1])
mapfile = pathlib.Path(sys.argv[2])
dirs = {
    "lucide": pathlib.Path(sys.argv[3]),
    "phosphor": pathlib.Path(sys.argv[4]),
    "tabler": pathlib.Path(sys.argv[5]),
    "heroicons": pathlib.Path(sys.argv[6]),
    "material-symbols": pathlib.Path(sys.argv[7]),
}
licenses = {
    "lucide": pathlib.Path(sys.argv[8]),
    "phosphor": pathlib.Path(sys.argv[9]),
    "tabler": pathlib.Path(sys.argv[10]),
    "heroicons": pathlib.Path(sys.argv[11]),
    "material-symbols": pathlib.Path(sys.argv[12]),
}
pins = {
    "lucide": sys.argv[13],
    "phosphor": sys.argv[14],
    "tabler": sys.argv[15],
    "heroicons": sys.argv[16],
    "material-symbols": sys.argv[17],
}
headers = {
    "lucide": (
        "Lucide Icons — ISC. https://lucide.dev  https://github.com/lucide-icons/lucide",
        f"Style: 24px outline (stroke). Pinned {pins['lucide']}. Rendered with rsvg-convert.",
    ),
    "phosphor": (
        "Phosphor Icons — MIT. https://phosphoricons.com  https://github.com/phosphor-icons/core",
        f"Style: regular weight. Pinned {pins['phosphor']}. Rendered with rsvg-convert.",
    ),
    "tabler": (
        "Tabler Icons — MIT. https://tabler.io/icons  https://github.com/tabler/tabler-icons",
        f"Style: outline. Pinned {pins['tabler']}. Rendered with rsvg-convert.",
    ),
    "heroicons": (
        "Heroicons — MIT. https://heroicons.com  https://github.com/tailwindlabs/heroicons",
        f"Style: 24px outline (stroke optically 2px). Pinned {pins['heroicons']}. Rendered with rsvg-convert.",
    ),
    "material-symbols": (
        "Material Symbols — Apache-2.0. https://fonts.google.com/icons",
        "https://github.com/google/material-design-icons",
        f"SVG export: https://github.com/marella/material-symbols {pins['material-symbols']} (outlined, weight 400).",
        "Rendered with rsvg-convert.",
    ),
}

rows = []
stems = []
for line in mapfile.read_text().splitlines():
    if not line.strip() or line.startswith("#"):
        continue
    parts = line.split("\t")
    if len(parts) != 6:
        raise SystemExit(f"bad map line: {line!r}")
    dest, lucide, phos, tabler, hero, mat = parts
    stems.append(dest)
    rows.append((dest, {
        "lucide": lucide,
        "phosphor": phos,
        "tabler": tabler,
        "heroicons": hero,
        "material-symbols": mat,
    }))

stems.append("no-icon")
(root / "STEMS.txt").write_text(
    "# Canonical PNG stems shipped in every premiere set (24×24 + @2x 48×48).\n"
    "# Regenerated by icons/render.sh — do not hand-edit.\n"
    + "\n".join(stems) + "\n"
)

NO_ICON_SVG = """<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#000000" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
  <rect x="3.5" y="3.5" width="17" height="17" rx="2" stroke-dasharray="3 2"/>
  <path d="M8 8l8 8M16 8l-8 8"/>
</svg>
"""

def prepare_svg(src: pathlib.Path, dest: pathlib.Path, thicken: bool) -> None:
    text = src.read_text(encoding="utf-8")
    text = text.replace("currentColor", "#000000")
    if thicken:
        text = text.replace('stroke-width="1.5"', 'stroke-width="2"')
        text = text.replace("stroke-width='1.5'", "stroke-width='2'")
    if "<svg" in text and "color=" not in text.split(">", 1)[0]:
        text = text.replace("<svg", '<svg color="#000000"', 1)
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(text, encoding="utf-8")

def render(svg: pathlib.Path, png: pathlib.Path, px: int) -> None:
    subprocess.run(
        ["rsvg-convert", "-w", str(px), "-h", str(px), "--keep-image-data", "-o", str(png), str(svg)],
        check=True,
    )

prep = pathlib.Path(sys.argv[2]).parent / "uitoolkit-icon-prep"
for name, srcdir in dirs.items():
    out = root / name
    out.mkdir(parents=True, exist_ok=True)
    for old in out.glob("*.png"):
        old.unlink()
    lic = licenses[name]
    if lic.is_file():
        shutil.copyfile(lic, out / "LICENSE")
    sources = list(headers[name]) + [""]
    for dest, mapping in rows:
        src_name = mapping[name]
        src = srcdir / f"{src_name}.svg"
        if not src.is_file():
            raise SystemExit(f"{name}: missing upstream SVG {src}")
        prepared = prep / name / f"{dest}.svg"
        prepare_svg(src, prepared, thicken=(name == "heroicons"))
        render(prepared, out / f"{dest}.png", 24)
        render(prepared, out / f"{dest}@2x.png", 48)
        sources.append(f"{dest:<16}{src_name}")
    no_svg = prep / name / "no-icon.svg"
    no_svg.parent.mkdir(parents=True, exist_ok=True)
    no_svg.write_text(NO_ICON_SVG, encoding="utf-8")
    render(no_svg, out / "no-icon.png", 24)
    render(no_svg, out / "no-icon@2x.png", 48)
    sources.append("no-icon         placeholder (missing stem)")
    (out / "SOURCES.txt").write_text("\n".join(sources) + "\n")
    print(f"rendered {name}: {len(rows) + 1} stems")

style_dir = root.parent / "style"
shutil.copyfile(root / "lucide" / "no-icon.png", style_dir / "no-icon.png")
shutil.copyfile(root / "lucide" / "no-icon@2x.png", style_dir / "no-icon@2x.png")

print(f"rendered packs in {root}")
PY
