package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
)

// inspector is the column on the right: what is chosen, its properties, its
// tags, its albums and what can be done with it.
func (a *app) inspector(c *ui.Context) {
	p := a.paletteNow()
	ui.Column(c).Width(inspectorWidth).Shrink(0).Background(p.chrome).Children(func() {
		ui.Row(c).Height(titlebarHeight).Padding(0, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			a.inspectorHeading(c, "信息")
			ui.Spacer(c).Grow(1)
			if a.iconButton(c, icons.x, "隐藏照片信息 (⇧⌘I)", false).Clicked() {
				a.inspectorOpen = false
			}
		})
		a.hairline(c, false)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(16).Gap(4).Children(func() {
				selected := a.selected()
				switch {
				case len(selected) > 1:
					a.inspectorMany(c, selected)
				case len(selected) == 1:
					a.inspectorPhoto(c, selected[0])
				default:
					if p := a.lib.Get(a.current); p != nil && a.mode == 2 {
						a.inspectorPhoto(c, p)
						return
					}
					a.inspectorOverview(c)
				}
			})
		})
	})
}

// inspectorSection is a titled part of the inspector, under a hairline.
func (a *app) inspectorSection(c *ui.Context, title string, fn func()) {
	p := a.paletteNow()
	ui.Box(c).Height(8)
	a.hairline(c, false)
	ui.Column(c).PaddingY(10).Gap(8).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			a.inspectorHeading(c, title)
			ui.Spacer(c).Grow(1)
		})
		fn()
	})
	_ = p
}

// inspectorOverview is what the inspector shows when no photo is chosen: what
// the library holds.
func (a *app) inspectorOverview(c *ui.Context) {
	p := a.paletteNow()
	d := a.library()
	counts := d.counts
	ui.Column(c).Gap(8).PaddingY(12).Children(func() {
		ui.Box(c).Size(44, 44).Radius(10).Border(1, p.borderSubtle).Background(p.elevated).
			Center().Children(func() { glyph(c, icons.info, 20, p.textSecondary) })
		ui.Text(c, "选择一张照片").FontSize(15).Bold()
		ui.Text(c, "在照片上单击可查看拍摄信息、评分、标签和所属相册。").
			FontSize(12).TextColor(p.textSecondary).Wrap()
	})
	a.hairline(c, false)
	ui.Column(c).Padding(12, 0).Gap(2).Children(func() {
		a.inspectorHeading(c, "资料库")
		ui.Box(c).Height(4)
		a.overviewRow(c, "照片", itoa(counts.all))
		a.overviewRow(c, "收藏", itoa(counts.favorites))
		a.overviewRow(c, "已编辑", itoa(counts.edited))
		a.overviewRow(c, "相册", itoa(len(d.albums)))
		a.overviewRow(c, "标签", itoa(len(d.tags)))
		a.overviewRow(c, "占用空间", humanSize(counts.bytes))
	})
}

// overviewRow is one line of the library overview.
func (a *app) overviewRow(c *ui.Context, label, value string) {
	p := a.paletteNow()
	ui.Row(c).Gap(8).Padding(4, 0).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(p.textSecondary).Grow(1)
		ui.Text(c, value).FontSize(12).Bold().TextColor(p.textPrimary)
	})
}

// inspectorPhoto is the inspector for one photo.
func (a *app) inspectorPhoto(c *ui.Context, ph *Photo) {
	p := a.paletteNow()
	// The picture, cropped to 4:3 as the prototype's preview is: the heart's
	// confetti plays on it.
	preview := ui.Box(c).FillWidth().AspectRatio(4.0 / 3.0).Radius(4).Clip().Background(p.imageMat).
		Children(func() {
			// The inspector is small: show any cached thumbnail at once, and
			// let the 480px version replace it when it is ready.
			if bmp, ready := a.previewBitmap(ph, 480); ready {
				ui.Image(c, bmp).Fit(ui.Cover).Fill()
			}
		})
	ui.Column(c).PaddingY(10).Gap(2).Children(func() {
		ui.Text(c, ph.Name).FontSize(20).Bold().Wrap()
		meta := ph.Taken().Format("2006-01-02 15:04")
		if !ph.Edit.IsZero() {
			meta += " · 已编辑"
		}
		ui.Text(c, meta).FontSize(12).TextColor(p.textSecondary)
	})
	// The rating, and the heart beside it.
	ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
		a.ratingRow(c, ph, 16)
		ui.Spacer(c).Grow(1)
		ic := icons.heart
		col := p.textSecondary
		tip := "收藏 (F)"
		if ph.Favorite {
			ic, col, tip = icons.heartFilled, p.favorite, "取消收藏 (F)"
		}
		heart := a.iconButtonTint(c, ic, tip, col)
		if heart.Clicked() {
			ph.Favorite = !ph.Favorite
			a.save()
			a.viewKey = ""
			if ph.Favorite {
				a.celebrate(c, heart, preview.Bounds())
			}
		}
	})

	a.inspectorSection(c, "照片信息", func() {
		a.infoRow(c, "文件名", ph.Name)
		a.infoRow(c, "尺寸", fmt.Sprintf("%d × %d", ph.Width, ph.Height))
		a.infoRow(c, "文件大小", humanSize(ph.Size))
		a.infoRow(c, "拍摄时间", ph.Taken().Format("2006-01-02 15:04:05"))
		a.infoRow(c, "文件夹", filepath.Base(filepath.Dir(ph.Path)))
		a.infoRow(c, "相册", a.albumNames(ph))
		if ph.Make != "" || ph.Model != "" {
			a.infoRow(c, "相机", strings.TrimSpace(ph.Make+" "+ph.Model))
		}
		if ph.ISO > 0 {
			a.infoRow(c, "ISO", itoa(ph.ISO))
		}
		if ph.FNumber > 0 {
			a.infoRow(c, "光圈", fmt.Sprintf("f/%.1f", ph.FNumber))
		}
		if ph.ExposureTime != "" {
			a.infoRow(c, "曝光", ph.ExposureTime)
		}
		if ph.FocalLength > 0 {
			a.infoRow(c, "焦距", itoa(ph.FocalLength)+" mm")
		}
		if ph.HasGPS {
			a.infoRow(c, "位置", fmt.Sprintf("%.5f, %.5f", ph.Lat, ph.Lon))
		}
	})

	a.inspectorSection(c, "标签", func() {
		if len(ph.Tags) == 0 {
			ui.Text(c, "还没有标签").FontSize(11).TextColor(p.textDisabled)
		}
		ui.Row(c).Wrap().Gap(4).Children(func() {
			for i, tag := range ph.Tags {
				i, tag := i, tag
				if a.cardChip(c, tag, true).Clicked() {
					a.removeTag(ph, i)
				}
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			input := ui.TextInput(c, &a.tagEntry).Placeholder("添加标签").Grow(1).Label("添加标签")
			apply := func() {
				tag := strings.TrimSpace(a.tagEntry)
				if tag == "" {
					return
				}
				if !ph.HasTag(tag) {
					ph.Tags = append(ph.Tags, tag)
				}
				a.lib.LearnTag(tag)
				a.tagEntry = ""
				a.save()
				a.viewKey = ""
				if a.set.WriteMetadata {
					go writeMetadata(ph.Path, ph.Caption, ph.Tags, ph.Stars)
				}
			}
			if input.Submitted() {
				apply()
			}
			if a.iconButton(c, icons.plus, "添加标签", false).Clicked() {
				apply()
			}
		})
	})

	a.inspectorSection(c, "所属相册", func() {
		albums := a.library().albums
		if len(albums) == 0 {
			ui.Text(c, "还没有相册").FontSize(11).TextColor(p.textDisabled)
		} else {
			ui.Row(c).Gap(6).Children(func() {
				names := make([]string, 0, len(albums))
				for _, al := range albums {
					names = append(names, al.name)
				}
				if a.albumPick == "" {
					a.albumPick = names[0]
				}
				if ui.Select(c, &a.albumPick, names).Grow(1).Label("相册").Changed() {
					// A choice is only made when the button is used.
				}
				if a.actionButton(c, "加入", icons.plus).Clicked() {
					for _, al := range albums {
						if al.name == a.albumPick {
							a.lib.AddToAlbum(al.id, []*Photo{ph})
							a.save()
							a.viewKey = ""
							a.status = "已加入相册「" + al.name + "」"
						}
					}
				}
			})
		}
	})

	a.inspectorSection(c, "操作", func() {
		ui.Row(c).Wrap().Gap(6).Children(func() {
			if a.actionButton(c, "大图查看", icons.expand).Clicked() {
				a.openViewer(ph.Path)
			}
			if a.actionButton(c, "编辑", icons.sliders).Clicked() {
				a.openPhoto(ph.Path)
				a.openEditPanel(ph)
			}
			if a.actionButton(c, "导出", icons.download).Clicked() {
				a.sel.Clear()
				a.sel.Add(ph.Path)
				a.openExport()
			}
			if a.actionButton(c, "在 Finder 中显示", icons.folder).Clicked() {
				revealInFinder(ph.Path)
			}
			if a.actionButton(c, "移出库", icons.trash).Clicked() {
				a.sel.Clear()
				a.sel.Add(ph.Path)
				a.forgetSelected()
			}
		})
	})
}

// albumNames lists the albums a photo is in, for the metadata grid.
func (a *app) albumNames(ph *Photo) string {
	var names []string
	for _, al := range a.library().albums {
		if ph.InAlbum(al.id) {
			names = append(names, al.name)
		}
	}
	if len(names) == 0 {
		return "未加入相册"
	}
	return strings.Join(names, "、")
}

// removeTag takes one tag off a photo. A tag view lists photos by their
// tags, so the one that lost this tag leaves the list at once.
func (a *app) removeTag(ph *Photo, i int) {
	if i < 0 || i >= len(ph.Tags) {
		return
	}
	ph.Tags = append(ph.Tags[:i:i], ph.Tags[i+1:]...)
	a.save()
	a.viewKey = ""
}

// inspectorMany is the inspector for a choice of several photos.
func (a *app) inspectorMany(c *ui.Context, chosen []*Photo) {
	p := a.paletteNow()
	first := chosen[0]
	ui.Box(c).FillWidth().AspectRatio(4.0 / 3.0).Radius(4).Clip().Background(p.imageMat).
		Children(func() {
			if bmp, ready := a.thumbs.Bitmap(first.Path, first.ModUnix, 480, first.Edit); ready {
				ui.Image(c, bmp).Fit(ui.Cover).Fill()
			}
		})
	favorites := 0
	for _, ph := range chosen {
		if ph.Favorite {
			favorites++
		}
	}
	ui.Column(c).PaddingY(10).Gap(2).Children(func() {
		ui.Textf(c, "%d 张照片", len(chosen)).FontSize(20).Bold()
		ui.Textf(c, "已选 %d 张收藏 · 可一起打星、加标签或加入相册", favorites).
			FontSize(12).TextColor(p.textSecondary).Wrap()
	})

	a.inspectorSection(c, "批量操作", func() {
		ui.Row(c).Gap(6).Wrap().Children(func() {
			if a.chip(c, "★ 打星", false).Clicked() {
				a.starSelection(1)
			}
			if a.chip(c, "收藏", false).Clicked() {
				a.toggleFavoriteCurrent(c)
			}
			if a.chip(c, "标签…", false).Clicked() {
				a.tagOpen = true
			}
			if a.chip(c, "加入相册…", false).Clicked() {
				a.albumOpen = true
			}
			if strings.HasPrefix(a.nav, "album:") {
				if a.chip(c, "从当前相册移出", false).Clicked() {
					a.removeFromAlbum(strings.TrimPrefix(a.nav, "album:"))
				}
			}
			if a.chip(c, "取消选择", false).Clicked() {
				a.sel.Clear()
			}
		})
	})

	a.inspectorSection(c, "选中的照片", func() {
		ui.Row(c).Wrap().Gap(4).Children(func() {
			for i, ph := range chosen {
				if i >= 8 {
					ui.Textf(c, "+%d", len(chosen)-8).FontSize(11).TextColor(p.textSecondary)
					break
				}
				ui.Text(c, ph.Name).FontSize(11).TextColor(p.textPrimary).
					Padding(4, 8).Radius(4).Border(1, p.borderSubtle).Background(p.elevated).
					MaxWidth(150).SingleLine()
			}
		})
	})

	a.inspectorSection(c, "操作", func() {
		ui.Row(c).Wrap().Gap(6).Children(func() {
			if a.actionButton(c, "导出", icons.download).Clicked() {
				a.openExport()
			}
			if a.actionButton(c, "写入元数据", icons.tag).Clicked() {
				a.writeMetadataForSelection()
			}
			if a.actionButton(c, "移出库", icons.trash).Clicked() {
				a.forgetSelected()
			}
		})
	})
}
