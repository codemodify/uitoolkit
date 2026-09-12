#!/usr/bin/env bash
# Render shipped PNG icon packs from upstream SVGs (rsvg-convert).
# Not required at runtime — the PNGs in this directory are committed.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
CACHE="${TMPDIR:-/tmp}/uitoolkit-icon-src"
mkdir -p "$CACHE"

if ! command -v rsvg-convert >/dev/null 2>&1; then
	echo "rsvg-convert is required (librsvg2-bin)" >&2
	exit 1
fi

fetch() {
	local url="$1" dest="$2"
	if [[ -s "$dest" ]]; then
		return 0
	fi
	local delay=1 i
	for i in 1 2 3 4 5; do
		if curl -fsSL --retry 3 --retry-delay 1 "$url" -o "$dest"; then
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

render_svg() {
	local svg="$1" dest="$2" px="$3"
	rsvg-convert -w "$px" -h "$px" --keep-image-data -o "$dest" "$svg"
}

write_sources() {
	local dir="$1"
	local file="$dir/SOURCES.txt"
	shift
	: >"$file"
	printf '%s\n' "$@" >>"$file"
}

# --- Lucide (ISC) https://github.com/lucide-icons/lucide ---
# Pin: v0.544.0
LUCIDE_BASE="https://raw.githubusercontent.com/lucide-icons/lucide/0.544.0/icons"
LUCIDE_LICENSE="https://raw.githubusercontent.com/lucide-icons/lucide/0.544.0/LICENSE"

# --- Phosphor regular (MIT) https://github.com/phosphor-icons/core ---
PHOS_BASE="https://raw.githubusercontent.com/phosphor-icons/core/v2.0.8/assets/regular"
PHOS_LICENSE="https://raw.githubusercontent.com/phosphor-icons/core/v2.0.8/LICENSE"

# --- Tabler outline (MIT) https://github.com/tabler/tabler-icons ---
TABLER_BASE="https://raw.githubusercontent.com/tabler/tabler-icons/v3.34.1/icons/outline"
TABLER_LICENSE="https://raw.githubusercontent.com/tabler/tabler-icons/v3.34.1/LICENSE"

# --- Heroicons 24 outline (MIT) https://github.com/tailwindlabs/heroicons ---
HERO_BASE="https://raw.githubusercontent.com/tailwindlabs/heroicons/v2.2.0/src/24/outline"
HERO_LICENSE="https://raw.githubusercontent.com/tailwindlabs/heroicons/v2.2.0/LICENSE"

# --- Material Symbols outlined 400 (Apache-2.0) ---
# marella/material-symbols is a clean SVG export of Google's set.
MAT_BASE="https://raw.githubusercontent.com/marella/material-symbols/af93fdcc77d4c64314c2944b9eec99c1bd911eaf/svg/400/outlined"
MAT_LICENSE="https://raw.githubusercontent.com/google/material-design-icons/master/LICENSE"

pack() {
	local name="$1" base="$2" license_url="$3"
	shift 3
	local dir="$ROOT/$name"
	mkdir -p "$dir"
	fetch "$license_url" "$dir/LICENSE"
	local action src dest svg
	while [[ $# -gt 0 ]]; do
		action="$1"
		src="$2"
		shift 2
		svg="$CACHE/${name}-${src}.svg"
		fetch "$base/${src}.svg" "$svg"
		render_svg "$svg" "$dir/${action}.png" 24
		render_svg "$svg" "$dir/${action}@2x.png" 48
	done
}

pack lucide "$LUCIDE_BASE" "$LUCIDE_LICENSE" \
	new file-plus \
	open folder-open \
	save save \
	cut scissors \
	copy copy \
	paste clipboard-paste \
	undo undo-2 \
	redo redo-2 \
	search search \
	info info \
	warning triangle-alert \
	error circle-x \
	question circle-question-mark

write_sources "$ROOT/lucide" \
	"Lucide Icons — ISC. https://lucide.dev  https://github.com/lucide-icons/lucide" \
	"Style: 24px outline (stroke). Pinned v0.544.0. Rendered with rsvg-convert." \
	"" \
	"new        file-plus" \
	"open       folder-open" \
	"save       save" \
	"cut        scissors" \
	"copy       copy" \
	"paste      clipboard-paste" \
	"undo       undo-2" \
	"redo       redo-2" \
	"search     search" \
	"info       info" \
	"warning    triangle-alert" \
	"error      circle-x" \
	"question   circle-question-mark"

pack phosphor "$PHOS_BASE" "$PHOS_LICENSE" \
	new file-plus \
	open folder-open \
	save floppy-disk \
	cut scissors \
	copy copy \
	paste clipboard-text \
	undo arrow-counter-clockwise \
	redo arrow-clockwise \
	search magnifying-glass \
	info info \
	warning warning \
	error x-circle \
	question question

write_sources "$ROOT/phosphor" \
	"Phosphor Icons — MIT. https://phosphoricons.com  https://github.com/phosphor-icons/core" \
	"Style: regular weight. Pinned v2.0.8. Rendered with rsvg-convert." \
	"" \
	"new        file-plus" \
	"open       folder-open" \
	"save       floppy-disk" \
	"cut        scissors" \
	"copy       copy" \
	"paste      clipboard-text" \
	"undo       arrow-counter-clockwise" \
	"redo       arrow-clockwise" \
	"search     magnifying-glass" \
	"info       info" \
	"warning    warning" \
	"error      x-circle" \
	"question   question"

pack tabler "$TABLER_BASE" "$TABLER_LICENSE" \
	new file-plus \
	open folder-open \
	save device-floppy \
	cut scissors \
	copy copy \
	paste clipboard \
	undo arrow-back-up \
	redo arrow-forward-up \
	search search \
	info info-circle \
	warning alert-triangle \
	error circle-x \
	question help

write_sources "$ROOT/tabler" \
	"Tabler Icons — MIT. https://tabler.io/icons  https://github.com/tabler/tabler-icons" \
	"Style: outline. Pinned v3.34.1. Rendered with rsvg-convert." \
	"" \
	"new        file-plus" \
	"open       folder-open" \
	"save       device-floppy" \
	"cut        scissors" \
	"copy       copy" \
	"paste      clipboard" \
	"undo       arrow-back-up" \
	"redo       arrow-forward-up" \
	"search     search" \
	"info       info-circle" \
	"warning    alert-triangle" \
	"error      circle-x" \
	"question   help"

pack heroicons "$HERO_BASE" "$HERO_LICENSE" \
	new document-plus \
	open folder-open \
	save arrow-down-on-square \
	cut scissors \
	copy document-duplicate \
	paste clipboard-document \
	undo arrow-uturn-left \
	redo arrow-uturn-right \
	search magnifying-glass \
	info information-circle \
	warning exclamation-triangle \
	error x-circle \
	question question-mark-circle

write_sources "$ROOT/heroicons" \
	"Heroicons — MIT. https://heroicons.com  https://github.com/tailwindlabs/heroicons" \
	"Style: 24px outline. Pinned v2.2.0. Rendered with rsvg-convert." \
	"" \
	"new        document-plus" \
	"open       folder-open" \
	"save       arrow-down-on-square" \
	"cut        scissors" \
	"copy       document-duplicate" \
	"paste      clipboard-document" \
	"undo       arrow-uturn-left" \
	"redo       arrow-uturn-right" \
	"search     magnifying-glass" \
	"info       information-circle" \
	"warning    exclamation-triangle" \
	"error      x-circle" \
	"question   question-mark-circle"

pack material-symbols "$MAT_BASE" "$MAT_LICENSE" \
	new note_add \
	open folder_open \
	save save \
	cut content_cut \
	copy content_copy \
	paste content_paste \
	undo undo \
	redo redo \
	search search \
	info info \
	warning warning \
	error error \
	question help

write_sources "$ROOT/material-symbols" \
	"Material Symbols — Apache-2.0. https://fonts.google.com/icons" \
	"https://github.com/google/material-design-icons" \
	"SVG export: https://github.com/marella/material-symbols @ af93fdc (outlined, weight 400)." \
	"Rendered with rsvg-convert." \
	"" \
	"new        note_add" \
	"open       folder_open" \
	"save       save" \
	"cut        content_cut" \
	"copy       content_copy" \
	"paste      content_paste" \
	"undo       undo" \
	"redo       redo" \
	"search     search" \
	"info       info" \
	"warning    warning" \
	"error      error" \
	"question   help"

echo "rendered packs in $ROOT"
