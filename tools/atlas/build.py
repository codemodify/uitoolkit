#!/usr/bin/env python3
"""build.py writes the Theme Atlas page (out/index.html) from out/packs.tsv,
which render.sh writes, and template.html. ATLAS_COMMIT stamps the page."""
import html
import os
import subprocess

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.environ.get("ATLAS_OUT", os.path.join(HERE, "out"))
COMMIT = os.environ.get("ATLAS_COMMIT") or subprocess.run(
    ["git", "-C", HERE, "log", "-1", "--format=%h"], capture_output=True, text=True).stdout.strip()

packs = []
with open(os.path.join(OUT, "packs.tsv")) as f:
    for line in f:
        parts = line.rstrip("\n").split("\t")
        if len(parts) < 6:
            continue
        pid, year, lineage, engine, label, summary = parts[:6]
        packs.append(dict(id=pid, year=int(year), lineage=lineage, engine=engine, label=label, summary=summary))

# Desktop platforms first, then the web's and apps' looks in the order they
# first shipped, then the toolkit's own skins.
LANES = ["Windows", "Mac OS", "Unix", "Sun", "NeXT", "Window Maker", "Amiga", "IBM", "Be", "Qt", "KDE",
         "GNOME", "Red Hat", "Ubuntu", "Google", "Java"]
WEB_LANES = ["Solarized", "Gruvbox", "Dracula", "Atom", "Nord", "Tokyo Night", "Catppuccin", "Rosé Pine",
             "GitHub", "Vercel", "SourceGit", "shadcn/ui", "JetBrains", "Linear", "VS Code", "uitoolkit"]
FIRST_WEB = len(LANES)
LANES += WEB_LANES
LANE_NAME = {"Unix": "Unix · Motif / CDE", "GitHub": "GitHub · Primer", "Vercel": "Vercel · Geist",
             "Atom": "Atom · One", "JetBrains": "JetBrains · Islands", "uitoolkit": "uitoolkit · skins"}


def pending(p):
    # A pack still on the stock painter while its engine is written.
    return p["engine"] == "base" and p["year"] >= 2010


def is_default(p):
    return p["id"] == "metal-ocean"


def esc(s):
    return html.escape(s, quote=True)


# ---- timeline (swimlanes) -------------------------------------------------
Y0, Y1 = 1983, 2028
W, LEFT, RIGHT, TOP, LANE_H, GAP = 1000, 150, 16, 18, 26, 30
BOTTOM = TOP + LANE_H * len(LANES) + GAP
H = BOTTOM + 34


def xof(year):
    return LEFT + (year - Y0) / (Y1 - Y0) * (W - LEFT - RIGHT)


def yof(i):
    return TOP + i * LANE_H + (GAP if i >= FIRST_WEB else 0) + LANE_H / 2


svg = [f'<svg class="lanes" viewBox="0 0 {W} {H}" role="img" aria-labelledby="lanes-title">',
       '<title id="lanes-title">Theme packs by year and platform</title>']
gy = TOP + FIRST_WEB * LANE_H + GAP / 2 + 4
svg.append(f'<text class="group-label" x="{LEFT - 14}" y="{gy:.1f}" text-anchor="end">Web and app looks</text>')
svg.append(f'<line class="group-rule" x1="{LEFT}" x2="{W - RIGHT}" y1="{gy - 4:.1f}" y2="{gy - 4:.1f}"/>')
for i, lane in enumerate(LANES):
    y = yof(i)
    svg.append(f'<text class="lane-label" x="{LEFT - 14}" y="{y + 4:.1f}" text-anchor="end">{esc(LANE_NAME.get(lane, lane))}</text>')
    svg.append(f'<line class="lane-rule" x1="{LEFT}" x2="{W - RIGHT}" y1="{y:.1f}" y2="{y:.1f}"/>')
for yr in range(1985, 2026, 5):
    x = xof(yr)
    svg.append(f'<line class="tick" x1="{x:.1f}" x2="{x:.1f}" y1="{TOP - 6}" y2="{BOTTOM}"/>')
    svg.append(f'<text class="year" x="{x:.1f}" y="{H - 8}" text-anchor="middle">{yr}</text>')
total, seen = {}, {}
for p in packs:
    if p["lineage"] in LANES:
        key = (LANES.index(p["lineage"]), p["year"])
        total[key] = total.get(key, 0) + 1
for p in packs:
    if p["lineage"] not in LANES:
        continue
    i = LANES.index(p["lineage"])
    key = (i, p["year"])
    k = seen.get(key, 0)
    seen[key] = k + 1
    # Same year, same lane: fan out sideways, centred on the year.
    x = xof(p["year"]) + (k - (total[key] - 1) / 2) * 7.5
    cls = "dot pending" if pending(p) else ("dot default" if is_default(p) else "dot")
    svg.append(f'<a href="#p-{esc(p["id"])}"><circle class="{cls}" cx="{x:.1f}" cy="{yof(i):.1f}" r="4.5">'
               f'<title>{esc(p["label"])} · {p["year"]}</title></circle></a>')
svg.append("</svg>")
unplaced = sorted({p["lineage"] for p in packs if p["lineage"] not in LANES})
if unplaced:
    print("no timeline lane for:", ", ".join(unplaced))

# ---- cards ----------------------------------------------------------------
decades = {}
for p in packs:
    decades.setdefault(p["year"] // 10 * 10, []).append(p)

DECADE_NOTE = {
    1980: "The first desktops: the Mac's System 1, Workbench 1.3, OPEN LOOK, and NeXT's greys with black key-window titles.",
    1990: "Bevels everywhere: Windows 3.1 and 95, System 7 and Platinum, OS/2 Warp, BeOS, Motif and CDE on Unix, Window Maker, and Swing's Metal.",
    2000: "Gloss and gel: Aqua, Luna and Aero, brushed metal, Java's Ocean and Nimbus, and the Linux desktop grows up with Bluecurve, Keramik, Plastik, Clearlooks and Human.",
    2010: "Flat and modern: Metro and Windows 10, Fusion, Breeze, Adwaita, OS X Yosemite, Material and FlatLaf, while the editor palettes arrive: Solarized, Gruvbox, Dracula, Atom's One and Nord.",
    2020: "Today: libadwaita's Adwaita and GNOME 48's, Windows 11's Fluent, macOS Big Sur and Tahoe's Liquid Glass, Plasma 6's Breeze, Material 3 and Material 3 Expressive, the looks of the web and its tools — GitHub's Primer, Vercel's Geist, shadcn/ui, Linear, SourceGit, VS Code, JetBrains Islands, Catppuccin, Tokyo Night, Rosé Pine — and the toolkit's own skins.",
}

W_IMG, H_IMG = 569, 381  # the Settings preview panel (docs/settings.md, "Screenshot geometry")


def card(p):
    tags = []
    if is_default(p):
        tags.append('<span class="tag tag-default">Default today</span>')
    if pending(p):
        tags.append('<span class="tag tag-pending">Engine in progress</span>')
    kind = "skin" if p["engine"] == "skin" else "engine " + p["engine"]
    return f'''<article class="card" id="p-{esc(p["id"])}" data-lineage="{esc(p["lineage"])}">
  <button class="shot" type="button" data-id="{esc(p["id"])}" data-label="{esc(p["label"])}" aria-label="Open {esc(p["label"])} full size">
    <img src="previews/{esc(p["id"])}.png" width="{W_IMG}" height="{H_IMG}" loading="lazy" alt="The Settings preview window drawn in {esc(p["label"])}">
  </button>
  <div class="caption">
    <h3>{esc(p["label"])}</h3>
    <p class="meta">{p["year"]} · {esc(p["lineage"])} · {esc(kind)}</p>
    {"<p class='tags'>" + "".join(tags) + "</p>" if tags else ""}
    <p class="summary">{esc(p["summary"])}</p>
    <code class="cmd">UITK_THEME={esc(p["id"])}</code>
  </div>
</article>'''


sections = []
for d in sorted(decades):
    ps = decades[d]
    cards = "\n".join(card(p) for p in ps)
    sections.append(f'''<section class="decade" data-decade="{d}" aria-labelledby="d-{d}">
  <header class="decade-head">
    <h2 id="d-{d}" class="decade-num">{d}s</h2>
    <p class="decade-note"><span class="count" data-total="{len(ps)}">{len(ps)} packs</span>{esc(DECADE_NOTE.get(d, ""))}</p>
  </header>
  <div class="grid">
{cards}
  </div>
</section>''')

engines = sorted({p["engine"] for p in packs if p["engine"] not in ("base", "skin")})
skins = [p for p in packs if p["engine"] == "skin"]
chips = [f'<button type="button" class="chip" aria-pressed="true" data-filter="*">All <span>{len(packs)}</span></button>']
for lane in LANES:
    n = sum(1 for p in packs if p["lineage"] == lane)
    if n:
        chips.append(f'<button type="button" class="chip" aria-pressed="false" data-filter="{esc(lane)}">'
                     f'{esc(LANE_NAME.get(lane, lane))} <span>{n}</span></button>')

page = open(os.path.join(HERE, "template.html")).read()
page = (page.replace("{{TIMELINE}}", "\n".join(svg))
            .replace("{{SECTIONS}}", "\n".join(sections))
            .replace("{{CHIPS}}", "\n".join(chips))
            .replace("{{COUNT}}", str(len(packs)))
            .replace("{{ENGINES}}", str(len(engines)))
            .replace("{{SKINS}}", str(len(skins)))
            .replace("{{COMMIT}}", esc(COMMIT))
            .replace("{{SPAN}}", "{}–{}".format(min(p["year"] for p in packs), max(p["year"] for p in packs))))
open(os.path.join(OUT, "index.html"), "w").write(page)
print("wrote", os.path.join(OUT, "index.html"), "with", len(packs), "packs,", len(engines), "engines,", len(skins), "skins")
