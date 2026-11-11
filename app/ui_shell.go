package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
)

// The app calls itself 時, Toki in Latin script: the prototype's design with
// this project's own name. The title bar shows one name, the Chinese one by
// default; the Latin spelling stays here for the day the UI is translated.
const (
	brandName      = "時"
	brandNameLatin = "Toki"
)

// brandTitle is the single name the UI shows: Chinese until the UI is
// translated.
func brandTitle() string { return brandName }

// titlebar is the window's own title bar, drawn under the system's window
// controls: the mark, the name of what is shown, the search, the appearance,
// the import and the inspector.
func (a *app) titlebar(c *ui.Context) {
	p := a.paletteNow()
	bar := c.TitleBar()
	height := titlebarHeight
	if bar.Height > height {
		height = bar.Height
	}
	ui.Row(c).Height(height).
		PaddingStart(bar.Left + 12).PaddingEnd(12).
		Gap(12).AlignItems(ui.Center).
		Background(p.chrome).DragWindow().Children(func() {
		ui.Row(c).Width(sidebarWidth - 12).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			glyph(c, brandMark, 20, p.accent)
			ui.Text(c, brandTitle()).FontSize(14).Bold().SingleLine()
		})

		ui.Text(c, a.windowSubtitle()).FontSize(12).TextColor(p.textSecondary).
			Grow(1).TextAlign(ui.Center).SingleLine()

		ui.Row(c).Gap(8).Shrink(0).AlignItems(ui.Center).Children(func() {
			a.searchField(c)
			ic := icons.sun
			tip := "切换为深色外观 (⇧⌘L)"
			if p.dark {
				ic, tip = icons.moon, "切换为浅色外观 (⇧⌘L)"
			}
			if a.iconButton(c, ic, tip, false).Clicked() {
				a.toggleTheme()
			}
			if a.primaryAction(c, "导入", icons.importFile).Clicked() {
				a.addFolder()
			}
			if a.iconButton(c, icons.sidebar, "显示或隐藏照片信息 (⇧⌘I)", a.inspectorOpen).Clicked() {
				a.inspectorOpen = !a.inspectorOpen
			}
			if a.iconButton(c, icons.gear, "设置", false).Clicked() {
				a.settingsOpen = true
			}
		})
	})
	ui.Box(c).Height(1).Background(p.borderSubtle)
}

// searchField is the title bar's search: a real field, so typing filters the
// library at once. Command-K puts the keyboard in it; the command palette,
// which also finds albums and tags, stays on ⌘⇧K.
func (a *app) searchField(c *ui.Context) {
	p := a.paletteNow()
	ui.Row(c).Width(220).Height(controlCompact).Gap(4).AlignItems(ui.Center).Children(func() {
		input := ui.TextInput(c, &a.query).Placeholder("搜索照片、标签、说明").
			Grow(1).Label("搜索照片")
		if a.focusSearch {
			input.Focus()
			a.focusSearch = false
		}
		if input.Changed() {
			a.viewKey = ""
			if a.mode == 2 {
				a.mode = 0
			}
			a.libScroll.ScrollTo(0, 0, ui.Smooth)
		}
		if a.query != "" {
			if a.iconButton(c, icons.x, "清除搜索", false).Clicked() {
				a.query = ""
				a.viewKey = ""
				a.focusSearch = true
			}
		} else {
			ui.Text(c, "⌘K").FontSize(11).TextColor(p.textDisabled)
		}
	})
}

// windowSubtitle is the name of the place the window shows, in the middle of
// the title bar.
func (a *app) windowSubtitle() string {
	switch {
	case a.lightbox:
		if p := a.lib.Get(a.current); p != nil {
			return p.Name
		}
		return "照片"
	case a.mode == 2:
		if a.editOpen {
			return "编辑照片"
		}
		return "查看详情"
	case a.mode == 1:
		return "时间轴"
	}
	if q := strings.TrimSpace(a.query); q != "" {
		return "搜索 · “" + q + "”"
	}
	nav := a.nav
	switch {
	case nav == "" || nav == "all":
		return "照片图库"
	case nav == "stars":
		return "星标照片"
	case nav == "recent":
		return "最近添加"
	case nav == "favorites":
		return "收藏"
	case nav == "edited":
		return "已编辑"
	case strings.HasPrefix(nav, "album:"):
		if al := a.lib.Album(strings.TrimPrefix(nav, "album:")); al != nil {
			return al.Name
		}
		return "相册"
	case strings.HasPrefix(nav, "tag:"):
		return "标签 · " + strings.TrimPrefix(nav, "tag:")
	case strings.HasPrefix(nav, "folder:"):
		return filepath.Base(strings.TrimPrefix(nav, "folder:"))
	case strings.HasPrefix(nav, "year:"):
		return strings.TrimPrefix(nav, "year:") + " 年"
	}
	return "照片图库"
}

// sidebar is the column of places: the library, the albums, the tags, the
// folders and the years. Every count comes from the cached library index, so
// a redraw is O(number of places), not O(number of photos).
func (a *app) sidebar(c *ui.Context) {
	p := a.paletteNow()
	d := a.library()
	counts := d.counts
	ui.Column(c).Width(sidebarWidth).Shrink(0).Background(p.chrome).Children(func() {
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(16, 8).Gap(8).Children(func() {
				ui.Column(c).Gap(4).Children(func() {
					if a.navButton(c, "全部照片", icons.image, counts.all, a.mode == 0 && a.nav == "all").Clicked() {
						a.navigate("all")
					}
					if a.navButton(c, "最近添加", icons.clock, counts.recent, a.nav == "recent").Clicked() {
						a.navigate("recent")
					}
					if a.navButton(c, "收藏", icons.heart, counts.favorites, a.nav == "favorites").Clicked() {
						a.navigate("favorites")
					}
					if a.navButton(c, "星标照片", icons.star, counts.stars, a.nav == "stars").Clicked() {
						a.navigate("stars")
					}
					if a.navButton(c, "已编辑", icons.edit, counts.edited, a.nav == "edited").Clicked() {
						a.navigate("edited")
					}
				})
				// The timeline is not a place of its own in the sidebar: the
				// arrangement buttons in the toolbar and View > 时间轴 (⌘3)
				// already switch to it.

				// The albums, each with the tint the prototype gives it.
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					a.sectionLabel(c, "我的相册")
					ui.Spacer(c).Grow(1)
					if a.iconButton(c, icons.plus, "新建相册", false).Clicked() {
						a.albumName = ""
						a.albumDesc = ""
						a.newAlbumOpen = true
					}
				})
				ui.Column(c).Gap(2).Children(func() {
					if len(d.albums) == 0 {
						ui.Text(c, "还没有相册").FontSize(12).TextColor(p.textDisabled).Padding(0, 8)
					}
					for _, al := range d.albums {
						al := al
						on := a.nav == "album:"+al.id
						b := a.albumButton(c, al.name, al.tint, al.count, on)
						if b.Clicked() {
							a.navigate("album:" + al.id)
						}
						b.ContextMenu(func(m *ui.Menu) {
							if m.Item("从中移出选中的照片").Disabled(len(a.selected()) == 0).Chosen() {
								a.removeFromAlbum(al.id)
							}
							m.Separator()
							if m.Item("删除相册「" + al.name + "」").Chosen() {
								a.deleteAlbum(al.id)
							}
						})
					}
				})

				if len(d.tags) > 0 {
					a.sectionLabel(c, "标签")
					ui.Column(c).Gap(2).Children(func() {
						for _, row := range d.tags {
							row := row
							b := a.navButton(c, row.name, icons.tag, row.count, a.nav == "tag:"+row.name)
							if b.Clicked() {
								a.navigate("tag:" + row.name)
							}
							b.ContextMenu(func(m *ui.Menu) {
								if m.Item("删除标签「" + row.name + "」").Chosen() {
									a.confirmDeleteTag(row.name)
								}
							})
						}
					})
				}

				if len(d.folders) > 0 {
					ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
						a.sectionLabel(c, "文件夹")
						ui.Spacer(c).Grow(1)
						if a.iconButton(c, icons.plus, "导入文件夹 (⌘O)", false).Clicked() {
							a.addFolder()
						}
					})
					ui.Column(c).Gap(2).Children(func() {
						for _, row := range d.folders {
							row := row
							b := a.navButton(c, filepath.Base(row.path), icons.folder, row.count, a.nav == "folder:"+row.path)
							if b.Clicked() {
								a.navigate("folder:" + row.path)
							}
							b.ContextMenu(func(m *ui.Menu) {
								if m.Item("移除文件夹「" + filepath.Base(row.path) + "」…").Chosen() {
									a.confirmRemoveFolder(row.path)
								}
							})
						}
					})
				}

				if len(d.years) > 0 {
					a.sectionLabel(c, "年份")
					ui.Column(c).Gap(2).Children(func() {
						for _, row := range d.years {
							row := row
							if a.navButton(c, row.year+" 年", icons.camera, row.count, a.nav == "year:"+row.year).Clicked() {
								a.navigate("year:" + row.year)
							}
						}
					})
				}
			})
		})

		a.hairline(c, false)
		ui.Column(c).Padding(12, 16).Gap(4).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, "本地资料库").FontSize(12).TextColor(p.textSecondary).Grow(1)
				ui.Text(c, humanSize(counts.bytes)).FontSize(12).TextColor(p.textPrimary)
			})
			ui.Textf(c, "%d 张照片 · 不会自动上传", counts.all).FontSize(11).TextColor(p.textDisabled)
		})
	})
}

// libraryCounts are the numbers the sidebar shows.
type libCounts struct {
	all, recent, favorites, stars, edited int
	bytes                                 int64
}

// navigate shows one place of the library, from the top.
func (a *app) navigate(nav string) {
	a.nav = nav
	a.mode = min(a.mode, 1)
	a.mode = 0
	a.filter = "all"
	a.viewKey = ""
	a.libScroll.Y = 0
	a.libScroll.ScrollTo(0, 0, ui.Smooth)
	a.timelineScroll.Y = 0
	a.railDrag = false
}

// statusbar is the line under the photos: how many, what is chosen and what
// the batch actions are.
func (a *app) statusbar(c *ui.Context) {
	p := a.paletteNow()
	n := len(a.photos)
	chosen := len(a.selected())
	ui.Row(c).Height(statusbarHeight).Padding(0, 16).Gap(10).AlignItems(ui.Center).
		Background(p.chrome).Children(func() {
		text := fmt.Sprintf("%d 张照片", n)
		if chosen > 0 {
			text = fmt.Sprintf("%d 张照片 · 已选 %d 张", n, chosen)
		}
		ui.Text(c, text).FontSize(12).TextColor(p.textSecondary)
		if a.scanning {
			ui.Textf(c, "正在扫描… %s", a.scanMsg).FontSize(12).TextColor(p.accent)
		}
		if a.preparing {
			label := "正在后台准备缩略图…"
			if a.prepareTotal > 0 {
				label = fmt.Sprintf("正在后台准备缩略图… %d/%d", a.prepareDone, a.prepareTotal)
			}
			ui.Text(c, label).FontSize(12).TextColor(p.accent)
		}
		if a.status != "" {
			ui.Text(c, a.status).FontSize(12).TextColor(p.textDisabled).SingleLine()
		}
		ui.Spacer(c).Grow(1)
		if chosen > 0 {
			if a.quietButton(c, "打星", icons.star).Clicked() {
				a.starSelection(1)
			}
			if a.quietButton(c, "标签", icons.tag).Clicked() {
				a.tagOpen = true
			}
			if a.quietButton(c, "加入相册", icons.albums).Clicked() {
				a.albumOpen = true
			}
			if a.quietButton(c, "导出", icons.download).Clicked() {
				a.openExport()
			}
			if a.quietButton(c, "移出库", icons.trash).Clicked() {
				a.forgetSelected()
			}
		}
	})
}

// toggleFavoriteCurrent flips the heart of the photo shown, or of the chosen
// photos.
func (a *app) toggleFavoriteCurrent(c *ui.Context) {
	target := a.target()
	if len(target) == 0 {
		return
	}
	on := !target[0].Favorite
	for _, p := range target {
		p.Favorite = on
	}
	a.save()
	a.viewKey = ""
	if on {
		a.celebrateHere(c, ui.Element{})
		a.status = fmt.Sprintf("已收藏 %d 张", len(target))
	}
}

// removeFromAlbum takes the chosen photos out of an album.
func (a *app) removeFromAlbum(id string) {
	for _, p := range a.selected() {
		p.Albums = without(p.Albums, id)
	}
	a.save()
	a.viewKey = ""
}

// deleteAlbum removes an album and takes it off its photos.
func (a *app) deleteAlbum(id string) {
	a.lib.DeleteAlbum(id)
	if a.nav == "album:"+id {
		a.navigate("all")
	}
	a.save()
	a.viewKey = ""
	a.status = "相册已删除"
}

// openCommand opens the command palette with an empty query.
func (a *app) openCommand() {
	a.commandOpen = true
	a.commandQuery = ""
}

// commandPalette is the search over photos, albums, tags and folders, over a
// dimmed window.
func (a *app) commandPalette(c *ui.Context) {
	if !a.commandOpen {
		return
	}
	p := a.paletteNow()
	ui.Modal(c, &a.commandOpen, func() {
		ui.Column(c).Width(560).Radius(10).Border(1, p.borderStrong).Background(p.elevated).
			Clip().Children(func() {
			ui.Row(c).Height(48).Padding(0, 12).Gap(8).AlignItems(ui.Center).Children(func() {
				glyph(c, icons.search, 18, p.textSecondary)
				input := ui.TextInput(c, &a.commandQuery).AutoFocus().
					Placeholder("搜索照片、相册、标签或文件夹").Grow(1).Label("搜索")
				_ = input
				ui.Text(c, "esc").FontSize(11).TextColor(p.textDisabled)
			})
			a.hairline(c, false)
			ui.Column(c).Gap(2).Padding(8).Children(func() {
				results := a.commandResults()
				if len(results) == 0 {
					ui.Column(c).Padding(16).Gap(4).Children(func() {
						ui.Text(c, "没有找到匹配的内容").FontSize(13)
						ui.Text(c, "试试更短的关键词，或搜一个标签名").FontSize(11).TextColor(p.textDisabled)
					})
					return
				}
				for i, r := range results {
					r := r
					on := i == 0
					row := ui.ButtonBase(c).Height(44).Padding(0, 8).Gap(10).Radius(6).Label(r.title)
					if on {
						row = row.Background(p.selected)
					}
					if row.Hovered() {
						row = row.Background(p.selected)
					}
					row.Children(func() {
						ui.Box(c).Size(28, 28).Radius(4).Background(p.chrome).Center().Children(func() {
							glyph(c, r.icon, 15, p.textSecondary)
						})
						ui.Column(c).Gap(1).Grow(1).Children(func() {
							ui.Text(c, r.title).FontSize(13).TextColor(p.textPrimary).SingleLine()
							ui.Text(c, r.sub).FontSize(11).TextColor(p.textSecondary).SingleLine()
						})
					})
					if row.Clicked() {
						a.runCommand(r)
					}
				}
			})
			a.hairline(c, false)
			ui.Row(c).Height(32).Padding(0, 12).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "↵ 打开").FontSize(11).TextColor(p.textDisabled)
				ui.Text(c, "⌘K 搜索").FontSize(11).TextColor(p.textDisabled)
			})
		})
	})
}

// commandResult is one row of the palette.
type commandResult struct {
	title, sub string
	icon       *ui.SVG
	run        func()
}

// commandResults matches the query against the photos, the albums, the tags
// and the folders.
func (a *app) commandResults() []commandResult {
	q := strings.ToLower(strings.TrimSpace(a.commandQuery))
	var out []commandResult
	add := func(r commandResult) {
		if len(out) < 9 {
			out = append(out, r)
		}
	}
	match := func(s string) bool {
		return q == "" || strings.Contains(strings.ToLower(s), q)
	}
	d := a.library()
	for _, p := range d.all {
		if len(out) >= 6 {
			break
		}
		if !match(p.Name) && !match(p.Caption) && !containsTag(p.Tags, q) {
			continue
		}
		p := p
		add(commandResult{
			title: p.Name,
			sub:   p.Taken().Format("2006-01-02 15:04") + " · " + filepath.Base(filepath.Dir(p.Path)),
			icon:  icons.image,
			run: func() {
				a.commandOpen = false
				a.sel.Clear()
				a.sel.Add(p.Path)
				a.current = p.Path
				a.openViewer(p.Path)
			},
		})
	}
	for _, al := range d.albums {
		if !match(al.name) {
			continue
		}
		al := al
		add(commandResult{
			title: al.name,
			sub:   "相册",
			icon:  icons.albums,
			run: func() {
				a.commandOpen = false
				a.navigate("album:" + al.id)
			},
		})
	}
	for _, row := range d.tags {
		if !match(row.name) {
			continue
		}
		tag := row.name
		add(commandResult{
			title: tag,
			sub:   "标签",
			icon:  icons.tag,
			run: func() {
				a.commandOpen = false
				a.navigate("tag:" + tag)
			},
		})
	}
	for _, row := range d.folders {
		if !match(filepath.Base(row.path)) {
			continue
		}
		dir := row.path
		add(commandResult{
			title: filepath.Base(dir),
			sub:   dir,
			icon:  icons.folder,
			run: func() {
				a.commandOpen = false
				a.navigate("folder:" + dir)
			},
		})
	}
	return out
}

// runCommand carries out the row of the palette that was chosen.
func (a *app) runCommand(r commandResult) {
	if r.run != nil {
		r.run()
	}
}

// containsTag reports whether tags holds a tag matching the query.
func containsTag(tags []string, q string) bool {
	if q == "" {
		return len(tags) > 0
	}
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}
