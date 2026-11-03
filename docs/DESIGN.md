# CarTrak design system

The app follows Google's Material design (Material 3 with the Expressive touches). It is built so a new design
release is a change to a few values, not a rewrite of 60 pages.

## Where things live

| File | What it holds |
|---|---|
| `internal/web/static/css/tokens.css` | **The values.** Radii, spacing, type sizes, motion curves, tonal fills, elevation. Edit here first. |
| `css/look.css` | The theme colours (`--bg`, `--card`, `--accent` ...), light and dark, and the seasonal colours. Also older page layouts. |
| `css/material.css` | **The components.** Buttons, fields, chips, cards, banners, sheets, switches, the bottom bar, press and ripple. Uses tokens only. |
| `css/design.css` | Shared page parts: the header with its back arrow, the fields with units, the status line. |
| `js/touchfeedback.js` | The ripple and the vibration (Display, Touch feedback). |
| `js/sheetdrag.js`, `js/focustrap.js` | Pulling a bottom sheet closed; keeping Tab inside an open sheet. |

`css/app.css`, `history.css`, `menu.css`, `responsive.css` are page layouts. Their corner radii and text sizes use the tokens.

## Rules for any change

1. **Use a token.** No raw radius, font size, curve or colour in `material.css` (a test enforces it). A new value goes in `tokens.css` first.
2. **Button kinds.** Filled is the one thing to do on the screen. `.sec` (tonal) is another way forward. `.outline` is a safe choice beside a filled one. `.text` is the quietest ("Not now", "Cancel"). `.danger` is only for what cannot be undone.
3. **Switch or box.** A checkbox that turns something on or off is a switch automatically. A checkbox in a list of things to pick from gets `class: 'check'` and stays a box.
4. **Sheets.** A bottom sheet is `.sheetbg.bottom` with a `.sheet.more` inside: call `dragToClose` and `trapFocus`, mark it `role: 'dialog'` and `aria-modal`.
5. **A card that is one link** is `class: 'card cardlink'`, not an inline style.
6. **New stylesheet or script?** Add it to the `SHELL` list in `static/sw.js` and, for a stylesheet, to `index.html` (a test checks the list).
7. Touch targets are at least 44 px; keep text at the token sizes.

## When Google releases a new design

1. **Read what changed** (m3.material.io and the Android Developers blog). Do not rely on a summary: some coverage of new Android versions is leaks and betas.
2. **Map it onto tokens.** Corner shapes change `--r-*`. Type changes `--t-*`. Motion changes `--spring` and `--decel`. Tonal surfaces change `--tonal`, `--tonal-hi` and the card tint in `material.css`. Elevation changes `--lift` and `--lift-hi`.
3. **Add new components** to `material.css` using tokens. Keep one definition per component; delete the old rule instead of adding an override.
4. **Run the checks:**
   - `go test ./internal/web` (the guard tests: tokens only, every token defined, every file cached).
   - The usual checks in `CLAUDE.md`.
   - Paste `docs/design-check.js` into the browser console at 360, 411 and 1100 px wide, in dark and light and with a seasonal theme: it reports sideways scroll, small tap targets and low contrast.
   - Look at the pages that changed most (Home, Trips, History, More, Notifications, a settings page) in both themes.
5. **Say what was and was not done** in the CHANGELOG and `docs/STATUS.md`. Real-phone feel, iPhone and TalkBack cannot be tested from a desktop.

## Known gaps (not yet Material)

- Text fields are plain outlined boxes with a label above: no floating label, no helper text.
- No top app bar that elevates on scroll (it would cost screen height), no floating action button, no wavy progress.
- The guided tour, Wrapped, the battery ring and the charts have their own looks.
- Seasonal themes replace the accent colour, so the same screen can look brown, green or red. Dynamic colour from the wallpaper is not possible on the web.
- Swipe-back is left to Android's own back gesture.
