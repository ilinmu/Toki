package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// palette holds the design tokens of the native UI. They are the tokens of
// the OpenDesign prototype (NATIVE_UI_SPEC.md) turned into Go values: a cold
// ink-blue neutral, one accent for selection and focus, and four colors that
// only ever mean one thing each.
type palette struct {
	dark bool

	stage    ui.Color // the picture stage, behind the grid
	chrome   ui.Color // title bar, sidebar, inspector
	elevated ui.Color // popovers, fields, cards over the stage
	hover    ui.Color
	selected ui.Color

	borderSubtle ui.Color
	borderStrong ui.Color

	textPrimary   ui.Color
	textSecondary ui.Color
	textDisabled  ui.Color

	accent         ui.Color
	accentHover    ui.Color
	accentContrast ui.Color
	rating         ui.Color
	favorite       ui.Color
	success        ui.Color
	danger         ui.Color

	imageMat ui.Color

	viewerBG     ui.Color
	viewerChrome ui.Color
	viewerLine   ui.Color

	railBorder ui.Color
	railTrack  ui.Color
	railTick   ui.Color
	railThumb  ui.Color

	tints [6]ui.Color
}

// albumTintNames are the six album tints, in the order new albums take them.
var albumTintNames = [6]string{"lake", "sage", "clay", "amber", "rose", "slate"}

// paletteFor returns the design tokens of one appearance.
func paletteFor(dark bool) *palette {
	if dark {
		return &palette{
			dark:           true,
			stage:          ui.Hex("#141a23"),
			chrome:         ui.Hex("#1a212b"),
			elevated:       ui.Hex("#222a36"),
			hover:          ui.Hex("#29323f"),
			selected:       ui.Hex("#303b4d"),
			borderSubtle:   ui.Hex("#28313d"),
			borderStrong:   ui.Hex("#414e60"),
			textPrimary:    ui.Hex("#eef1f6"),
			textSecondary:  ui.Hex("#a8b2c0"),
			textDisabled:   ui.Hex("#6d7888"),
			accent:         ui.Hex("#5b9bff"),
			accentHover:    ui.Hex("#7cb0ff"),
			accentContrast: ui.Hex("#0a1220"),
			rating:         ui.Hex("#f1b24a"),
			favorite:       ui.Hex("#f27a9b"),
			success:        ui.Hex("#46c08a"),
			danger:         ui.Hex("#ef5f67"),
			imageMat:       ui.Hex("#10151c"),
			viewerBG:       ui.Hex("#0c0c0c"),
			viewerChrome:   ui.Hex("#171717"),
			viewerLine:     ui.Hex("#2a2a2a"),
			railBorder:     ui.RGBA(255, 255, 255, 0.18),
			railTrack:      ui.RGBA(238, 241, 246, 0.27),
			railTick:       ui.RGBA(238, 241, 246, 0.42),
			railThumb:      ui.Hex("#eef1f6"),
			tints: [6]ui.Color{
				ui.Hex("#6fa8dc"), ui.Hex("#7fb08a"), ui.Hex("#d08a6a"),
				ui.Hex("#d9b36a"), ui.Hex("#c98a9a"), ui.Hex("#8fa3b8"),
			},
		}
	}
	return &palette{
		dark:           false,
		stage:          ui.Hex("#edf2f8"),
		chrome:         ui.Hex("#ffffff"),
		elevated:       ui.Hex("#ffffff"),
		hover:          ui.Hex("#e5ecf4"),
		selected:       ui.Hex("#d5e2f2"),
		borderSubtle:   ui.Hex("#dde5ee"),
		borderStrong:   ui.Hex("#bfcbda"),
		textPrimary:    ui.Hex("#141a23"),
		textSecondary:  ui.Hex("#55606e"),
		textDisabled:   ui.Hex("#929eac"),
		accent:         ui.Hex("#2f6fe4"),
		accentHover:    ui.Hex("#235fc9"),
		accentContrast: ui.Hex("#ffffff"),
		rating:         ui.Hex("#b37418"),
		favorite:       ui.Hex("#c33f63"),
		success:        ui.Hex("#1b7f4f"),
		danger:         ui.Hex("#d93a3f"),
		imageMat:       ui.Hex("#e4eaf2"),
		viewerBG:       ui.Hex("#0c0c0c"),
		viewerChrome:   ui.Hex("#171717"),
		viewerLine:     ui.Hex("#2a2a2a"),
		railBorder:     ui.RGBA(20, 26, 35, 0.14),
		railTrack:      ui.RGBA(20, 26, 35, 0.18),
		railTick:       ui.RGBA(20, 26, 35, 0.32),
		railThumb:      ui.Hex("#141a23"),
		tints: [6]ui.Color{
			ui.Hex("#3a78a8"), ui.Hex("#4e7d58"), ui.Hex("#a85f42"),
			ui.Hex("#96702e"), ui.Hex("#9c5568"), ui.Hex("#5a7189"),
		},
	}
}

// theme turns the tokens into the theme the widgets draw with: the same
// colors, the prototype's 6px controls, 4px half-step spacing and 13px text.
func (p *palette) theme() *ui.Theme {
	var t *ui.Theme
	if p.dark {
		t = ui.DarkTheme()
	} else {
		t = ui.LightTheme()
	}
	t.Dark = p.dark
	t.Background = p.stage
	t.Surface = p.chrome
	t.SurfaceHover = p.hover
	t.SurfacePressed = p.selected
	t.Border = p.borderSubtle
	t.Text = p.textPrimary
	t.TextMuted = p.textSecondary
	t.Accent = p.accent
	t.AccentHover = p.accentHover
	t.AccentPressed = p.accentHover
	t.AccentText = p.accentContrast
	// The accent keeps its place when the window is not the active one: the
	// prototype has no grayed-out look for a background window.
	t.InactiveAccent = p.accent
	t.Danger = p.danger
	t.Warning = p.rating
	t.Success = p.success
	t.Selection = p.selected
	t.Focus = p.accent
	t.FocusRingWidth = 2
	t.FocusRingOffset = 0
	t.Inverse = p.elevated
	t.InverseText = p.textPrimary
	t.Scrollbar = p.borderStrong
	t.ScrollbarWidth = 8
	t.Radius = 6
	t.Spacing = 4
	t.FontSize = 13
	t.TooltipDelay = 500 * time.Millisecond
	return t
}

// tint returns the color of a named album tint, or the first for a name the
// library does not know.
func (p *palette) tint(name string) ui.Color {
	for i, n := range albumTintNames {
		if n == name {
			return p.tints[i]
		}
	}
	return p.tints[0]
}

// applyTheme builds the frame's theme, from the preference when the user chose
// one and from the system's appearance otherwise, and keeps the tokens for the
// frame in a.pal.
func (a *app) applyTheme(c *ui.Context) *palette {
	// The first frame reads the desktop's appearance, before the app's own
	// theme replaces the frame's; after that the system's answer is gone.
	if !a.themeResolved {
		a.systemDark = c.Theme().Dark
		a.themeResolved = true
	}
	dark := a.systemDark
	if a.set.Theme == "dark" {
		dark = true
	} else if a.set.Theme == "light" {
		dark = false
	}
	p := paletteFor(dark)
	c.SetTheme(p.theme())
	a.pal = p
	return p
}

// paletteNow is the frame's palette, for the helpers that take no Context.
func (a *app) paletteNow() *palette {
	if a.pal == nil {
		a.pal = paletteFor(a.systemDark)
	}
	return a.pal
}

// toggleTheme switches between the dark and the light appearance, and
// remembers the choice.
func (a *app) toggleTheme() {
	if a.paletteNow().dark {
		a.set.Theme = "light"
	} else {
		a.set.Theme = "dark"
	}
	a.saveSettings()
}
