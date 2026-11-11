package main

import (
	"path/filepath"

	"github.com/egoist/mygo"
)

// shownPhoto is the photo a viewer shows: one of the library, or a file
// opened from the file manager that is not in the library.
func (a *app) shownPhoto() *Photo {
	if p := a.lib.Get(a.current); p != nil {
		return p
	}
	if a.external != nil && a.external.Path == a.current {
		return a.external
	}
	return nil
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
// the library keeps its place in the filmstrip; an unknown file is shown as a
// one-off photo without changing the library.
func (a *app) showOpenedFile(path string) {
	abs, err := filepath.Abs(path)
	if err != nil || !isImagePath(abs) {
		return
	}
	if a.lib.Get(abs) != nil {
		a.external = nil
		a.openViewer(abs)
		return
	}
	p, err := photoFromFile(abs)
	if err != nil {
		a.status = "打不开 " + filepath.Base(abs) + "：" + err.Error()
		return
	}
	a.external = p
	a.openViewer(abs)
}

// registerOpenFiles installs the file-manager handler before the app runs,
// so files that launched the app are delivered too.
func (a *app) registerOpenFiles() {
	mygo.App.OnOpenFile(a.queueOpenedFile)
}
