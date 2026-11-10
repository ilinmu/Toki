package main

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// dialogs draws the panels the user opens over the window.
func (a *app) dialogs(c *ui.Context) {
	a.exportDialog(c)
	a.settingsDialog(c)
	a.newAlbumDialog(c)
	a.albumDialog(c)
	a.tagDialog(c)
}

// newAlbumDialog names a new album and gives it a line of description.
func (a *app) newAlbumDialog(c *ui.Context) {
	if !a.newAlbumOpen {
		return
	}
	p := a.paletteNow()
	ui.Modal(c, &a.newAlbumOpen, func() {
		ui.Column(c).Width(400).Gap(10).Children(func() {
			ui.Text(c, "新相册").FontSize(16).Bold()
			ui.Text(c, "为一组照片命名，之后可以把照片放进它").FontSize(12).TextColor(p.textSecondary)
			ui.Field(c, "相册名称", func() {
				ui.TextInput(c, &a.albumName).Placeholder("例如：海边的周末").AutoFocus().MaxLength(24)
			})
			ui.Field(c, "简短说明（可选）", func() {
				ui.TextArea(c, &a.albumDesc).Placeholder("这组照片的共同记忆").Height(64).MaxLength(80)
			})
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "取消").Clicked() {
					a.newAlbumOpen = false
				}
				if ui.PrimaryButton(c, "创建相册").Disabled(strings.TrimSpace(a.albumName) == "").Clicked() {
					a.createAlbum()
				}
			})
		})
	})
}

// createAlbum makes the album the dialog names, and shows it.
func (a *app) createAlbum() {
	name := strings.TrimSpace(a.albumName)
	if name == "" {
		return
	}
	al := a.lib.CreateAlbumWith(name, strings.TrimSpace(a.albumDesc))
	a.albumName, a.albumDesc = "", ""
	a.save()
	a.viewKey = ""
	a.navigate("album:" + al.ID)
	a.status = "已创建相册「" + al.Name + "」"
}

// exportDialog asks where to export the chosen photos and how.
func (a *app) exportDialog(c *ui.Context) {
	if !a.exportOpen {
		return
	}
	t := c.Theme()
	ui.Modal(c, &a.exportOpen, func() {
		ui.Column(c).Gap(10).Width(460).Children(func() {
			ui.Text(c, "导出照片").FontSize(16).Bold()
			ui.Textf(c, "%d 张照片，编辑会一起应用，原文件不动", len(a.target())).FontSize(11).TextColor(t.TextMuted)

			ui.Field(c, "导出位置", func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, a.form.Dir).FontSize(11).SingleLine().Grow(1).TextColor(t.TextMuted)
					if ui.Button(c, "浏览…").Clicked() {
						if paths, err := dialogOpenDir(a.win, "选择导出位置"); err == nil && len(paths) > 0 {
							a.form.Dir = paths[0]
						}
					}
				})
			})
			ui.Field(c, "文件夹名", func() {
				ui.TextInput(c, &a.form.Folder).Placeholder("libra-export")
			})
			ui.Field(c, "图像尺寸", func() {
				name := exportSizeNames[a.form.Size]
				if ui.Select(c, &name, exportSizeNames).Changed() {
					for i, n := range exportSizeNames {
						if n == name {
							a.form.Size = i
						}
					}
				}
			})
			ui.Field(c, "JPEG 质量", func() {
				var names []string
				for _, q := range exportQualities {
					names = append(names, q.Name)
				}
				name := names[a.set.ExportQuality%len(names)]
				if ui.Select(c, &name, names).Changed() {
					for i, n := range names {
						if n == name {
							a.set.ExportQuality = i
							a.saveSettings()
						}
					}
				}
			})
			ui.Field(c, "水印文字", func() {
				ui.TextInput(c, &a.form.Watermark).Placeholder("姓名、网址或版权声明，留空则不加")
			})
			ui.Checkbox(c, &a.form.Numbers, "文件名加序号以保持顺序")
			ui.Divider(c)
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "取消").Clicked() {
					a.exportOpen = false
				}
				if ui.PrimaryButton(c, "导出").Clicked() {
					photos := a.target()
					form := a.form
					a.runExport(photos, form)
				}
			})
		})
	})
}

// settingsDialog changes the preferences.
func (a *app) settingsDialog(c *ui.Context) {
	if !a.settingsOpen {
		return
	}
	t := c.Theme()
	ui.Modal(c, &a.settingsOpen, func() {
		ui.Column(c).Gap(10).Width(440).Children(func() {
			ui.Text(c, "设置").FontSize(16).Bold()

			ui.Field(c, "照片排序", func() {
				name := sortName(a.set.Sort)
				if ui.Select(c, &name, sortNames).Changed() {
					a.set.Sort = sortKey(name)
					a.saveSettings()
					a.viewKey = ""
				}
			})
			ui.Text(c, "尺寸与排序在工具栏右侧。").FontSize(10).TextColor(t.TextMuted)
			if ui.Checkbox(c, &a.set.Dedupe, "扫描时跳过内容重复的照片").Changed() {
				a.saveSettings()
			}
			if ui.Checkbox(c, &a.set.ConfirmForget, "把照片移出库前确认").Changed() {
				a.saveSettings()
			}
			if ui.Checkbox(c, &a.set.ShowNames, "在缩略图下显示文件名").Changed() {
				a.saveSettings()
			}
			if ui.Checkbox(c, &a.set.WriteMetadata, "更改说明或标签时同时写入文件元数据").Changed() {
				a.saveSettings()
			}
			ui.Divider(c)
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Textf(c, "缩略图缓存：%.1f MB", float64(a.thumbs.size())/(1<<20)).FontSize(11).TextColor(t.TextMuted).Grow(1)
				if ui.Button(c, "清空").Clicked() {
					if err := a.thumbs.clearDisk(); err == nil {
						a.status = "缩略图缓存已清空"
					}
				}
			})
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.PrimaryButton(c, "完成").Clicked() {
					a.settingsOpen = false
				}
			})
		})
	})
}

// albumDialog adds the chosen photos to an album.
func (a *app) albumDialog(c *ui.Context) {
	if !a.albumOpen {
		return
	}
	t := c.Theme()
	ui.Modal(c, &a.albumOpen, func() {
		ui.Column(c).Gap(10).Width(380).Children(func() {
			ui.Text(c, "加入相册").FontSize(16).Bold()
			ui.Textf(c, "选中 %d 张", len(a.target())).FontSize(11).TextColor(t.TextMuted)
			for _, al := range a.lib.AlbumList() {
				al := al
				count := 0
				for _, p := range a.lib.Snapshot() {
					if p.InAlbum(al.ID) {
						count++
					}
				}
				if ui.Button(c, al.Name+"  ·  "+itoa(count)+" 张").Clicked() {
					a.lib.AddToAlbum(al.ID, a.target())
					a.save()
					a.albumOpen = false
					a.status = "已加入相册「" + al.Name + "」"
				}
			}
			ui.Divider(c)
			ui.Field(c, "新建相册", func() {
				ui.Row(c).Gap(6).Children(func() {
					ui.TextInput(c, &a.albumName).Placeholder("相册名").Grow(1)
					if ui.PrimaryButton(c, "创建并加入").Clicked() && a.albumName != "" {
						al := a.lib.CreateAlbum(a.albumName)
						a.lib.AddToAlbum(al.ID, a.target())
						a.albumName = ""
						a.save()
						a.albumOpen = false
						a.status = "已创建相册「" + al.Name + "」"
					}
				})
			})
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.Button(c, "取消").Clicked() {
					a.albumOpen = false
				}
			})
		})
	})
}

// tagDialog tags the chosen photos.
func (a *app) tagDialog(c *ui.Context) {
	if !a.tagOpen {
		return
	}
	t := c.Theme()
	ui.Modal(c, &a.tagOpen, func() {
		ui.Column(c).Gap(10).Width(400).Children(func() {
			ui.Text(c, "添加标签").FontSize(16).Bold()
			ui.Textf(c, "选中 %d 张", len(a.target())).FontSize(11).TextColor(t.TextMuted)
			ui.Row(c).Gap(6).Children(func() {
				input := ui.TextInput(c, &a.tagText).Placeholder("标签名，例如 家庭").Grow(1).AutoFocus()
				apply := func() {
					tag := strings.TrimSpace(a.tagText)
					if tag == "" {
						return
					}
					for _, p := range a.target() {
						if !p.HasTag(tag) {
							p.Tags = append(p.Tags, tag)
						}
					}
					a.lib.LearnTag(tag)
					a.tagText = ""
					a.save()
					a.viewKey = ""
					a.status = "已加标签「" + tag + "」"
				}
				if ui.PrimaryButton(c, "添加").Clicked() || input.Submitted() {
					apply()
				}
			})
			if tags := a.lib.AllTags(); len(tags) > 0 {
				ui.Text(c, "已有标签").FontSize(11).TextColor(t.TextMuted)
				ui.Row(c).Wrap().Gap(6).Children(func() {
					for _, tag := range tags {
						tag := tag
						if ui.Button(c, tag).Clicked() {
							for _, p := range a.target() {
								if !p.HasTag(tag) {
									p.Tags = append(p.Tags, tag)
								}
							}
							a.save()
							a.viewKey = ""
							a.status = "已加标签「" + tag + "」"
						}
					}
				})
			}
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.PrimaryButton(c, "完成").Clicked() {
					a.tagOpen = false
				}
			})
		})
	})
}

// dialogOpenDir asks for one folder.
func dialogOpenDir(parent *mygo.Window, title string) ([]string, error) {
	return mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: parent, Title: title, Directory: true})
}
