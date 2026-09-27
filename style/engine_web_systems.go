//go:build theme_engine_all || theme_engine_web

package style

// shadcn/ui, Vercel's Geist and Linear as web packs (see engine_web.go and
// engine_web_packs.go).

// shadcnSpecs are shadcn/ui's classic new-york style (registry v4, 2025) in
// the neutral base colour: the OKLCH tokens converted to sRGB.

// --radius 10px: rounded-md 8 for controls, lg 10 for dialogs and the
// tab track, sm 6 for menu rows, xl 14 for cards; 4px check boxes.

// focus-visible: border-ring and a 3px ring of the ring colour at 50%
// outside the control.

// Tabs are a segmented TabsList: a muted track, the active trigger a
// raised card.

// shadow-md under menus, shadow-lg round dialogs; tips flat.

// Controls are 36px (h-9) inside the 3px ring's room.

// geistSpecs are Vercel's Geist design system (2023; the colour page's raw
// sRGB values of 2026).

// Materials: 6px controls, 12px menus, modals and cards.

// Focus: a 2px gap in the page colour, then a 2px ring; a focused
// input darkens its ring and takes a 3px wash beyond it instead.

// Rings plus soft drops: the menu's 0 16px 24px -8px, the tool
// tip's small shadow, the modal's larger one.

// Medium (36px) controls inside the 4px focus room; 36px menu rows.

// linearSpecs are Linear (2026): its app shell colours and, where the
// generator's output is not published, the text and border shades its site
// serves.

// 4px controls, 8px inputs and menus; compact rounded tabs.

// A 1px focus ring.

// 6px scroll thumbs that widen to 10.
