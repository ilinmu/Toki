package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
)

// A photo opened from the file manager is browsed as its folder: the viewer's
// filmstrip and its arrow keys walk the images beside it, and the library is
// left alone until the photo is favorited. The state of that folder lives on
// app: external, externalList, externalIndex and externalFor.

// shownPhoto is the photo a viewer shows: one of the library, or a file
// opened from the file manager that is not in the library.
func (a *app) shownPhoto() *Photo {
	if p := a.lib.Get(a.current); p != nil {
		return p
	}
	if a.external != nil && a.external.Path == a.current {
		return a.external
	}
	return a.oneOffPhoto(a.current)
}

// shownPhotos is the list the viewer steps through: the photos the library
// shows, or the images of the folder a file was opened from.
func (a *app) shownPhotos() []*Photo {
	if a.externalFor != "" {
		return a.externalList
	}
	return a.photos
}

// shownIndex is where a photo is in shownPhotos.
func (a *app) shownIndex(path string) (int, bool) {
	if a.externalFor != "" {
		i, ok := a.externalIndex[path]
		return i, ok
	}
	i, ok := a.photoIndex[path]
	return i, ok
}

// oneOffPhoto returns the photo at path among the folder opened from the file
// manager, when that is the folder being browsed.
func (a *app) oneOffPhoto(path string) *Photo {
	if a.externalFor == "" {
		return nil
	}
	if i, ok := a.externalIndex[path]; ok {
		return a.externalList[i]
	}
	return nil
}

// leaveOneOff stops browsing the folder of a file opened from the file
// manager, so the viewer goes back to the library's own photos.
func (a *app) leaveOneOff() {
	a.external, a.externalList, a.externalIndex, a.externalFor = nil, nil, nil, ""
}

// browseFolder makes the images of p's folder the photos the viewer walks, in
// the order the file manager lists them. Only the directory itself is read,
// not the folders under it, and nothing is added to the library: this is a
// look at a folder, not an import.
func (a *app) browseFolder(p *Photo) {
	dir := filepath.Dir(p.Path)
	a.leaveOneOff()
	entries, err := os.ReadDir(dir) // by file name
	if err != nil {
		a.externalFor = dir
		a.externalList = []*Photo{p}
		a.externalIndex = map[string]int{p.Path: 0}
		a.external = p
		return
	}
	list := make([]*Photo, 0, len(entries))
	index := make(map[string]int, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		if !isImagePath(path) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		// Every image but the one shown is only named here; that one has its
		// metadata read, so a folder of thousands opens at once.
		index[path] = len(list)
		list = append(list, &Photo{
			Path:      path,
			Name:      name,
			Root:      dir,
			Size:      info.Size(),
			ModUnix:   info.ModTime().Unix(),
			TakenUnix: info.ModTime().Unix(),
		})
	}
	if i, ok := index[p.Path]; ok {
		list[i] = p
	} else {
		list, index = []*Photo{p}, map[string]int{p.Path: 0}
	}
	a.externalFor = dir
	a.externalList = list
	a.externalIndex = index
	a.external = p
}

// enrichOneOff reads the metadata of a photo of a browsed folder the first
// time the viewer shows it.
func (a *app) enrichOneOff(p *Photo) {
	if p.Width > 0 || p.Height > 0 || a.lib.Get(p.Path) != nil {
		return
	}
	full, err := photoFromFile(p.Path)
	if err != nil {
		return
	}
	fav := p.Favorite
	*p = *full
	p.Favorite = fav
}

// queueOpenedFile is mygo.App.OnOpenFile's listener: a double click in the
// file manager, a drop on the Dock icon or a path on the command line. It only
// records the path; the window's next frame opens it on the main thread.
func (a *app) queueOpenedFile(path string) {
	a.openMu.Lock()
	a.pendingOpen = append(a.pendingOpen, path)
	a.openMu.Unlock()
	a.refreshUI()
}

// drainOpenedFiles opens the files queued since the last frame.
func (a *app) drainOpenedFiles() {
	a.openMu.Lock()
	paths := a.pendingOpen
	a.pendingOpen = nil
	a.openMu.Unlock()
	for _, path := range paths {
		a.showOpenedFile(path)
	}
}

// showOpenedFile shows one image in the app's photo viewer. A file already in
// the library keeps its place in the library's photos; an unknown file is
// shown with the other images of its folder, without importing them.
func (a *app) showOpenedFile(path string) {
	abs, err := filepath.Abs(path)
	if err != nil || !isImagePath(abs) {
		return
	}
	if a.lib.Get(abs) != nil {
		a.openViewer(abs)
		return
	}
	p, err := photoFromFile(abs)
	if err != nil {
		a.status = "打不开 " + filepath.Base(abs) + "：" + err.Error()
		return
	}
	a.browseFolder(p)
	a.openViewer(abs)
}

// setFavorite marks a photo a favorite or not. A photo opened from the file
// manager joins the library when it is favorited: a favorite is a photo the
// library keeps, so it must not be a look at a folder that is gone tomorrow.
// Only that photo joins; the rest of its folder stays out. The caller saves:
// a batch of favorites is one write.
func (a *app) setFavorite(p *Photo, on bool) {
	p.Favorite = on
	if on && a.lib.Get(p.Path) == nil {
		a.adoptPhoto(p)
		a.status = "已收藏，这张照片已加入资料库"
	}
}

// adoptPhoto puts one photo opened from the file manager into the library.
// Its folder is not imported: a folder of the library is a folder Toki
// watches, and one favorite is not a request to bring in everything beside
// it.
func (a *app) adoptPhoto(p *Photo) {
	if a.lib.Get(p.Path) != nil {
		return
	}
	a.enrichOneOff(p)
	q := *p
	q.Root = filepath.Dir(p.Path)
	q.Albums, q.Tags = append([]string(nil), p.Tags...), append([]string(nil), p.Albums...)
	a.lib.Upsert(&q)
	if print, err := fingerprint(p.Path); err == nil && print != "" {
		a.lib.RememberPrint(print, p.Path)
	}
}

// keepInLibrary returns the library's photo for p, adding a photo opened from
// the file manager when an action only library photos take asks for it.
func (a *app) keepInLibrary(p *Photo) *Photo {
	if q := a.lib.Get(p.Path); q != nil {
		return q
	}
	a.adoptPhoto(p)
	a.save() // the library index is rebuilt from what was saved
	a.rebuild()
	return a.lib.Get(p.Path)
}

// setStars gives a photo n stars, adding it to the library when it was only
// being looked at.
func (a *app) setStars(p *Photo, n int) {
	if a.lib.Get(p.Path) == nil {
		if n == 0 {
			return
		}
		if p = a.keepInLibrary(p); p == nil {
			return
		}
	}
	p.Stars = n
	a.save()
}

// registerOpenFiles installs the file-manager handler before the app runs,
// so files that launched the app are delivered too.
func (a *app) registerOpenFiles() {
	mygo.App.OnOpenFile(a.queueOpenedFile)
}
