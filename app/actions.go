package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// menu is the application's menu bar: the standard menus, and the items that
// do what the window does, with the keys that do it too.
func (a *app) menu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Role: mygo.RoleFileMenu, Submenu: []*mygo.MenuItem{
			{Label: "导入文件夹…", Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { a.addFolder() }},
			{Label: "刷新库", Accelerator: "CmdOrCtrl+R", Click: func(*mygo.MenuItem, *mygo.Window) { a.refresh() }},
			mygo.Separator(),
			{Label: "导出…", Accelerator: "CmdOrCtrl+E", Click: func(*mygo.MenuItem, *mygo.Window) { a.openExport() }},
			{Label: "写入文件元数据", Click: func(*mygo.MenuItem, *mygo.Window) { a.writeMetadataForSelection() }},
			mygo.Separator(),
			{Label: "新建相册…", Accelerator: "CmdOrCtrl+Shift+N", Click: func(*mygo.MenuItem, *mygo.Window) {
				a.albumName, a.albumDesc = "", ""
				a.newAlbumOpen = true
			}},
			{Label: "加入相册…", Click: func(*mygo.MenuItem, *mygo.Window) { a.albumOpen = true }},
			{Label: "给选中照片加标签…", Click: func(*mygo.MenuItem, *mygo.Window) {
				if len(a.target()) > 0 {
					a.tagOpen = true
				}
			}},
			mygo.Separator(),
			{Label: "清空缩略图缓存", Click: func(*mygo.MenuItem, *mygo.Window) {
				if err := a.thumbs.clearDisk(); err != nil {
					a.status = "清理失败: " + err.Error()
				} else {
					a.status = "缩略图缓存已清空"
				}
				a.refreshUI()
			}},
			{Label: "移出库…", Click: func(*mygo.MenuItem, *mygo.Window) { a.forgetSelected() }},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu, Submenu: []*mygo.MenuItem{
			{Label: "网格排列", Accelerator: "CmdOrCtrl+1", Click: func(*mygo.MenuItem, *mygo.Window) { a.mode, a.layout = 0, 0; a.saveLayout() }},
			{Label: "瀑布流排列", Accelerator: "CmdOrCtrl+2", Click: func(*mygo.MenuItem, *mygo.Window) { a.mode, a.layout = 0, 1; a.saveLayout() }},
			{Label: "时间轴", Accelerator: "CmdOrCtrl+3", Click: func(*mygo.MenuItem, *mygo.Window) { a.mode = 1 }},
			mygo.Separator(),
			{Label: "照片信息", Accelerator: "CmdOrCtrl+Shift+I", Click: func(*mygo.MenuItem, *mygo.Window) { a.inspectorOpen = !a.inspectorOpen }},
			{Label: "搜索照片…", Accelerator: "CmdOrCtrl+K", Click: func(*mygo.MenuItem, *mygo.Window) {
				a.focusSearch = true
				if a.mode == 2 {
					a.mode = 0
				}
			}},
			{Label: "命令面板…", Accelerator: "CmdOrCtrl+Shift+K", Click: func(*mygo.MenuItem, *mygo.Window) { a.openCommand() }},
			mygo.Separator(),
			{Label: "小缩略图", Accelerator: "CmdOrCtrl+9", Click: func(*mygo.MenuItem, *mygo.Window) { a.set.ThumbSize = 0; a.saveSettings() }},
			{Label: "中缩略图", Accelerator: "CmdOrCtrl+0", Click: func(*mygo.MenuItem, *mygo.Window) { a.set.ThumbSize = 1; a.saveSettings() }},
			{Label: "大缩略图", Click: func(*mygo.MenuItem, *mygo.Window) { a.set.ThumbSize = 2; a.saveSettings() }},
			mygo.Separator(),
			{Label: "全选", Accelerator: "CmdOrCtrl+A", Click: func(*mygo.MenuItem, *mygo.Window) { a.selectAll() }},
			{Label: "全屏查看", Accelerator: "CmdOrCtrl+Ctrl+F", Click: func(*mygo.MenuItem, *mygo.Window) { a.toggleFullScreen() }},
			mygo.Separator(),
			{Label: "设置…", Accelerator: "CmdOrCtrl+,", Click: func(*mygo.MenuItem, *mygo.Window) { a.settingsOpen = true }},
		}},
		{Role: mygo.RoleWindowMenu},
	})
}

// addFolder asks for folders to watch and adds them to the library.
func (a *app) addFolder() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:            a.win,
		Title:             "导入文件夹",
		ButtonLabel:       "导入",
		Message:           "Toki 会监视这些文件夹，新照片自动出现。原文件不会被移动。",
		Directory:         true,
		Multiple:          true,
		CreateDirectories: false,
	})
	if err != nil || len(paths) == 0 {
		return
	}
	added := false
	for _, p := range paths {
		if a.lib.AddFolder(p) {
			added = true
			a.watchFolder(p)
		}
	}
	if !added {
		a.status = "这些文件夹已经在监视中"
		return
	}
	a.save()
	a.viewKey = ""
	a.refresh()
}

// prewarm makes the thumbnails and small previews the views will ask for, in
// the background: an import prepares itself instead of showing placeholders
// one by one when the user first scrolls. It skips anything already on disk.
func (a *app) prewarm(photos []*Photo) {
	if len(photos) == 0 || !uiReady.Load() {
		return
	}
	type job struct {
		path string
		mod  int64
		size int
		edit EditState
	}
	sizes := []int{a.thumbPixels(), previewSmall}
	jobs := make([]job, 0, len(photos)*len(sizes))
	for _, p := range photos {
		edit := p.Edit.Clone()
		for _, size := range sizes {
			jobs = append(jobs, job{path: p.Path, mod: p.ModUnix, size: size, edit: edit})
		}
	}
	go func() {
		mygo.RunOnMain(func() {
			a.preparing = true
			a.refreshUI()
		})
		for _, j := range jobs {
			a.thumbs.Prewarm(j.path, j.mod, j.size, j.edit)
		}
		for a.thumbs.PrewarmPending() > 0 {
			time.Sleep(150 * time.Millisecond)
		}
		mygo.RunOnMain(func() {
			a.preparing = false
			a.refreshUI()
		})
	}()
}

// refresh scans every watched folder again, without blocking the window.
func (a *app) refresh() {
	if a.scanning {
		return
	}
	folders := a.lib.FolderList()
	if len(folders) == 0 {
		a.status = "还没有被监视的文件夹"
		return
	}
	a.scanning = true
	a.status = ""
	go func() {
		var total scanStats
		for _, dir := range folders {
			dir := dir
			st := a.scanFolder(dir, func(done int) {
				if done%200 != 0 {
					return
				}
				mygo.RunOnMain(func() {
					if a.win != nil {
						a.scanMsg = fmt.Sprintf("%s · %d", filepath.Base(dir), done)
						a.refreshUI()
					}
				})
			})
			total.Added += st.Added
			total.Updated += st.Updated
			total.Skipped += st.Skipped
			total.Duplicates += st.Duplicates
			total.Errors = append(total.Errors, st.Errors...)
		}
		gone := a.prune()
		a.save()
		photos := a.lib.Snapshot()
		mygo.RunOnMain(func() {
			a.scanning = false
			a.scanMsg = ""
			a.viewKey = ""
			a.status = fmt.Sprintf("扫描完成：新增 %d · 更新 %d · 未变 %d · 重复跳过 %d · 移除失效 %d",
				total.Added, total.Updated, total.Skipped, total.Duplicates, gone)
			a.prewarm(photos)
			a.refreshUI()
		})
	}()
}

// selectAll chooses every photo shown.
func (a *app) selectAll() {
	a.sel.Clear()
	for _, p := range a.photos {
		a.sel.Add(p.Path)
	}
}

// openPhoto shows one photo alone.
func (a *app) openPhoto(path string) {
	a.current = path
	a.mode = 2
	a.editOpen = false
}

// starSelection gives every chosen photo n stars, or none for 0.
func (a *app) starSelection(n int) {
	for _, p := range a.target() {
		if n == 1 && p.Stars > 0 {
			p.Stars = 0 // a second click on the star clears it
		} else {
			p.Stars = n
		}
	}
	a.save()
}

// forgetSelected takes photos out of the library, leaving the files alone.
func (a *app) forgetSelected() {
	photos := a.target()
	if len(photos) == 0 {
		return
	}
	if a.set.ConfirmForget {
		msg := fmt.Sprintf("把 %d 张照片移出库？文件会留在磁盘上，不会被删除。", len(photos))
		if len(photos) == 1 {
			msg = fmt.Sprintf("把「%s」移出库？文件会留在磁盘上，不会被删除。", photos[0].Name)
		}
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Type:    mygo.MessageQuestion,
			Title:   "移出库",
			Message: msg,
			Buttons: []string{"移出库", "取消"},
			Parent:  a.win,
		})
		if err != nil || res.Button != 0 {
			return
		}
	}
	var paths []string
	for _, p := range photos {
		paths = append(paths, p.Path)
		a.sel.Remove(p.Path)
	}
	a.lib.Forget(paths)
	a.current = ""
	if a.mode == 2 {
		a.mode = 0
	}
	a.viewKey = ""
	a.save()
	a.status = fmt.Sprintf("已移出 %d 张", len(paths))
}

// writeMetadataForSelection writes captions, tags and stars into the files
// themselves, so another program reads the same tags.
func (a *app) writeMetadataForSelection() {
	photos := a.target()
	if len(photos) == 0 {
		return
	}
	go func() {
		failed := 0
		for _, p := range photos {
			if err := writeMetadata(p.Path, p.Caption, p.Tags, p.Stars); err != nil {
				failed++
			}
		}
		mygo.RunOnMain(func() {
			if failed > 0 {
				a.status = fmt.Sprintf("写入元数据：%d 成功，%d 失败", len(photos)-failed, failed)
			} else {
				a.status = fmt.Sprintf("写入元数据：%d 张完成", len(photos))
			}
			a.refreshUI()
		})
	}()
}

// openExport opens the export dialog with the chosen photos.
func (a *app) openExport() {
	if len(a.target()) == 0 {
		a.status = "先选中照片再导出"
		return
	}
	a.exportOpen = true
}

// toggleFullScreen shows or leaves full screen.
func (a *app) toggleFullScreen() {
	if a.win == nil {
		return
	}
	if a.mode != 2 && a.current == "" {
		if len(a.photos) > 0 {
			a.openPhoto(a.photos[0].Path)
		}
	}
	a.win.SetFullScreen(!a.win.IsFullScreen())
}

// revealInFinder shows a photo in the desktop's file browser.
func revealInFinder(path string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", "-R", path).Start()
	case "windows":
		exec.Command("explorer", "/select,", path).Start()
	default:
		exec.Command("xdg-open", filepath.Dir(path)).Start()
	}
}

// watched is the file watch running for each watched folder, and the lock
// that keeps the map safe while a scan adds a folder.
var (
	watched   = map[string]context.CancelFunc{}
	watchedMu sync.Mutex
)

// confirmDeleteTag asks before a tag is taken off every photo.
func (a *app) confirmDeleteTag(tag string) {
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageQuestion,
		Title:   "删除标签",
		Message: fmt.Sprintf("删除标签「%s」？\n所有照片上的这个标签都会被移除，照片本身不会删除。", tag),
		Buttons: []string{"删除", "取消"},
		Parent:  a.win,
	})
	if err != nil || res.Button != 0 {
		return
	}
	a.deleteTag(tag)
}

// deleteTag removes a tag from the library and from every photo.
func (a *app) deleteTag(tag string) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return
	}
	a.lib.DeleteTag(tag)
	if a.nav == "tag:"+tag {
		a.navigate("all")
	} else {
		a.viewKey = ""
	}
	a.save()
	a.status = "已删除标签「" + tag + "」"
}

// confirmRemoveFolder asks before a watched folder is taken out of the
// library. It is explicit that no file is deleted.
func (a *app) confirmRemoveFolder(dir string) {
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Type:  mygo.MessageQuestion,
		Title: "移除文件夹",
		Message: fmt.Sprintf("把「%s」从资料库中移除？\n\n"+
			"只会移除 Toki 里的照片记录，不会删除磁盘上的任何文件；之后可以重新导入。", filepath.Base(dir)),
		Buttons: []string{"移除文件夹", "取消"},
		Parent:  a.win,
	})
	if err != nil || res.Button != 0 {
		return
	}
	a.removeFolder(dir)
}

// removeFolder stops watching a folder and forgets its photos. The files stay
// on disk and the folder can be imported again.
func (a *app) removeFolder(dir string) {
	a.lib.RemoveFolder(dir)
	a.unwatchFolder(dir)
	if a.nav == "folder:"+dir {
		a.navigate("all")
	} else {
		a.viewKey = ""
	}
	a.save()
	a.status = "已移除文件夹「" + filepath.Base(dir) + "」，本地文件没有被删除"
}

// unwatchFolder stops the file watch of a folder that left the library.
func (a *app) unwatchFolder(dir string) {
	watchedMu.Lock()
	defer watchedMu.Unlock()
	if cancel, ok := watched[dir]; ok {
		cancel()
		delete(watched, dir)
	}
}

// watchFolder watches a folder, so a photo copied or taken into it shows up
// without the user asking for a scan.
func (a *app) watchFolder(dir string) {
	watchedMu.Lock()
	if _, running := watched[dir]; running {
		watchedMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	watchedMu.Unlock()

	w, err := mygo.WatchFiles(ctx, dir, mygo.FileWatchOptions{
		Recursive: true,
		Debounce:  600 * time.Millisecond,
		MaxDelay:  3 * time.Second,
		Ignore: func(path string, isDir bool) bool {
			base := strings.ToLower(filepath.Base(path))
			if isDir {
				return strings.HasPrefix(base, ".") || skipDirs[base]
			}
			return strings.HasPrefix(base, ".")
		},
	})
	if err != nil {
		cancel()
		return
	}
	watchedMu.Lock()
	watched[dir] = cancel
	watchedMu.Unlock()

	go func() {
		for {
			events, err := w.Next(ctx)
			if err != nil {
				return
			}
			interesting := false
			for _, e := range events {
				if e.IsDir {
					continue
				}
				if isImagePath(e.Path) || isImagePath(e.OldPath) {
					interesting = true
					break
				}
			}
			if !interesting {
				continue
			}
			st := a.scanFolder(dir, nil)
			if st.Added == 0 && st.Updated == 0 {
				continue
			}
			a.save()
			added := st.Added
			photos := a.lib.Snapshot()
			mygo.RunOnMain(func() {
				a.viewKey = ""
				if added > 0 {
					a.status = fmt.Sprintf("监视 %s：新增 %d 张", filepath.Base(dir), added)
					a.prewarm(photos)
				}
				a.refreshUI()
			})
		}
	}()
}

// watchAll starts watching every folder the library has.
func (a *app) watchAll() {
	for _, dir := range a.lib.FolderList() {
		a.watchFolder(dir)
	}
}
