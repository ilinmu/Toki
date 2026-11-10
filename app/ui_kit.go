package main

import (
	"github.com/egoist/mygo/ui"
)

// The geometry of the shell, in DIPs, as the prototype lays it out.
const (
	titlebarHeight  float32 = 48
	sidebarWidth    float32 = 240
	inspectorWidth  float32 = 272
	statusbarHeight float32 = 28
	controlCompact  float32 = 28
	controlRegular  float32 = 32
)

// glyph draws one icon of the set in a color and at a size.
func glyph(c *ui.Context, ic *ui.SVG, size float32, col ui.Color) ui.Element {
	if ic == nil {
		return ui.Box(c)
	}
	return ui.Icon(c, ic).FontSize(size).TextColor(col)
}

// iconButton is a 28px square holding one icon: transparent until the pointer
// rests on it, then it takes the hover fill and the stronger text. on keeps it
// filled, as a toggle that is on does.
func (a *app) iconButton(c *ui.Context, ic *ui.SVG, tip string, on bool) ui.Element {
	p := a.paletteNow()
	col := p.textSecondary
	bg := ui.Transparent
	if on {
		bg = p.selected
		col = p.textPrimary
	}
	b := ui.ButtonBase(c).Size(controlCompact, controlCompact).Radius(6).Padding(0).Label(tip).Tooltip(tip)
	b = b.Background(bg)
	if b.Hovered() || b.FocusVisible() {
		b = b.Background(p.hover)
		col = p.textPrimary
	}
	return b.Children(func() { glyph(c, ic, 16, col) })
}

// iconButtonTint is an icon button whose icon is filled with a color, as a
// favorite or a rating is.
func (a *app) iconButtonTint(c *ui.Context, ic *ui.SVG, tip string, col ui.Color) ui.Element {
	p := a.paletteNow()
	b := ui.ButtonBase(c).Size(controlCompact, controlCompact).Radius(6).Padding(0).Label(tip).Tooltip(tip)
	if b.Hovered() || b.FocusVisible() {
		b = b.Background(p.hover)
	}
	return b.Children(func() { glyph(c, ic, 16, col) })
}

// actionButton is a regular control with an icon and a label: the secondary
// button of the prototype.
func (a *app) actionButton(c *ui.Context, label string, ic *ui.SVG) ui.Element {
	p := a.paletteNow()
	b := ui.ButtonBase(c).Height(controlRegular).Padding(0, 10).Gap(6).Radius(6).Label(label)
	b = b.Border(1, p.borderStrong).Background(p.elevated)
	if b.Hovered() || b.FocusVisible() {
		b = b.Background(p.hover)
	}
	return b.Children(func() {
		if ic != nil {
			glyph(c, ic, 15, p.textSecondary)
		}
		ui.Text(c, label).FontSize(13).TextColor(p.textPrimary)
	})
}

// primaryAction is the one button of a view that is filled with the accent.
func (a *app) primaryAction(c *ui.Context, label string, ic *ui.SVG) ui.Element {
	p := a.paletteNow()
	b := ui.ButtonBase(c).Height(controlRegular).Padding(0, 12).Gap(6).Radius(6).Label(label)
	b = b.Background(p.accent)
	if b.Hovered() {
		b = b.Background(p.accentHover)
	}
	return b.Children(func() {
		if ic != nil {
			glyph(c, ic, 15, p.accentContrast)
		}
		ui.Text(c, label).FontSize(13).Bold().TextColor(p.accentContrast)
	})
}

// quietButton is a borderless button, for a row of actions in a panel.
func (a *app) quietButton(c *ui.Context, label string, ic *ui.SVG) ui.Element {
	p := a.paletteNow()
	b := ui.ButtonBase(c).Height(controlCompact).Padding(0, 8).Gap(6).Radius(6).Label(label)
	if b.Hovered() || b.FocusVisible() {
		b = b.Background(p.hover)
	}
	return b.Children(func() {
		if ic != nil {
			glyph(c, ic, 14, p.textSecondary)
		}
		ui.Text(c, label).FontSize(12).TextColor(p.textSecondary)
	})
}

// chip is a filter of the toolbar: transparent until chosen, then outlined
// with the accent and filled as a selection is.
func (a *app) chip(c *ui.Context, label string, on bool) ui.Element {
	p := a.paletteNow()
	b := ui.ButtonBase(c).Height(controlCompact).Padding(0, 8).Radius(6).Label(label)
	border, bg, col := p.borderSubtle, ui.Transparent, p.textSecondary
	if on {
		border, bg, col = p.accent, p.selected, p.textPrimary
	}
	b = b.Border(1, border).Background(bg)
	if b.Hovered() && !on {
		b = b.Background(p.hover)
		col = p.textPrimary
	}
	return b.Children(func() { ui.Text(c, label).FontSize(13).TextColor(col) })
}

// navButton is a row of the sidebar: icon, name and count.
func (a *app) navButton(c *ui.Context, label string, ic *ui.SVG, count int, on bool) ui.Element {
	p := a.paletteNow()
	bg, col := ui.Transparent, p.textSecondary
	if on {
		bg, col = p.selected, p.textPrimary
	}
	b := ui.ButtonBase(c).Height(controlRegular).Padding(0, 8).Gap(8).Radius(6).Label(label)
	b = b.Background(bg)
	if b.Hovered() && !on {
		b = b.Background(p.hover)
		col = p.textPrimary
	}
	return b.Children(func() {
		glyph(c, ic, 16, col)
		ui.Text(c, label).FontSize(13).TextColor(col).Grow(1).SingleLine()
		if count >= 0 {
			ui.Text(c, itoa(count)).FontSize(11).TextColor(p.textSecondary)
		}
	})
}

// albumButton is a row of the sidebar's album list: a tint dot, the name and
// how many photos it holds.
func (a *app) albumButton(c *ui.Context, name string, tint string, count int, on bool) ui.Element {
	p := a.paletteNow()
	bg, col := ui.Transparent, p.textSecondary
	if on {
		bg, col = p.selected, p.textPrimary
	}
	b := ui.ButtonBase(c).Height(controlRegular).Padding(0, 8).Gap(8).Radius(6).Label(name)
	b = b.Background(bg)
	if b.Hovered() && !on {
		b = b.Background(p.hover)
		col = p.textPrimary
	}
	b.Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(p.tint(tint)).Shrink(0)
		ui.Text(c, name).FontSize(13).TextColor(col).Grow(1).SingleLine()
		ui.Text(c, itoa(count)).FontSize(11).TextColor(p.textSecondary)
	})
	return b
}

// sectionLabel is the small heading over a group of sidebar rows.
func (a *app) sectionLabel(c *ui.Context, label string) {
	p := a.paletteNow()
	ui.Text(c, label).FontSize(11).Bold().TextColor(p.textSecondary).Padding(0, 8)
}

// ratingRow draws the five stars that rate a photo; a second click on the star
// that is set clears the rating.
func (a *app) ratingRow(c *ui.Context, p *Photo, size float32) {
	pal := a.paletteNow()
	if size <= 0 {
		size = 16
	}
	ui.Row(c).Gap(0).AlignItems(ui.Center).Children(func() {
		for n := 1; n <= 5; n++ {
			n := n
			on := p.Stars >= n
			ic := icons.star
			if on {
				ic = icons.starFilled
			}
			col := pal.textDisabled
			if on {
				col = pal.rating
			}
			b := ui.ButtonBase(c).Size(controlCompact, controlCompact).Padding(0).Radius(4)
			b = b.Label(itoa(n) + " 星").Tooltip(itoa(n) + " 星")
			if b.Hovered() {
				b = b.Background(pal.hover)
				if !on {
					col = pal.textSecondary
				}
			}
			b = b.Children(func() { glyph(c, ic, size, col) })
			if b.Clicked() {
				if p.Stars == n {
					p.Stars = 0
				} else {
					p.Stars = n
				}
				a.save()
			}
		}
	})
}

// infoRow is one line of the inspector's metadata grid: a label and its value
// over a hairline.
func (a *app) infoRow(c *ui.Context, label, value string) {
	p := a.paletteNow()
	ui.Row(c).Gap(8).Padding(6, 0).AlignItems(ui.Start).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(p.textSecondary).Width(64).Shrink(0)
		ui.Text(c, value).FontSize(12).TextColor(p.textPrimary).Grow(1).Wrap()
	})
	ui.Box(c).Height(1).Background(p.borderSubtle)
}

// inspectorHeading is the title of a section of the inspector.
func (a *app) inspectorHeading(c *ui.Context, label string) {
	p := a.paletteNow()
	ui.Text(c, label).FontSize(11).Bold().TextColor(p.textSecondary)
}

// cardChip is a tag in a list of tags, removable when a photo is chosen.
func (a *app) cardChip(c *ui.Context, label string, removable bool) ui.Element {
	p := a.paletteNow()
	b := ui.ButtonBase(c).Height(24).Padding(0, 8).Gap(4).Radius(4).Label(label)
	b = b.Border(1, p.borderSubtle).Background(p.elevated)
	if b.Hovered() {
		b = b.Border(1, p.borderStrong).Background(p.hover)
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(p.textPrimary)
		if removable {
			glyph(c, icons.x, 11, p.textDisabled)
		}
	})
	return b
}

// hairline is the 1px rule the panels are separated with.
func (a *app) hairline(c *ui.Context, vertical bool) {
	p := a.paletteNow()
	if vertical {
		ui.Box(c).Width(1).Background(p.borderSubtle).Shrink(0)
		return
	}
	ui.Box(c).Height(1).Background(p.borderSubtle)
}
