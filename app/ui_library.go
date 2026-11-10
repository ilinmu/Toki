package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// One card is a square picture over a 28px name bar, as the prototype draws
// it; the flow model places them, so the view draws only what is in sight.
const (
	cardLabelHeight float32 = 28
	cardGap         float32 = 8
	groupHeadHeight float32 = 26
	flowPaddingX    float32 = 24
)

// flowCard is where one photo shows in the flow.
type flowCard struct {
	p      *Photo
	x, y   float32
	w, h   float32
	imageH float32
}

// flowHeader is a date heading of the grid, and where it starts.
type flowHeader struct {
	label string
	count int
	y     float32
}

// railGroup is one date of the date rail: its heading, how many photos it
// holds and how far down the flow it starts.
type railGroup struct {
	label string
	count int
	y     float32
}

// flowModel is the layout of one frame of the library.
type flowModel struct {
	width, height float32
	cards         []flowCard
	headers       []flowHeader
	groups        []railGroup
}

// computeFlow places the photos in columns: a grid of squares under date
// headings, or a masonry of the photos at their own shapes, each in the
// shortest column.
func computeFlow(photos []*Photo, width float32, masonry bool, target float32, showNames bool) flowModel {
	labelH := cardLabelHeight
	if !showNames {
		labelH = 0
	}
	if width < 80 {
		width = 80
	}
	if target < 80 {
		target = 188
	}
	cols := int((width + cardGap) / (target + cardGap))
	if cols < 1 {
		cols = 1
	}
	cardW := (width - cardGap*float32(cols-1)) / float32(cols)
	m := flowModel{width: width}

	if masonry {
		colY := make([]float32, cols)
		for _, p := range photos {
			col := 0
			for i := 1; i < cols; i++ {
				if colY[i] < colY[col]-0.01 {
					col = i
				}
			}
			imageH := cardW / clampRatio(p)
			card := flowCard{
				p:      p,
				x:      float32(col) * (cardW + cardGap),
				y:      colY[col],
				w:      cardW,
				imageH: imageH,
				h:      imageH + labelH,
			}
			m.cards = append(m.cards, card)
			colY[col] = card.y + card.h + cardGap
		}
		var tallest float32
		for _, y := range colY {
			tallest = max(tallest, y)
		}
		if tallest > 0 {
			m.height = tallest - cardGap
		}
		// The date rail reads the flow by the scroll offset: a date sits where
		// its first photo does.
		seen := map[string]bool{}
		for _, card := range m.cards {
			label := dayLabel(card.p.Taken())
			if seen[label] {
				for i := range m.groups {
					if m.groups[i].label == label {
						m.groups[i].count++
					}
				}
				continue
			}
			seen[label] = true
			m.groups = append(m.groups, railGroup{label: label, count: 1, y: card.y})
		}
		sortGroupsByY(m.groups)
		return m
	}

	// The grid: days under headings, each day a wall of squares.
	type daySet struct {
		label  string
		photos []*Photo
	}
	var days []daySet
	for _, p := range photos {
		label := dayLabel(p.Taken())
		if n := len(days); n > 0 && days[n-1].label == label {
			days[n-1].photos = append(days[n-1].photos, p)
			continue
		}
		days = append(days, daySet{label: label, photos: []*Photo{p}})
	}
	cellBox := cardW + labelH
	var y float32
	for gi, day := range days {
		if gi > 0 {
			y += 32
		}
		m.headers = append(m.headers, flowHeader{label: day.label, y: y, count: len(day.photos)})
		y += groupHeadHeight + 12
		m.groups = append(m.groups, railGroup{label: day.label, y: y, count: len(day.photos)})
		for i, p := range day.photos {
			row, col := i/cols, i%cols
			m.cards = append(m.cards, flowCard{
				p:      p,
				x:      float32(col) * (cardW + cardGap),
				y:      y + float32(row)*(cellBox+cardGap),
				w:      cardW,
				imageH: cardW,
				h:      cellBox,
			})
		}
		rows := (len(day.photos) + cols - 1) / cols
		y += float32(rows) * (cellBox + cardGap)
	}
	if n := len(m.cards); n > 0 {
		last := m.cards[n-1]
		m.height = last.y + last.h
	}
	return m
}

// clampRatio is a photo's width over its height, kept from making a column
// card too tall or too flat.
func clampRatio(p *Photo) float32 {
	if p.Width <= 0 || p.Height <= 0 {
		return 1
	}
	r := float32(p.Width) / float32(p.Height)
	if r < 0.5 {
		return 0.5
	}
	if r > 2.2 {
		return 2.2
	}
	return r
}

// sortGroupsByY orders the rail's dates by where they start in the flow.
func sortGroupsByY(groups []railGroup) {
	for i := 1; i < len(groups); i++ {
		for j := i; j > 0 && groups[j].y < groups[j-1].y; j-- {
			groups[j], groups[j-1] = groups[j-1], groups[j]
		}
	}
}

// dayLabel names the day a photo was taken, as the prototype's headings do.
func dayLabel(t time.Time) string {
	now := time.Now()
	startOf := func(v time.Time) time.Time {
		return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, v.Location())
	}
	diff := int(startOf(now).Sub(startOf(t)).Hours() / 24)
	switch diff {
	case 0:
		return "今天"
	case 1:
		return "昨天"
	}
	weekdays := [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	if t.Year() == now.Year() {
		return fmt.Sprintf("%d月%d日 %s", int(t.Month()), t.Day(), weekdays[int(t.Weekday())])
	}
	return fmt.Sprintf("%d年%d月%d日", t.Year(), int(t.Month()), t.Day())
}

// thumbTarget is how wide a card is wanted, by the preference.
func (a *app) thumbTarget() float32 {
	switch a.set.ThumbSize {
	case 0:
		return 140
	case 2:
		return 228
	}
	return 180
}

// libraryView is the photo library: the view's heading, the filters, and the
// photos in the arrangement chosen.
func (a *app) libraryView(c *ui.Context) {
	p := a.paletteNow()
	winW, winH := c.Size()
	contentW := winW - sidebarWidth - 1
	if a.inspectorOpen {
		contentW -= inspectorWidth + 1
	}
	if contentW < 200 {
		contentW = 200
	}
	innerW := contentW - flowPaddingX*2
	if a.layout == 1 {
		// The date rail takes its own 24px at the right of the masonry.
		innerW -= 24
	}
	model := computeFlow(a.photos, innerW, a.layout == 1, a.thumbTarget(), a.set.ShowNames)

	ui.Column(c).Grow(1).Background(p.stage).Children(func() {
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			ui.Scroll(c).Grow(1).TrackScroll(&a.libScroll).Ref(&a.scrollRef).Children(func() {
				ui.Column(c).Padding(24, flowPaddingX).Children(func() {
					a.viewHeader(c)
					a.libraryToolbar(c)
					if len(a.photos) == 0 {
						a.emptyLibrary(c)
						return
					}
					scrollY := a.libScroll.Y
					viewport := winH
					ui.Box(c).Width(model.width).Height(model.height).Ref(&a.flowRef).Children(func() {
						if a.layout == 0 {
							for _, hd := range model.headers {
								if hd.y > scrollY+viewport || hd.y+groupHeadHeight < scrollY-200 {
									continue
								}
								a.groupHeading(c, hd, model.width)
							}
						}
						for _, card := range model.cards {
							if card.y > scrollY+viewport || card.y+card.h < scrollY-200 {
								continue
							}
							a.photoCard(c, card)
						}
					})
				})
			})
			// The date rail belongs to the masonry, as the design has it: a
			// single continuous flow is what it can read.
			if a.layout == 1 && len(model.groups) > 1 {
				a.dateRail(c, model.groups, winH)
			}
		})
	})
}

// viewHeader is the heading of the view: where it is, what it holds and what
// can be done with it.
func (a *app) viewHeader(c *ui.Context) {
	p := a.paletteNow()
	eyebrow, title, subtitle := a.scopeInfo()
	ui.Row(c).AlignItems(ui.End).Justify(ui.SpaceBetween).Gap(24).Children(func() {
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Text(c, eyebrow).FontSize(11).Bold().TextColor(p.textSecondary)
			ui.Text(c, title).FontSize(15).Bold().SingleLine()
			ui.Text(c, subtitle).FontSize(12).TextColor(p.textSecondary).SingleLine()
		})
		ui.Row(c).Gap(8).Shrink(0).Children(func() {
			if a.actionButton(c, "新建相册", icons.plus).Clicked() {
				a.albumName = ""
				a.albumDesc = ""
				a.newAlbumOpen = true
			}
			if (a.lightbox || len(a.photos) > 0) && a.primaryAction(c, "导入照片", icons.importFile).Clicked() {
				a.addFolder()
			}
		})
	})
	ui.Box(c).Height(24)
}

// scopeInfo names the place the library shows.
func (a *app) scopeInfo() (eyebrow, title, subtitle string) {
	n := len(a.photos)
	if a.mode == 1 {
		return "时间轴", "时间轴", fmt.Sprintf("%d 张照片 · 每行一张，最新的在上", n)
	}
	if q := strings.TrimSpace(a.query); q != "" {
		return "搜索", "“" + q + "”", fmt.Sprintf("%d 张匹配照片", n)
	}
	switch {
	case a.nav == "recent":
		return "图库", "最近添加", "过去 30 天进入图库的照片"
	case a.nav == "favorites":
		return "图库", "收藏", "被标记为喜欢的照片"
	case a.nav == "stars":
		return "图库", "星标照片", "打过星的照片"
	case a.nav == "edited":
		return "图库", "已编辑", "经过调整的照片"
	case strings.HasPrefix(a.nav, "album:"):
		if al := a.lib.Album(strings.TrimPrefix(a.nav, "album:")); al != nil {
			sub := al.Description
			if sub == "" {
				sub = fmt.Sprintf("%d 张照片", n)
			}
			return "相册", al.Name, sub
		}
		return "相册", "相册", fmt.Sprintf("%d 张照片", n)
	case strings.HasPrefix(a.nav, "tag:"):
		return "标签", strings.TrimPrefix(a.nav, "tag:"), fmt.Sprintf("%d 张照片", n)
	case strings.HasPrefix(a.nav, "folder:"):
		return "文件夹", filepath.Base(strings.TrimPrefix(a.nav, "folder:")), fmt.Sprintf("%d 张照片", n)
	case strings.HasPrefix(a.nav, "year:"):
		return "年份", strings.TrimPrefix(a.nav, "year:") + " 年", fmt.Sprintf("%d 张照片", n)
	}
	return "图库", "全部照片", fmt.Sprintf("%d 张照片 · 本地资料库", n)
}

// libraryToolbar is the row of filters and arrangements under the heading.
func (a *app) libraryToolbar(c *ui.Context) {
	p := a.paletteNow()
	ui.Row(c).Wrap().Gap(8).AlignItems(ui.Center).Padding(0, 0, 0, 16).Children(func() {
		filters := []struct{ id, label string }{
			{"all", "全部"}, {"favorite", "收藏"}, {"edited", "已编辑"}, {"rated", "四星以上"},
		}
		for _, f := range filters {
			f := f
			if a.chip(c, f.label, a.filter == f.id).Clicked() {
				a.filter = f.id
				a.viewKey = ""
				a.libScroll.ScrollTo(0, 0, ui.Smooth)
			}
		}
		ui.Box(c).Width(1).Height(20).Background(p.borderSubtle).MarginX(4)
		ui.Spacer(c).Grow(1)

		// The order, the arrangement and the size of the cards.
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			name := sortName(a.set.Sort)
			if ui.Select(c, &name, sortNames).Width(150).Tooltip("照片排序").Changed() {
				a.set.Sort = sortKey(name)
				a.saveSettings()
				a.viewKey = ""
			}
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "排列").FontSize(12).TextColor(p.textSecondary)
				if a.iconButton(c, icons.grid, "网格排列 (⌘1)", a.layout == 0 && a.mode == 0).Clicked() {
					a.mode, a.layout = 0, 0
					a.saveLayout()
				}
				if a.iconButton(c, icons.masonry, "瀑布流排列 (⌘2)", a.layout == 1 && a.mode == 0).Clicked() {
					a.mode, a.layout = 0, 1
					a.saveLayout()
				}
				if a.iconButton(c, icons.clock, "时间轴列表 (⌘3)", a.mode == 1).Clicked() {
					a.mode = 1
				}
			})
			if ui.Segmented(c, &a.set.ThumbSize, "小", "中", "大").Tooltip("缩略图尺寸").Changed() {
				a.saveSettings()
			}
		})
		ui.Textf(c, "%d 张", len(a.photos)).FontSize(12).TextColor(p.textSecondary)
	})
	ui.Box(c).Height(16)
}

// groupHeading is the date over a group of the grid.
func (a *app) groupHeading(c *ui.Context, hd flowHeader, width float32) {
	p := a.paletteNow()
	ui.Row(c).Absolute().Left(0).Top(hd.y).Width(width).Height(groupHeadHeight).
		AlignItems(ui.End).Children(func() {
		ui.Text(c, hd.label).FontSize(13).Bold().Grow(1)
		ui.Textf(c, "%d 张", hd.count).FontSize(11).TextColor(p.textSecondary)
	})
}

// photoCard draws one photo of the flow: the picture, its name, the heart
// that favors it and the mark that shows it is chosen.
func (a *app) photoCard(c *ui.Context, card flowCard) {
	p := a.paletteNow()
	ph := card.p
	chosen := a.sel.Has(ph.Path)
	rad := float32(4)

	box := ui.Box(c).Key(ph.Path).Absolute().Left(card.x).Top(card.y).
		Width(card.w).Height(card.h).Clip()
	labelH := float32(0)
	if a.set.ShowNames {
		labelH = cardLabelHeight
	}
	// The card keeps its own size: the picture over the name bar, both as tall
	// as the flow laid them out, so a cell never draws over the next.
	box.Children(func() {
		btn := ui.ButtonBase(c).FillWidth().Height(card.h).Padding(0).Gap(0).Radius(rad).Clip()
		border := ui.Transparent
		if chosen {
			border = p.accent
		}
		btn = btn.Background(p.chrome).Border(2, border)
		if btn.Hovered() && !chosen {
			btn = btn.Border(2, p.borderStrong)
		}
		btn = btn.Label(ph.Name)
		btn.Children(func() {
			ui.Column(c).Width(card.w - 4).Height(card.h - 4).Children(func() {
				// The picture, cropped to the card.
				pic := ui.Box(c).FillWidth().Height(card.h - 4 - labelH).Clip().Background(p.imageMat)
				pic.Children(func() {
					if bmp, ready := a.thumbs.Bitmap(ph.Path, ph.ModUnix, a.thumbPixels(), ph.Edit); ready {
						ui.Image(c, bmp).Fit(ui.Cover).FillWidth().Height(card.h - 4 - labelH)
					} else {
						ui.Box(c).Fill().Center().Children(func() {
							glyph(c, icons.image, 20, p.textDisabled)
						})
					}
				})
				hovered := box.Hovered() || box.FocusWithin()
				if a.set.ShowNames {
					ui.Row(c).Height(cardLabelHeight).Shrink(0).Padding(0, 8).Gap(8).
						AlignItems(ui.Center).Background(p.chrome).Children(func() {
						label := ui.Text(c, ph.Name).FontSize(12).SingleLine().Grow(1)
						if hovered || chosen {
							label.TextColor(p.textPrimary)
						} else {
							label.TextColor(p.textSecondary)
						}
						if !ph.Edit.IsZero() {
							ui.Text(c, "已编辑").FontSize(11).TextColor(p.textDisabled)
						} else if ph.Favorite {
							glyph(c, icons.heartFilled, 12, p.favorite)
						}
					})
				}
			})
		})

		if btn.Clicked() {
			a.choosePhoto(ph.Path, c.Modifiers()&ui.Cmd != 0 || c.Modifiers()&ui.Shift != 0)
		}
		if btn.DoubleClicked() {
			a.sel.Clear()
			a.sel.Add(ph.Path)
			a.current = ph.Path
			a.openViewer(ph.Path)
		}

		// The heart shows on the card that is chosen or under the pointer, and
		// always on one that is a favorite. The whole card counts as under the
		// pointer, so the heart stays while the pointer is on it.
		cardHovered := box.Hovered() || box.FocusWithin()
		showHeart := ph.Favorite || chosen || cardHovered
		if showHeart {
			heartLabel := "收藏这张照片"
			if ph.Favorite {
				heartLabel = "取消收藏"
			}
			heart := ui.ButtonBase(c).Absolute().Top(4).Right(4).Size(26, 26).Radius(4).
				Padding(0).Border(1, p.borderSubtle).Background(p.chrome).
				Label(heartLabel).Tooltip("收藏 (F)")
			if heart.Hovered() {
				heart = heart.Background(p.hover)
			}
			ic := icons.heart
			col := p.textSecondary
			if ph.Favorite {
				ic, col = icons.heartFilled, p.favorite
			}
			heart.Children(func() { glyph(c, ic, 14, col) })
			if heart.Clicked() {
				ph.Favorite = !ph.Favorite
				a.save()
				a.viewKey = ""
				if ph.Favorite {
					a.celebrate(c, heart, box.Bounds())
					c.Toast("已收藏 " + ph.Name)
				}
			}
		}

		// The mark of a chosen photo. It is a check box: a click adds the
		// photo to the choice or takes it out, and leaves the others chosen,
		// which is what choosing several photos is for.
		if chosen || cardHovered {
			label := "选择这张照片"
			if chosen {
				label = "取消选择"
			}
			mark := ui.ButtonBase(c).Absolute().Top(4).Left(4).Size(20, 20).Radius(4).
				Padding(0).Label(label).Tooltip(label).
				Border(1, p.borderStrong).Background(p.chrome)
			if chosen {
				mark = mark.Border(1, p.accent).Background(p.accent)
			} else if mark.Hovered() {
				mark = mark.Border(1, p.accent).Background(p.hover)
			}
			mark.Children(func() {
				if chosen {
					glyph(c, icons.check, 13, p.accentContrast)
				}
			})
			if mark.Clicked() {
				if a.sel.Has(ph.Path) {
					a.sel.Remove(ph.Path)
				} else {
					a.sel.Add(ph.Path)
				}
				a.current = ph.Path
				a.viewKey = ""
			}
		}
	})

	box.ContextMenu(func(m *ui.Menu) {
		if m.Item("全屏查看").Chosen() {
			a.sel.Clear()
			a.sel.Add(ph.Path)
			a.openViewer(ph.Path)
		}
		if m.Item("查看详情（带信息面板）").Chosen() {
			a.openPhoto(ph.Path)
		}
		m.Separator()
		if m.Item("收藏").Checked(ph.Favorite).Chosen() {
			ph.Favorite = !ph.Favorite
			a.save()
			a.viewKey = ""
		}
		for n := 1; n <= 5; n++ {
			if m.Item(starString(n)).Checked(ph.Stars == n).Chosen() {
				ph.Stars = n
				a.save()
			}
		}
		if m.Item("清除星标").Checked(ph.Stars == 0).Chosen() {
			ph.Stars = 0
			a.save()
		}
		m.Separator()
		if m.Item("在 Finder 中显示").Chosen() {
			revealInFinder(ph.Path)
		}
		if m.Item("移出库…").Chosen() {
			a.sel.Clear()
			a.sel.Add(ph.Path)
			a.forgetSelected()
		}
	})
}

// favoriteLabel is what the heart's tooltip says.
func favoriteLabel(on bool) string {
	if on {
		return "取消收藏"
	}
	return "收藏"
}

// choosePhoto chooses a photo of the grid, adding it to the choice when a
// modifier is held.
func (a *app) choosePhoto(path string, additive bool) {
	if additive {
		if a.sel.Has(path) {
			a.sel.Remove(path)
		} else {
			a.sel.Add(path)
		}
	} else {
		a.sel.Clear()
		a.sel.Add(path)
	}
	a.current = path
}

// emptyLibrary is what the library shows when nothing matches.
func (a *app) emptyLibrary(c *ui.Context) {
	p := a.paletteNow()
	searching := strings.TrimSpace(a.query) != ""
	filtered := a.filter != "all"
	heading := "这里还没有照片"
	body := "导入一个文件夹开始 · 支持 JPEG、PNG、GIF、BMP、WebP"
	switch {
	case searching:
		heading = "这里还没有照片"
		body = "没有找到匹配的照片。试试更短的关键词，或清除搜索。"
	case filtered:
		heading = "这里还没有照片"
		body = "这个筛选下还没有照片。清除筛选可以查看全部。"
	case a.nav == "favorites":
		body = "在照片上按 F，或点缩略图上的心形，就能收藏它。"
	case strings.HasPrefix(a.nav, "album:"):
		body = "选中照片后，把相册加入所选，或把照片拖进相册。"
	}
	ui.Column(c).Width(520).MaxWidthPercent(100).Center().Padding(32).Gap(8).
		Border(1, p.borderStrong).Radius(10).Children(func() {
		ui.Box(c).Size(48, 48).Radius(10).Border(1, p.borderSubtle).Background(p.chrome).
			Center().Children(func() { glyph(c, icons.image, 22, p.textSecondary) })
		ui.Text(c, heading).FontSize(15).Bold()
		ui.Text(c, body).FontSize(13).TextColor(p.textSecondary).TextAlign(ui.Center).Wrap()
		ui.Box(c).Height(8)
		if ui.PrimaryButton(c, "导入文件夹").Clicked() {
			a.addFolder()
		}
	})
}

// dateRail is the slider on the right edge that reads the flow by date: the
// handle follows the scroll, and a tick jumps to the day it stands for. The
// dates sit at the scroll offset that brings their first photo to the top, so
// the rail matches the flow at every window size.
func (a *app) dateRail(c *ui.Context, groups []railGroup, winH float32) {
	p := a.paletteNow()
	railH := winH * 0.56
	if railH > 520 {
		railH = 520
	}
	if railH < 240 {
		railH = 240
	}
	const thumbH float32 = 28

	// Where the flow starts inside what scrolls, from the last frame's boxes.
	scrollBox := c.Resolve(a.scrollRef).Bounds()
	flowBox := c.Resolve(a.flowRef).Bounds()
	if scrollBox.H <= 0 || flowBox.H <= 0 {
		// The first frame has no bounds yet: keep the rail's room, draw it
		// once the boxes are known.
		ui.Spacer(c).Width(24).Shrink(0)
		return
	}
	flowTop := flowBox.Y - scrollBox.Y + a.libScroll.Y
	maxScroll := a.libScroll.MaxY
	if maxScroll < 1 {
		maxScroll = 1
	}
	travel := railH - 8
	thumbTravel := railH - thumbH
	scrollY := clamp(float64(a.libScroll.Y), 0, float64(maxScroll))

	// The y of a day inside the content, as the scroll offset that shows it.
	topOf := func(groupY float32) float32 {
		return clamp32((flowTop+groupY)/maxScroll, 0, 1) * travel
	}

	active := 0
	for i, g := range groups {
		if flowTop+g.y <= float32(scrollY)+8 {
			active = i
		}
	}
	thumbTop := float32(scrollY/float64(maxScroll)) * thumbTravel

	// Where each date's tick sits on the rail, before drawing anything. Two
	// dates can start in the same row of the masonry, so the rail spaces
	// them at least a few pixels apart; a long library squeezes them back
	// into the rail.
	tops := make([]float32, len(groups))
	gap := clamp32((railH-8)/float32(max(len(groups)-1, 1)), 2, 8)
	for i := range groups {
		tops[i] = 4 + topOf(groups[i].y)
		if i > 0 && tops[i] < tops[i-1]+gap {
			tops[i] = tops[i-1] + gap
		}
	}
	if last := tops[len(tops)-1]; last > railH-4 {
		k := (railH - 8) / (last - 4)
		for i := range tops {
			tops[i] = 4 + (tops[i]-4)*k
		}
	}

	ui.Column(c).Width(24).Shrink(0).Justify(ui.Center).Children(func() {
		// The rail takes the pointer itself: a click goes to the date whose
		// tick is nearest, so the whole track is a target and no date can be
		// missed between two ticks.
		nearest := func(y float32) int {
			best, bestD := 0, float32(1e9)
			for i := range tops {
				d := y - tops[i]
				if d < 0 {
					d = -d
				}
				if d < bestD {
					best, bestD = i, d
				}
			}
			return best
		}
		rail := ui.Box(c).Width(24).Height(railH).
			Label("时间轴").Description("按日期跳到照片")
		hovered := -1
		if rail.Hovered() || rail.FocusWithin() {
			if _, y, ok := rail.PointerPosition(); ok {
				hovered = nearest(y)
			}
		}
		rail = rail.HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind != ui.InputPointerDown || ev.Button != 0 {
				return false
			}
			i := nearest(ev.Y)
			a.libScroll.ScrollTo(0, flowTop+groups[i].y, ui.Smooth)
			c.Invalidate()
			return true
		})
		rail.Children(func() {
			ui.Box(c).Absolute().Left(11).Top(4).Width(2).Height(railH - 8).
				Background(p.railTrack)
			for i := range groups {
				isActive := i == active
				w, h := float32(8), float32(1)
				col := p.railTick
				if isActive {
					w, h, col = 14, 2, p.accent
				}
				ui.Box(c).Absolute().Left(12-w/2).Top(tops[i]).Size(w, h).Background(col)
				if i == hovered {
					ui.Box(c).Absolute().Left(2).Top(tops[i]-1).Size(20, 2).Background(p.textPrimary)
				}
			}
			// The handle is drawn last, over the date it sits on.
			ui.Box(c).Absolute().Left(4).Top(thumbTop).Size(16, thumbH).Radius(8).
				Border(1, p.railBorder).Background(p.railThumb)
		})
		// Resting on the rail names the date under the pointer.
		ui.TooltipBase(c, rail, func(tip ui.Element) {
			i := active
			if _, y, ok := rail.PointerPosition(); ok {
				i = nearest(y)
			}
			tip.AttachTo(rail, ui.AnchorStart, ui.AnchorEnd).MarginEnd(8)
			tip.Height(28).Padding(0, 10).AlignItems(ui.Center).Radius(6).
				Border(1, p.railBorder).Background(p.chrome)
			tip.Children(func() {
				ui.Text(c, groups[i].label).FontSize(12).Bold().SingleLine()
			})
		})
	})
}

// clamp32 keeps v within lo..hi.
func clamp32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// timelineView is the one-column timeline, newest first, with the month over
// the first photo of each month.
func (a *app) timelineView(c *ui.Context) {
	p := a.paletteNow()
	if len(a.photos) == 0 {
		a.emptyLibrary(c)
		return
	}
	ui.Column(c).Grow(1).Background(p.stage).Children(func() {
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(24, flowPaddingX).Gap(8).Children(func() {
				a.viewHeader(c)
				for i, ph := range a.photos {
					ph := ph
					startOfMonth := i == 0 || monthLabel(a.photos[i-1]) != monthLabel(ph)
					row := ui.ButtonBase(c).FillWidth().Height(64).Padding(8, 12).Gap(12).Radius(6).
						AlignItems(ui.Center).Label(ph.Name)
					if a.sel.Has(ph.Path) {
						row = row.Background(p.selected)
					} else if row.Hovered() {
						row = row.Background(p.hover)
					}
					row.Children(func() {
						if startOfMonth {
							ui.Text(c, monthLabel(ph)).FontSize(13).Bold().Width(96)
						} else {
							ui.Spacer(c).Width(96)
						}
						if bmp, ready := a.thumbs.Bitmap(ph.Path, ph.ModUnix, 160, ph.Edit); ready {
							ui.Image(c, bmp).Size(56, 56).Fit(ui.Cover).Radius(4)
						} else {
							ui.Box(c).Size(56, 56).Radius(4).Background(p.imageMat)
						}
						ui.Column(c).Gap(2).Grow(1).Children(func() {
							ui.Text(c, ph.Name).FontSize(13).SingleLine()
							sub := ph.Taken().Format("2006-01-02 15:04")
							if len(ph.Tags) > 0 {
								sub += " · " + strings.Join(ph.Tags, ", ")
							}
							ui.Text(c, sub).FontSize(11).TextColor(p.textSecondary).SingleLine()
						})
						if ph.Favorite {
							glyph(c, icons.heartFilled, 14, p.favorite)
						}
						if ph.Stars > 0 {
							ui.Row(c).Gap(1).Children(func() {
								for n := 0; n < ph.Stars; n++ {
									glyph(c, icons.starFilled, 11, p.rating)
								}
							})
						}
					})
					if row.Clicked() {
						a.choosePhoto(ph.Path, false)
					}
					if row.DoubleClicked() {
						a.sel.Clear()
						a.sel.Add(ph.Path)
						a.current = ph.Path
						a.openViewer(ph.Path)
					}
				}
			})
		})
	})
}
