package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// The viewer shows one photo over a dark stage: the chrome above and below it,
// the picture fitted and zoomed with the wheel around the pointer, and the
// filmstrip of the photos either side. Its state lives on app: lightbox,
// lightZoom, lightX, lightY.

// The viewer keeps its neutral dark stage in both appearances, so a photo is
// seen against the same black and its chrome stays out of the way.
var (
	viewerText     = ui.RGBA(238, 241, 246, 0.95)
	viewerMuted    = ui.RGBA(238, 241, 246, 0.62)
	viewerDisabled = ui.RGBA(238, 241, 246, 0.32)
	viewerBgColor  = ui.RGB(12, 12, 12)
	viewerChromeC  = ui.RGB(23, 23, 23)
	viewerLineC    = ui.RGB(42, 42, 42)
	viewerRatingC  = ui.RGB(241, 178, 74)
	viewerLoveC    = ui.RGB(242, 122, 155)
)

// openViewer shows a photo over the window's stage, full screen. A photo of
// the library is walked among the library's photos; a file opened from the
// file manager among the images of its folder.
func (a *app) openViewer(path string) {
	if p := a.lib.Get(path); p != nil {
		a.leaveOneOff()
		a.showInViewer(p)
	} else if p := a.oneOffPhoto(path); p != nil {
		a.showInViewer(p)
	} else {
		return
	}
	a.lightbox = true
	a.slideshow = false
	a.lightDragging = false
	a.lightMoved = false
	if a.win == nil || a.win.IsFullScreen() {
		return
	}
	win := a.win
	a.lightWasFull = true
	// Full screen runs the system's transition: asking for it from another
	// goroutine lets this frame draw first, instead of holding the window
	// while the desktop animates it.
	go win.SetFullScreen(true)
}

// showInViewer shows one photo of the ones being browsed, fitted and unzoomed.
func (a *app) showInViewer(p *Photo) {
	a.current = p.Path
	if a.externalFor != "" {
		// A one-off photo that joined the library is read from the library, so
		// its stars and its edit are the ones that count.
		if a.lib.Get(p.Path) != nil {
			a.external = nil
		} else {
			a.enrichOneOff(p)
			a.external = p
		}
	}
	a.lightZoom = 1
	a.lightZoomShown = 1
	a.lightX, a.lightY = 0, 0
	a.lightXShown, a.lightYShown = 0, 0
	a.lightAnimating = false
	a.wheelDelta = 0
	a.draftFor = ""
	a.filmReveal = true
}

// openLightbox shows a photo over the window's mask, and takes the window
// full screen to give it the whole display.
func (a *app) openLightbox(path string) { a.openViewer(path) }

// closeLightbox leaves the lightbox, and full screen with it when the
// lightbox asked for it. The folder a file was opened from is left with the
// lightbox, unless its photo joined the library by being favorited.
func (a *app) closeLightbox() {
	a.lightbox = false
	a.lightZoom = 1
	a.lightZoomShown = 1
	a.lightX, a.lightY = 0, 0
	a.lightXShown, a.lightYShown = 0, 0
	a.lightAnimating = false
	a.slideshow = false
	a.wheelDelta = 0
	a.filmReveal = false
	if a.externalFor != "" {
		if a.lib.Get(a.current) == nil {
			a.current = ""
		}
		a.leaveOneOff()
	}
	if !a.lightWasFull || a.win == nil {
		a.lightWasFull = false
		return
	}
	a.lightWasFull = false
	win := a.win
	go win.SetFullScreen(false)
}

// viewerShortcuts handles the keys that work while a photo shows over the
// stage.
func (a *app) viewerShortcuts(c *ui.Context) {
	if c.Shortcut(0, ui.KeyEscape) {
		a.closeLightbox()
	}
	if c.Shortcut(0, ui.KeyLeft) {
		a.step(-1)
	}
	if c.Shortcut(0, ui.KeyRight) {
		a.step(1)
	}
	if c.Shortcut(0, ui.KeySpace) {
		if a.slideshow {
			a.slideshowPaused = !a.slideshowPaused
		} else {
			a.step(1)
		}
	}
	if c.Shortcut(0, ui.Key0) {
		a.lightZoom, a.lightX, a.lightY = 1, 0, 0
	}
	if c.Shortcut(ui.Shift, ui.Key1) {
		a.lightZoom = 1
		a.lightX, a.lightY = 0, 0
	}
	if c.Shortcut(0, ui.KeyF) {
		if p := a.shownPhoto(); p != nil {
			a.setFavorite(p, !p.Favorite)
			a.save()
			a.viewKey = ""
			if p.Favorite {
				a.celebrateHere(c, ui.Element{})
			}
		}
	}
	if c.Shortcut(0, ui.KeyP) {
		a.slideshow = !a.slideshow
		a.slideshowPaused = false
	}
	for n := 1; n <= 5; n++ {
		n := n
		key := []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5}[n-1]
		if c.Shortcut(0, key) {
			if p := a.shownPhoto(); p != nil {
				if p.Stars == n {
					p.Stars = 0
				} else {
					p.Stars = n
				}
				a.save()
			}
		}
	}
}

// viewViewer builds the stage, the chrome above and below it and the
// filmstrip.
func (a *app) viewViewer(c *ui.Context) {
	p := a.shownPhoto()
	if p == nil {
		a.lightbox = false
		return
	}
	a.runSlideshow(c, p)
	w, h := c.Size()
	const barTop, barBottom float32 = 48, 112
	areaW := w - 2*48 - 32
	areaH := h - barTop - barBottom - 32

	ui.Column(c).Fill().Background(viewerBgColor).Children(func() {
		a.viewerTop(c, p)
		ui.Box(c).Height(1).Background(viewerLineC)

		// The stage: the photo in the middle, an arrow on either side.
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			a.viewerArrow(c, icons.chevronLeft, "‹ 上一张", "上一张 (←)", -1)
			area := ui.Box(c).Grow(1).Fill().Clip()
			area.HandleInput(func(ev ui.InputEvent) bool {
				if !a.lightboxInput(ev, areaW, areaH) {
					return false
				}
				c.Invalidate()
				return true
			})
			area.Children(func() {
				a.animateZoom(c)
				bmp, ready := a.lightboxBitmap(c, p, areaW, areaH)
				if !ready {
					ui.Box(c).Fill().Center().Children(func() {
						ui.Text(c, "载入中…").FontSize(12).TextColor(viewerMuted)
					})
					return
				}
				bw, bh := bmp.Size()
				zoom := a.lightZoomShown
				if zoom <= 0 {
					zoom = a.lightZoom
				}
				dispW, dispH := lightboxFit(bw, bh, areaW, areaH, zoom)
				a.lightDispW, a.lightDispH = dispW, dispH
				// Keep the target itself inside the photo's bounds, and draw the
				// eased pan inside the bounds of the eased size too.
				tw, th := lightboxFit(bw, bh, areaW, areaH, a.lightZoom)
				a.clampPan(tw, th, areaW, areaH)
				px, py := clampPanXY(a.lightXShown, a.lightYShown, dispW, dispH, areaW, areaH)
				ui.Box(c).Fill().Center().Children(func() {
					ui.Image(c, bmp).Size(dispW, dispH).Translate(float32(px), float32(py))
				})
			})
			a.viewerArrow(c, icons.chevronRight, "下一张 ›", "下一张 (→)", 1)
		})

		ui.Box(c).Height(1).Background(viewerLineC)
		a.viewerFooter(c, p, areaW, areaH)
	})
}

// viewerTop is the bar over the stage: the way out, what is shown, and what
// can be done with it without leaving.
func (a *app) viewerTop(c *ui.Context, p *Photo) {
	ui.Row(c).Height(titlebarHeight).Padding(0, 16).Gap(12).AlignItems(ui.Center).
		Background(viewerChromeC).Children(func() {
		if a.viewerIconButton(c, icons.x, "关闭 (esc)", false).Clicked() {
			a.closeLightbox()
		}
		ui.Column(c).Gap(1).Grow(1).Children(func() {
			ui.Text(c, p.Name).FontSize(13).Bold().TextColor(viewerText).SingleLine()
			ui.Textf(c, "%s · %d × %d", p.Taken().Format("2006-01-02 15:04"), p.Width, p.Height).
				FontSize(11).TextColor(viewerMuted).SingleLine()
		})
		if a.slideshow {
			state := "自动放映"
			if a.slideshowPaused {
				state = "已暂停"
			}
			ui.Text(c, state).FontSize(11).TextColor(viewerMuted)
		}
		ui.Text(c, a.lightboxCounter()).FontSize(11).TextColor(viewerMuted)

		// The rating, the heart and the other tools.
		ui.Row(c).Gap(0).AlignItems(ui.Center).Children(func() {
			for n := 1; n <= 5; n++ {
				n := n
				on := p.Stars >= n
				ic := icons.star
				col := viewerDisabled
				if on {
					ic, col = icons.starFilled, viewerRatingC
				}
				b := ui.ButtonBase(c).Size(controlCompact, controlCompact).Padding(0).Radius(4).
					Label(itoa(n) + " 星").Tooltip(itoa(n) + " 星")
				if b.Hovered() {
					b = b.Background(ui.RGBA(255, 255, 255, 0.08))
					if !on {
						col = viewerMuted
					}
				}
				b.Children(func() { glyph(c, ic, 16, col) })
				if b.Clicked() {
					if p.Stars == n {
						a.setStars(p, 0)
					} else {
						a.setStars(p, n)
					}
				}
			}
		})
		ui.Box(c).Width(1).Height(20).Background(viewerLineC).MarginX(4)
		heartIcon, heartCol := icons.heart, viewerMuted
		if p.Favorite {
			heartIcon, heartCol = icons.heartFilled, viewerLoveC
		}
		heart := a.viewerIconButtonTint(c, heartIcon, favoriteLabel(p.Favorite)+" (F)", heartCol)
		if heart.Clicked() {
			a.setFavorite(p, !p.Favorite)
			a.save()
			a.viewKey = ""
			if p.Favorite {
				a.celebrateHere(c, heart)
			}
		}
		if a.viewerIconButton(c, icons.download, "导出", false).Clicked() {
			if q := a.keepInLibrary(p); q != nil {
				a.sel.Clear()
				a.sel.Add(q.Path)
				a.openExport()
			}
		}
		if a.viewerIconButton(c, icons.sliders, "编辑", false).Clicked() {
			if q := a.keepInLibrary(p); q != nil {
				a.closeLightbox()
				a.openPhoto(q.Path)
				a.openEditPanel(q)
			}
		}
		ui.Box(c).Width(1).Height(20).Background(viewerLineC).MarginX(4)
		playIcon := icons.play
		playTip := "幻灯片放映 (P)"
		if a.slideshow {
			playIcon, playTip = icons.pause, "暂停放映 (P)"
		}
		if a.viewerIconButton(c, playIcon, playTip, a.slideshow).Clicked() {
			if !a.slideshow {
				a.slideshow = true
				a.slideshowPaused = false
				a.slideAt = time.Now().Add(3 * time.Second)
			} else {
				a.slideshowPaused = !a.slideshowPaused
			}
		}
	})
}

// runSlideshow steps to the next photo while the slideshow runs.
func (a *app) runSlideshow(c *ui.Context, p *Photo) {
	if !a.slideshow || a.slideshowPaused {
		return
	}
	now := c.Now()
	if a.slideAt.IsZero() {
		a.slideAt = now.Add(3 * time.Second)
	}
	if now.After(a.slideAt) {
		a.step(1)
		a.slideAt = now.Add(3 * time.Second)
	}
	c.After(250 * time.Millisecond)
}

// viewerArrow is one of the two arrows beside the stage.
func (a *app) viewerArrow(c *ui.Context, ic *ui.SVG, name, tip string, delta int) {
	ui.Box(c).Width(48).Shrink(0).Center().Children(func() {
		b := ui.ButtonBase(c).Size(32, 32).Radius(6).Padding(0).Label(name).Tooltip(tip)
		if b.Hovered() {
			b = b.Background(ui.RGBA(255, 255, 255, 0.08))
		}
		b.Children(func() { glyph(c, ic, 20, viewerMuted) })
		if b.Clicked() {
			a.step(delta)
		}
	})
}

// viewerIconButton is an icon button in the viewer's own colors.
func (a *app) viewerIconButton(c *ui.Context, ic *ui.SVG, tip string, on bool) ui.Element {
	col := viewerMuted
	bg := ui.Transparent
	if on {
		bg, col = ui.RGBA(255, 255, 255, 0.12), viewerText
	}
	b := ui.ButtonBase(c).Size(controlCompact, controlCompact).Radius(6).Padding(0).
		Label(tip).Tooltip(tip).Background(bg)
	if b.Hovered() {
		b = b.Background(ui.RGBA(255, 255, 255, 0.08))
		col = viewerText
	}
	return b.Children(func() { glyph(c, ic, 16, col) })
}

// viewerIconButtonTint is a viewer button whose icon carries a color.
func (a *app) viewerIconButtonTint(c *ui.Context, ic *ui.SVG, tip string, col ui.Color) ui.Element {
	b := ui.ButtonBase(c).Size(controlCompact, controlCompact).Radius(6).Padding(0).
		Label(tip).Tooltip(tip)
	if b.Hovered() {
		b = b.Background(ui.RGBA(255, 255, 255, 0.08))
	}
	return b.Children(func() { glyph(c, ic, 16, col) })
}

// viewerFooter is the bar under the stage: where the photo is from, the
// filmstrip of the photos either side, and the zoom.
func (a *app) viewerFooter(c *ui.Context, p *Photo, areaW, areaH float32) {
	ui.Row(c).Height(112).Padding(0, 16).Gap(16).AlignItems(ui.Center).
		Background(viewerChromeC).Children(func() {
		ui.Column(c).Gap(2).Width(200).Shrink(0).Children(func() {
			title := strings.Join(p.Tags, " · ")
			if title == "" {
				title = p.Name
			}
			ui.Text(c, title).FontSize(12).TextColor(viewerText).SingleLine()
			sub := p.Taken().Format("2006 年 1 月 2 日")
			if !p.Edit.IsZero() {
				sub += " · 已编辑"
			}
			if a.externalFor != "" {
				if a.lib.Get(p.Path) == nil {
					sub += " · 库外照片"
				} else {
					sub += " · 已加入资料库"
				}
			}
			ui.Text(c, sub).FontSize(11).TextColor(viewerMuted).SingleLine()
		})

		// The filmstrip is a virtual row: a spacer gives the scroll container
		// its width, and only the thumbnails in sight are built. A library of
		// ten thousand photos no longer creates ten thousand buttons per frame.
		photos := a.shownPhotos()
		const filmStep float32 = 72 // 64px thumbnail + 8px gap
		totalW := float32(len(photos))*filmStep - 8
		if totalW < 1 {
			totalW = 64
		}
		viewport := c.Resolve(a.filmScrollRef).Bounds().W
		if viewport <= 0 {
			viewport = 800 // the first frame has no committed bounds yet
		}
		if a.filmReveal {
			// Every switch of photo brings its thumbnail into the filmstrip, so
			// stepping through many photos never leaves the one shown behind.
			if i, ok := a.shownIndex(p.Path); ok {
				a.filmScroll.X = filmStripOffset(i, viewport, totalW)
			}
			a.filmReveal = false
		}
		left := a.filmScroll.X
		first := int(left/filmStep) - 1
		if first < 0 {
			first = 0
		}
		last := int((left+viewport)/filmStep) + 2
		if last > len(photos) {
			last = len(photos)
		}
		ui.ScrollHorizontal(c).Grow(1).TrackScroll(&a.filmScroll).Ref(&a.filmScrollRef).Children(func() {
			ui.Box(c).Width(totalW).Height(64).Ref(&a.filmBoxRef).Children(func() {
				for i := first; i < last; i++ {
					ph := photos[i]
					active := ph.Path == p.Path
					thumb := ui.ButtonBase(c).Absolute().Left(float32(i)*filmStep).Top(0).
						Size(64, 64).Padding(0).Radius(4).Clip().
						Border(1, ui.Transparent).Background(ui.RGB(16, 21, 28)).
						Label(ph.Name).Tooltip(ph.Name)
					if active {
						thumb = thumb.Border(1, a.paletteNow().accent)
					} else if thumb.Hovered() {
						thumb = thumb.Border(1, ui.RGBA(255, 255, 255, 0.35))
					}
					thumb.Children(func() {
						if bmp, ready := a.thumbs.Bitmap(ph.Path, ph.ModUnix, 160, ph.Edit); ready {
							ui.Image(c, bmp).Fit(ui.Cover).Fill()
						}
					})
					if thumb.Clicked() {
						a.showInViewer(ph)
					}
				}
			})
		})

		ui.Row(c).Gap(4).Shrink(0).AlignItems(ui.Center).Children(func() {
			if a.viewerIconButton(c, icons.minus, "缩小", false).Clicked() {
				a.zoomAround(1/1.15, areaW/2, areaH/2, areaW, areaH)
			}
			ui.Text(c, fmt.Sprintf("%.0f%%", a.lightZoom*100)).FontSize(11).
				TextColor(viewerMuted).Width(44).TextAlign(ui.Center)
			if a.viewerIconButton(c, icons.plus, "放大", false).Clicked() {
				a.zoomAround(1.15, areaW/2, areaH/2, areaW, areaH)
			}
			fit := ui.ButtonBase(c).Height(controlCompact).Padding(0, 10).Radius(6).
				Label("适应窗口").Tooltip("滚轮缩放 · 0 复位")
			fit = fit.Border(1, viewerLineC)
			if fit.Hovered() {
				fit = fit.Background(ui.RGBA(255, 255, 255, 0.08))
			}
			fit.Children(func() { ui.Text(c, "适应窗口").FontSize(12).TextColor(viewerText) })
			if fit.Clicked() {
				a.lightZoom, a.lightX, a.lightY = 1, 0, 0
			}
		})
	})
}

// filmStripOffset is where the filmstrip scrolls to so the thumbnail at i
// shows, as near the middle of it as the two ends allow.
func filmStripOffset(i int, viewport, totalW float32) float32 {
	const filmStep, filmThumb float32 = 72, 64
	x := float32(i)*filmStep + (filmThumb-viewport)/2
	if max := totalW - viewport; x > max {
		x = max
	}
	return max(0, x)
}

// lightboxInput applies one input event of the picture area: the wheel and
// a pinch zoom around the pointer, a drag pans, and a click on the mask
// closes the lightbox.
func (a *app) lightboxInput(ev ui.InputEvent, areaW, areaH float32) bool {
	switch ev.Kind {
	case ui.InputZoom:
		a.zoomAround(float64(ev.Scale), ev.X, ev.Y, areaW, areaH)
		return true
	case ui.InputScroll:
		// A mouse wheel is a notch (non-precise). Some high-resolution mice
		// report precise pixel deltas, so those are accumulated until one
		// notch's worth; a touchpad's momentum events are left alone. Ctrl/⌘
		// with the wheel zooms at once, as viewers usually do.
		if ev.Momentum {
			return false
		}
		dy := float64(ev.DY)
		if math.Abs(dy) < 0.1 || math.Abs(float64(ev.DX)) > math.Abs(dy)*1.5 {
			return false
		}
		if !ev.Precise {
			a.wheelDelta = 0
			if dy < 0 {
				a.zoomAround(1.15, ev.X, ev.Y, areaW, areaH)
			} else {
				a.zoomAround(1/1.15, ev.X, ev.Y, areaW, areaH)
			}
			return true
		}
		a.wheelDelta += dy
		mods := ev.Mods & (ui.Cmd | ui.Ctrl)
		threshold := 24.0
		if mods != 0 {
			threshold = 1
		}
		if math.Abs(a.wheelDelta) < threshold {
			return true
		}
		d := a.wheelDelta
		a.wheelDelta = 0
		factor := math.Exp(-d / 120)
		if factor > 1.35 {
			factor = 1.35
		}
		if factor < 1/1.35 {
			factor = 1 / 1.35
		}
		a.zoomAround(factor, ev.X, ev.Y, areaW, areaH)
		return true
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		// A drag starts where the picture is, not where an easing zoom is
		// heading.
		a.settleZoom()
		a.lightDragging = true
		a.lightMoved = false
		a.lightDragX, a.lightDragY = ev.X, ev.Y
		return true
	case ui.InputPointerMove:
		if !a.lightDragging {
			return false
		}
		dx, dy := ev.X-a.lightDragX, ev.Y-a.lightDragY
		if math.Abs(float64(dx))+math.Abs(float64(dy)) > 2 {
			a.lightMoved = true
		}
		a.lightX += float64(dx)
		a.lightY += float64(dy)
		a.lightXShown, a.lightYShown = a.lightX, a.lightY
		a.lightDragX, a.lightDragY = ev.X, ev.Y
		return true
	case ui.InputPointerUp:
		if !a.lightDragging {
			return false
		}
		a.lightDragging = false
		if !a.lightMoved && a.lightZoom <= 1 {
			// A click on the mask, away from the photo, closes it.
			if !insidePhoto(ev.X, ev.Y, areaW, areaH, a.lightDispW, a.lightDispH, a.lightX, a.lightY) {
				a.closeLightbox()
			}
		}
		return true
	}
	return false
}

// zoomAround scales the photo by factor, keeping the point under (x, y) of
// the picture area where it is, as a photograph's viewer does.
func (a *app) zoomAround(factor float64, x, y, areaW, areaH float32) {
	if a.lightZoom <= 0 {
		a.lightZoom = 1
	}
	z := a.lightZoom * factor
	if z > lightboxMaxZoom {
		z = lightboxMaxZoom
	}
	if z < lightboxMinZoom {
		z = lightboxMinZoom
	}
	if z == a.lightZoom {
		return
	}
	if !a.lightAnimating {
		a.lightAnimLast = time.Now()
	}
	a.lightAnimating = true
	// The pointer, relative to the middle of the area the photo fills.
	cx := float64(x) - float64(areaW)/2
	cy := float64(y) - float64(areaH)/2
	ratio := z / a.lightZoom
	a.lightX = cx + (a.lightX-cx)*ratio
	a.lightY = cy + (a.lightY-cy)*ratio
	a.lightZoom = z
	if z <= 1 {
		// Smaller than the area: there is nothing to pan.
		a.lightX, a.lightY = 0, 0
	}
}

// lightboxMinZoom is how much smaller than its fit a photo can be shown.
const lightboxMinZoom = 0.1

// lightboxMaxZoom is how far a photo can be enlarged.
const lightboxMaxZoom = 8

// lightboxFit is the size a photo takes when it is fitted into an area, at
// the zoom the lightbox has.
func lightboxFit(photoW, photoH int, areaW, areaH float32, zoom float64) (float32, float32) {
	if photoW <= 0 || photoH <= 0 || areaW <= 0 || areaH <= 0 {
		return 0, 0
	}
	f := math.Min(float64(areaW)/float64(photoW), float64(areaH)/float64(photoH))
	return float32(float64(photoW) * f * zoom), float32(float64(photoH) * f * zoom)
}

// clampPan keeps the photo from being dragged off the area: an edge never
// comes in past the middle of the window.
func (a *app) clampPan(dispW, dispH float32, areaW, areaH float32) {
	maxX := float64(max(dispW-areaW, 0)) / 2
	maxY := float64(max(dispH-areaH, 0)) / 2
	a.lightX = clamp(a.lightX, -maxX, maxX)
	a.lightY = clamp(a.lightY, -maxY, maxY)
}

// clampPanXY keeps a pan inside the area for a shown size.
func clampPanXY(x, y float64, dispW, dispH, areaW, areaH float32) (float64, float64) {
	maxX := float64(max(dispW-areaW, 0)) / 2
	maxY := float64(max(dispH-areaH, 0)) / 2
	return clamp(x, -maxX, maxX), clamp(y, -maxY, maxY)
}

// settleZoom stops an easing zoom where it is, for a drag or a step.
func (a *app) settleZoom() {
	if !a.lightAnimating {
		return
	}
	a.lightAnimating = false
	a.lightZoom = a.lightZoomShown
	a.lightX, a.lightY = a.lightXShown, a.lightYShown
}

// animateZoom springs the shown zoom and pan toward their targets. The
// spring keeps its velocity between wheel events, so a quick series of
// notches builds one continuous, elastic motion.
func (a *app) animateZoom(c *ui.Context) {
	if !a.lightAnimating {
		a.lightZoomShown = a.lightZoom
		a.lightXShown, a.lightYShown = a.lightX, a.lightY
		a.lightZoomVel, a.lightXVel, a.lightYVel = 0, 0, 0
		return
	}
	now := time.Now()
	dt := now.Sub(a.lightAnimLast).Seconds()
	a.lightAnimLast = now
	z, zv := springStep(a.lightZoomShown, a.lightZoom, a.lightZoomVel, dt, zoomSpringK, zoomSpringC)
	x, xv := springStep(a.lightXShown, a.lightX, a.lightXVel, dt, zoomSpringK, zoomSpringC)
	y, yv := springStep(a.lightYShown, a.lightY, a.lightYVel, dt, zoomSpringK, zoomSpringC)
	a.lightZoomShown, a.lightZoomVel = z, zv
	a.lightXShown, a.lightXVel = x, xv
	a.lightYShown, a.lightYVel = y, yv
	if springAtRest(z, a.lightZoom, zv) && springAtRest(x, a.lightX, xv) && springAtRest(y, a.lightY, yv) {
		a.lightZoomShown = a.lightZoom
		a.lightXShown, a.lightYShown = a.lightX, a.lightY
		a.lightZoomVel, a.lightXVel, a.lightYVel = 0, 0, 0
		a.lightAnimating = false
		return
	}
	c.After(16 * time.Millisecond)
}

// clamp keeps v within lo..hi.
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// insidePhoto reports whether (x, y) lies on the photo as it is drawn.
func insidePhoto(x, y, areaW, areaH, dispW, dispH float32, panX, panY float64) bool {
	cx, cy := areaW/2+float32(panX), areaH/2+float32(panY)
	return x >= cx-dispW/2 && x <= cx+dispW/2 && y >= cy-dispH/2 && y <= cy+dispH/2
}

// lightboxBitmap is the picture the lightbox shows at the zoom it has, and
// asks for the pictures either side so stepping shows one at once.
func (a *app) lightboxBitmap(c *ui.Context, p *Photo, areaW, areaH float32) (*ui.Bitmap, bool) {
	want := previewFor(areaW, areaH, a.lightZoom)
	a.prefetchNeighbors(p, want)
	return a.photoBitmap(c, p, want)
}

// photoBitmap is the picture a view shows for a photo: the frame of an
// animated GIF that is due, or the still the cache has.
func (a *app) photoBitmap(c *ui.Context, p *Photo, want int) (*ui.Bitmap, bool) {
	if b := a.gifFrame(c, p, want); b != nil {
		return b, true
	}
	return a.previewBitmap(p, want)
}

// lightboxCounter says which photo this is among the ones being browsed.
func (a *app) lightboxCounter() string {
	if i, ok := a.shownIndex(a.current); ok {
		return fmt.Sprintf("%d / %d", i+1, len(a.shownPhotos()))
	}
	return ""
}

// oneUp shows one photo alone with the editor beside it: what a right click's
// "查看详情" opens, and where an edit happens.
func (a *app) oneUp(c *ui.Context) {
	p := a.lib.Get(a.current)
	if p == nil {
		if len(a.photos) == 0 {
			a.emptyLibrary(c)
			return
		}
		p = a.photos[0]
		a.current = p.Path
	}
	a.oneUpShortcuts(c)
	pal := a.paletteNow()
	ui.Column(c).Grow(1).Background(pal.stage).Children(func() {
		a.oneUpHeader(c, p)
		a.hairline(c, false)
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			a.oneUpPicture(c, p)
			if a.editOpen {
				a.editPanel(c, p)
			}
		})
	})
}

// oneUpShortcuts navigates between photos with the arrow keys.
func (a *app) oneUpShortcuts(c *ui.Context) {
	if c.Shortcut(0, ui.KeyLeft) && !c.FocusTakesText() {
		a.step(-1)
	}
	if c.Shortcut(0, ui.KeyRight) && !c.FocusTakesText() {
		a.step(1)
	}
}

// step moves to the photo before or after the one shown.
func (a *app) step(delta int) {
	photos := a.shownPhotos()
	if len(photos) == 0 {
		return
	}
	i, ok := a.shownIndex(a.current)
	if !ok {
		return
	}
	i = (i + delta + len(photos)) % len(photos)
	a.showInViewer(photos[i])
}

// oneUpHeader is the bar above the picture: where it is, how it is rated, and
// what the side panels show.
func (a *app) oneUpHeader(c *ui.Context, p *Photo) {
	pal := a.paletteNow()
	ui.Row(c).Height(titlebarHeight).Padding(0, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		if a.iconButton(c, icons.arrowLeft, "回到网格 (esc)", false).Clicked() {
			a.mode = 0
		}
		ui.Column(c).Gap(1).Grow(1).Children(func() {
			ui.Text(c, p.Name).FontSize(13).Bold().SingleLine()
			ui.Textf(c, "%s · %d × %d", p.Taken().Format("2006-01-02 15:04"), p.Width, p.Height).
				FontSize(11).TextColor(pal.textSecondary).SingleLine()
		})
		a.ratingRow(c, p, 15)
		heartIcon, heartCol := icons.heart, pal.textSecondary
		if p.Favorite {
			heartIcon, heartCol = icons.heartFilled, pal.favorite
		}
		heart := a.iconButtonTint(c, heartIcon, favoriteLabel(p.Favorite), heartCol)
		if heart.Clicked() {
			a.setFavorite(p, !p.Favorite)
			a.save()
			a.viewKey = ""
			if p.Favorite {
				a.celebrateHere(c, heart)
			}
		}
		ui.Box(c).Width(1).Height(20).Background(pal.borderSubtle).MarginX(4)
		if a.iconButton(c, icons.sliders, "编辑这张照片", a.editOpen).Clicked() {
			if a.editOpen && a.draftFor == p.Path {
				a.editOpen = false
				a.draftFor = ""
			} else {
				a.openEditPanel(p)
			}
		}
		if a.iconButton(c, icons.expand, "在全屏查看中打开", false).Clicked() {
			a.openViewer(p.Path)
		}
	})
}

// oneUpPicture draws the picture, fitted, or at its own size when zoomed.
func (a *app) oneUpPicture(c *ui.Context, p *Photo) {
	pal := a.paletteNow()
	w, h := c.Size()
	want := previewFor(w, h, 1)
	bmp, ready := a.photoBitmap(c, p, want)
	a.prefetchNeighbors(p, want)
	ui.Column(c).Grow(1).Fill().Center().Padding(24).Gap(12).Children(func() {
		if !ready {
			ui.Text(c, "载入中…").TextColor(pal.textSecondary)
		} else if a.zoom == 0 {
			ui.Image(c, bmp).Fit(ui.Contain).Grow(1)
		} else {
			ui.ScrollBoth(c).Grow(1).Children(func() {
				ui.Image(c, bmp)
			})
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if a.iconButton(c, icons.chevronLeft, "上一张 (←)", false).Clicked() {
				a.step(-1)
			}
			if a.iconButton(c, icons.chevronRight, "下一张 (→)", false).Clicked() {
				a.step(1)
			}
			label := "适应窗口"
			if a.zoom != 0 {
				label = "原始大小"
			}
			if a.actionButton(c, label, icons.expand).Clicked() {
				a.zoom = (a.zoom + 1) % 2
			}
		})
	})
}

// viewEdit is the edit the picture is drawn with: the draft while it is being
// edited, its own otherwise.
func (a *app) viewEdit(p *Photo) EditState {
	if a.draftFor == p.Path {
		return a.draft
	}
	return p.Edit
}

// openEditPanel starts editing a photo on a draft, so the edits show before
// they are applied.
func (a *app) openEditPanel(p *Photo) {
	a.draft = p.Edit.Clone()
	if !a.draft.HasCrop {
		// A photo that has not been cropped shows the whole frame, so the
		// crop sliders start at their full value rather than at zero.
		a.draft.CropW, a.draft.CropH = 1, 1
	}
	a.draftFor = p.Path
	a.editOpen = true
	a.status = ""
}

// applyEdit settles the draft into the photo's edit, as one step the user can
// undo.
func (a *app) applyEdit(p *Photo) {
	a.pushUndo(p)
	p.Edit = a.draft.Clone()
	a.draftFor = ""
	a.save()
	a.status = "已应用编辑"
}

// pushUndo remembers the current edit so it can be undone.
func (a *app) pushUndo(p *Photo) {
	p.Undo = append(p.Undo, p.Edit.Clone())
	if len(p.Undo) > 50 {
		p.Undo = p.Undo[len(p.Undo)-50:]
	}
	p.Redo = nil
}

// undoEdit steps back one edit, and reports whether it did.
func (a *app) undoEdit(p *Photo) bool {
	if len(p.Undo) == 0 {
		return false
	}
	p.Redo = append(p.Redo, p.Edit.Clone())
	p.Edit = p.Undo[len(p.Undo)-1]
	p.Undo = p.Undo[:len(p.Undo)-1]
	a.draftFor = ""
	a.save()
	return true
}

// redoEdit steps forward one edit again.
func (a *app) redoEdit(p *Photo) bool {
	if len(p.Redo) == 0 {
		return false
	}
	p.Undo = append(p.Undo, p.Edit.Clone())
	p.Edit = p.Redo[len(p.Redo)-1]
	p.Redo = p.Redo[:len(p.Redo)-1]
	a.draftFor = ""
	a.save()
	return true
}

// editPanel is the panel that crops, straightens, turns and colors photos.
func (a *app) editPanel(c *ui.Context, p *Photo) {
	pal := a.paletteNow()
	ui.Column(c).Width(300).Shrink(0).Padding(16).Gap(12).Background(pal.chrome).
		Border(1, pal.borderSubtle).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Text(c, "编辑").FontSize(15).Bold().Grow(1)
			if a.iconButton(c, icons.undo, "撤销 (⌘Z)", false).Clicked() {
				a.undoEdit(p)
				a.draft = p.Edit.Clone()
				a.draftFor = p.Path
			}
			if a.iconButton(c, icons.redo, "重做 (⇧⌘Z)", false).Clicked() {
				a.redoEdit(p)
				a.draft = p.Edit.Clone()
				a.draftFor = p.Path
			}
			if a.iconButton(c, icons.x, "关闭编辑器", false).Clicked() {
				a.draftFor = ""
				a.editOpen = false
			}
		})
		ui.Textf(c, "撤销 %d · 重做 %d", len(p.Undo), len(p.Redo)).FontSize(11).TextColor(pal.textSecondary)

		ui.Tabs(c, &a.editTab, "裁剪拉直", "旋转翻转", "色彩")
		switch a.editTab {
		case 1:
			a.rotateControls(c, p)
		case 2:
			a.colorControls(c, p)
		default:
			a.cropControls(c, p)
		}

		ui.Spacer(c).Grow(1)
		a.hairline(c, false)
		ui.Row(c).Gap(6).Children(func() {
			if a.actionButton(c, "还原", icons.undo).Clicked() {
				a.pushUndo(p)
				p.Edit = EditState{}
				a.draft = EditState{}
				a.draftFor = p.Path
				a.save()
			}
			ui.Spacer(c).Grow(1)
			if a.quietButton(c, "取消", nil).Clicked() {
				a.draftFor = ""
				a.editOpen = false
			}
			if a.primaryAction(c, "应用", icons.check).Clicked() {
				a.applyEdit(p)
			}
		})
		ui.Text(c, "编辑只记录在库里，原文件不被修改").FontSize(10).TextColor(pal.textDisabled)
	})
}

// cropControls crop, pan and straighten the draft.
func (a *app) cropControls(c *ui.Context, p *Photo) {
	pal := a.paletteNow()
	if a.draftFor != p.Path {
		a.draft = p.Edit.Clone()
		a.draftFor = p.Path
	}
	aspectName := "原始比例"
	for _, ar := range aspectRatios {
		if a.draft.HasCrop && a.draft.CropW > 0 && a.draft.CropH > 0 {
			shape := a.draft.CropW * float64(max(p.Width, 1)) / (a.draft.CropH * float64(max(p.Height, 1)))
			if ar.W != 0 && abs(shape-ar.W/ar.H) < 0.02 {
				aspectName = ar.Name
			}
		}
	}
	var names []string
	for _, ar := range aspectRatios {
		names = append(names, ar.Name)
	}
	a.sliderField(c, "比例", func() {
		if ui.Select(c, &aspectName, names).Changed() {
			for _, ar := range aspectRatios {
				if ar.Name == aspectName {
					x, y, w, h := cropForShape(p.Width, p.Height, ar)
					a.draft.CropX, a.draft.CropY, a.draft.CropW, a.draft.CropH = x, y, w, h
					a.draft.HasCrop = ar.W != 0
				}
			}
		}
	})
	a.sliderField(c, "宽度", func() {
		if ui.Slider(c, &a.draft.CropW, 0.15, 1).Changed() {
			a.draft.HasCrop = true
			a.draft.CropX = min(a.draft.CropX, 1-a.draft.CropW)
		}
	})
	a.sliderField(c, "高度", func() {
		if ui.Slider(c, &a.draft.CropH, 0.15, 1).Changed() {
			a.draft.HasCrop = true
			a.draft.CropY = min(a.draft.CropY, 1-a.draft.CropH)
		}
	})
	a.sliderField(c, "水平位置", func() {
		if ui.Slider(c, &a.draft.CropX, 0, 0.99).Changed() {
			a.draft.HasCrop = true
			a.draft.CropX = min(a.draft.CropX, 1-a.draft.CropW)
		}
	})
	a.sliderField(c, "垂直位置", func() {
		if ui.Slider(c, &a.draft.CropY, 0, 0.99).Changed() {
			a.draft.HasCrop = true
			a.draft.CropY = min(a.draft.CropY, 1-a.draft.CropH)
		}
	})
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "拉直").FontSize(11).TextColor(pal.textSecondary).Width(56)
		ui.Slider(c, &a.draft.Angle, -15, 15).Grow(1)
		ui.Textf(c, "%.1f°", a.draft.Angle).FontSize(11).TextColor(pal.textSecondary).Width(44)
	})
	if a.draft.Angle != 0 {
		a.draft.HasCrop = true
	}
}

// rotateControls turns and flips the draft.
func (a *app) rotateControls(c *ui.Context, p *Photo) {
	pal := a.paletteNow()
	if a.draftFor != p.Path {
		a.draft = p.Edit.Clone()
		a.draftFor = p.Path
	}
	ui.Row(c).Gap(8).Children(func() {
		if a.actionButton(c, "左转 90°", icons.rotateLeft).Grow(1).Clicked() {
			a.draft.Rotate = (a.draft.Rotate + 270) % 360
		}
		if a.actionButton(c, "右转 90°", icons.rotateRight).Grow(1).Clicked() {
			a.draft.Rotate = (a.draft.Rotate + 90) % 360
		}
	})
	ui.Row(c).Gap(8).Children(func() {
		if a.chip(c, "水平翻转", a.draft.FlipH).Clicked() {
			a.draft.FlipH = !a.draft.FlipH
		}
		if a.chip(c, "垂直翻转", a.draft.FlipV).Clicked() {
			a.draft.FlipV = !a.draft.FlipV
		}
	})
	ui.Textf(c, "当前：旋转 %d°%s%s", a.draft.Rotate, flippedText(a.draft.FlipH, " · 水平翻转"), flippedText(a.draft.FlipV, " · 垂直翻转")).
		FontSize(11).TextColor(pal.textSecondary).Wrap()
}

// colorControls changes the light and the color of the draft. The adjustments
// are rendered into the picture the cache makes, so the preview is the photo
// the export writes.
func (a *app) colorControls(c *ui.Context, p *Photo) {
	pal := a.paletteNow()
	if a.draftFor != p.Path {
		a.draft = p.Edit.Clone()
		a.draftFor = p.Path
	}
	a.valueSlider(c, "曝光", &a.draft.Exposure, -1, 1)
	a.valueSlider(c, "对比度", &a.draft.Contrast, -1, 1)
	a.valueSlider(c, "饱和度", &a.draft.Saturation, -1, 1)
	a.valueSlider(c, "色温", &a.draft.Temp, -1, 1)
	a.valueSlider(c, "暗角", &a.draft.Vignette, 0, 1)
	ui.Row(c).Gap(6).Children(func() {
		if a.chip(c, "重置色彩", false).Clicked() {
			a.draft.Exposure = 0
			a.draft.Contrast = 0
			a.draft.Saturation = 0
			a.draft.Temp = 0
			a.draft.Vignette = 0
		}
	})
	ui.Text(c, "预览与导出一致，改动只存在库里").FontSize(10).TextColor(pal.textDisabled).Wrap()
}

// sliderField is a labeled slider in the editor's panel.
func (a *app) sliderField(c *ui.Context, label string, fn func()) {
	pal := a.paletteNow()
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(pal.textSecondary)
		fn()
	})
}

// valueSlider is a slider of an adjustment between two bounds, with its value.
func (a *app) valueSlider(c *ui.Context, label string, value *float64, lo, hi float64) {
	pal := a.paletteNow()
	ui.Column(c).Gap(2).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(11).TextColor(pal.textSecondary).Grow(1)
			ui.Textf(c, "%+.0f", *value*100).FontSize(11).TextColor(pal.textPrimary)
		})
		ui.Slider(c, value, lo, hi)
	})
}

// flippedText is a suffix for the rotation summary.
func flippedText(on bool, text string) string {
	if on {
		return text
	}
	return ""
}

// humanSize writes a number of bytes for people.
func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return itoa(int(n)) + " B"
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

// abs is the absolute value of a float.
func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
