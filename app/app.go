package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// settings are the preferences the settings window changes, saved beside the
// library.
type settings struct {
	ThumbSize      int    `json:"thumb_size"`   // 0 small, 1 medium, 2 large
	Sort           string `json:"sort"`         // date, dateasc, name, stars
	DefaultView    string `json:"default_view"` // grid, timeline
	Theme          string `json:"theme"`        // "", dark, light
	Layout         string `json:"layout"`       // grid, masonry
	Dedupe         bool   `json:"dedupe"`       // skip pictures seen under another path
	ConfirmForget  bool   `json:"confirm_forget"`
	ConfirmUnwatch bool   `json:"confirm_unwatch"` // ask before a folder leaves the library
	WriteMetadata  bool   `json:"write_metadata"`  // write captions, tags and stars into the files
	ShowNames      bool   `json:"show_names"`
	ExportSize     int    `json:"export_size"`
	ExportQuality  int    `json:"export_quality"`
}

// defaultSettings are the preferences a new installation starts with.
func defaultSettings() settings {
	return settings{
		ThumbSize:      1,
		Sort:           "date",
		DefaultView:    "grid",
		Theme:          "",
		Layout:         "grid",
		Dedupe:         true,
		ConfirmForget:  true,
		ConfirmUnwatch: true,
		WriteMetadata:  false,
		ShowNames:      true,
		ExportSize:     0,
		ExportQuality:  1,
	}
}

// exportForm holds the choices of an export.
type exportForm struct {
	Dir       string
	Folder    string
	Size      int
	Numbers   bool
	Watermark string
}

// app is the whole application: its library, what the window shows and the
// choices the user is making.
type app struct {
	lib     *Library
	thumbs  *thumbCache
	win     *mygo.Window
	set     settings
	dataDir string

	// The look: the tokens of the frame, whether the system asked for a dark
	// one, and the palette the frame draws with.
	pal           *palette
	systemDark    bool
	themeResolved bool

	// What the library shows.
	nav      string // all, stars, recent, favorites, edited, album:<id>, folder:<path>, tag:<tag>, year:<year>
	mode     int    // 0 library, 1 timeline, 2 one photo
	layout   int    // 0 grid, 1 masonry
	filter   string // all, favorite, edited, rated
	query    string
	minStars int
	sel      ui.Selection[string]
	photos   []*Photo
	viewKey  string

	// Derived data. A frame only reads these: the library index and the flow
	// are rebuilt when their inputs change, not once per redraw.
	dataRev    atomic.Uint64
	libData    *libraryData
	photoRev   uint64
	photoIndex map[string]int
	flowCache  *flowCache
	railCache  *railCache

	// The thumbnail size actually laid out, sprung toward the segmented choice.
	thumbShown     float32
	thumbVel       float64
	thumbAnimLast  time.Time
	thumbAnimating bool

	// The date rail's drag, the timeline's and the filmstrip's scroll.
	railDrag          bool
	railGrabY         float32
	timelineScroll    ui.ScrollState
	timelineScrollRef ui.Handle
	timelineRef       ui.Handle
	filmScroll        ui.ScrollState
	filmScrollRef     ui.Handle
	filmBoxRef        ui.Handle
	// The filmstrip scrolls to the photo shown, once per switch.
	filmReveal bool

	// The lightbox: one photo over a dark mask, zoomed with the wheel.
	lightbox      bool
	lightZoom     float64
	lightX        float64
	lightY        float64
	lightDragging bool
	lightMoved    bool
	lightDragX    float32
	lightDragY    float32
	lightWasFull  bool
	lightDispW    float32
	lightDispH    float32

	// The zoom actually drawn, eased toward the target set by the wheel and
	// the buttons, so a notch never snaps the picture.
	lightZoomShown float64
	lightXShown    float64
	lightYShown    float64
	lightZoomVel   float64
	lightXVel      float64
	lightYVel      float64
	lightAnimLast  time.Time
	lightAnimating bool

	// The animated picture playing, if the photo shown is a GIF.
	anim gifShow

	// The confetti of the last favorite, while it flies.
	fetti *confettiBurst

	// The photo shown alone, and how.
	current         string
	zoom            int // 0 fit, 1 own size
	draft           EditState
	draftFor        string
	editOpen        bool
	editTab         int
	slideshow       bool
	slideshowPaused bool
	slideAt         time.Time

	// The shell and the command palette.
	inspectorOpen bool
	commandOpen   bool
	commandQuery  string
	focusSearch   bool
	libScroll     ui.ScrollState
	scrollRef     ui.Handle
	flowRef       ui.Handle

	// Dialogs.
	exportOpen   bool
	settingsOpen bool
	albumOpen    bool
	newAlbumOpen bool
	albumName    string
	albumDesc    string
	tagEntry     string
	albumPick    string
	tagOpen      bool
	tagText      string
	form         exportForm

	// A photo opened from the file manager but not in the library, the folder
	// it came from and the photos in it: the viewer walks that folder, not the
	// library, and nothing is added to the library unless it is favorited.
	external      *Photo
	externalList  []*Photo
	externalIndex map[string]int
	externalFor   string
	openMu        sync.Mutex
	pendingOpen   []string

	// A precise (high-resolution) wheel accumulates deltas until one notch.
	wheelDelta float64

	// Work in progress.
	scanning       bool
	scanMsg        string
	preparing      bool
	prepareDone    int
	prepareTotal   int
	prewarmRunning atomic.Bool
	status         string
}

// newApp makes the app with its library loaded.
func newApp() *app { return newAppAt(appDataDir()) }

// newAppAt makes the app keeping its library, settings and thumbnails in dir.
func newAppAt(dataDir string) *app {
	a := &app{
		lib:           NewLibrary(filepath.Join(dataDir, "library.json")),
		dataDir:       dataDir,
		set:           defaultSettings(),
		nav:           "all",
		filter:        "all",
		inspectorOpen: true,
		form:          exportForm{},
	}
	if err := a.lib.Load(); err != nil {
		log.Printf("library: %v", err)
	}
	a.loadSettings()
	a.form.Size = a.set.ExportSize
	a.form.Dir = filepath.Join(homeDir(), "Pictures", "Toki Export")
	a.form.Folder = "libra-export"
	a.thumbs = newThumbCache(filepath.Join(dataDir, "thumbs"), a.refreshUI)
	if a.set.DefaultView == "timeline" {
		a.mode = 1
	}
	if a.set.Layout == "masonry" {
		a.layout = 1
	}
	// The system's appearance is the one to start from; the first frame reads
	// it, before the app's own theme replaces it.
	a.systemDark = true
	// TOKI_SEED adds a folder to watch on a fresh installation, which is how
	// the app is tried out on a demo folder; LIBRA_SEED is the name it had.
	seed := os.Getenv("TOKI_SEED")
	if seed == "" {
		seed = os.Getenv("LIBRA_SEED")
	}
	if seed != "" && len(a.lib.FolderList()) == 0 {
		if a.lib.AddFolder(seed) {
			a.save()
			a.refresh()
		}
	}
	return a
}

// appDataDir is where the library, the settings and the thumbnails live.
func appDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "Toki")
	}
	dir := filepath.Join(base, "Toki")
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	// The app used to be called Libra: a library made then keeps working
	// where it is, instead of starting empty under the new name.
	legacy := filepath.Join(base, "Libra")
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	return dir
}

// homeDir is the user's home folder.
func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

// settingsPath is where the preferences are saved.
func (a *app) settingsPath() string { return filepath.Join(a.dataDir, "settings.json") }

// loadSettings reads the preferences, keeping the defaults for what is missing.
func (a *app) loadSettings() {
	b, err := os.ReadFile(a.settingsPath())
	if err != nil {
		return
	}
	// The defaults are read first, so a preference added in a later version
	// starts at its default for a library saved before it existed.
	s := defaultSettings()
	if json.Unmarshal(b, &s) == nil {
		if s.Sort == "" {
			s.Sort = defaultSettings().Sort
		}
		a.set = s
	}
}

// saveSettings writes the preferences.
func (a *app) saveSettings() {
	b, err := json.MarshalIndent(a.set, "", "  ")
	if err != nil {
		return
	}
	os.MkdirAll(a.dataDir, 0o755)
	os.WriteFile(a.settingsPath(), b, 0o644)
}

// save writes the library and invalidates every derived view of it.
func (a *app) save() {
	if err := a.lib.Save(); err != nil {
		a.status = "保存失败: " + err.Error()
	}
	a.dataRev.Add(1)
}

// winReady says whether the window exists and the app loop runs, so work
// finishing early, before the first frame, cannot ask a window that is not
// there yet to draw.
var winReady atomic.Bool

// refreshUI asks the window to draw again, when there is one.
func (a *app) refreshUI() {
	if a.win == nil || !winReady.Load() {
		return
	}
	a.win.Invalidate()
}

// sortNames and sortKeys are the orders the toolbar and the settings offer,
// in the same order.
var sortNames = []string{"按日期（新→旧）", "按日期（旧→新）", "按名称", "按星标"}
var sortKeys = []string{"date", "dateasc", "name", "stars"}

// sortName is the label of an order.
func sortName(key string) string {
	for i, k := range sortKeys {
		if k == key {
			return sortNames[i]
		}
	}
	return sortNames[0]
}

// sortKey is the order a label names.
func sortKey(name string) string {
	for i, n := range sortNames {
		if n == name {
			return sortKeys[i]
		}
	}
	return sortKeys[0]
}

// thumbPixels is the pixel size of a thumbnail, by the preference.
func (a *app) thumbPixels() int {
	switch a.set.ThumbSize {
	case 0:
		return 160
	case 2:
		return 480
	}
	return 288
}

// rebuild refreshes the list of photos shown, when what decides it changed.
func (a *app) rebuild() {
	key := strings.Join([]string{a.nav, a.query, itoa(a.minStars), a.set.Sort, a.filter, itoa(len(a.lib.Photos))}, "\x00")
	if key == a.viewKey {
		return
	}
	a.viewKey = key
	// A rebuild may follow an in-place change to a photo (a new star, a new
	// tag), so the library index has to be taken again before it is read.
	a.dataRev.Add(1)
	all := a.library().all
	out := make([]*Photo, 0, len(all))
	for _, p := range all {
		if !a.inNav(p) || !a.matches(p) || !a.inFilter(p) {
			continue
		}
		out = append(out, p)
	}
	switch a.set.Sort {
	case "name":
		sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	case "stars":
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Stars != out[j].Stars {
				return out[i].Stars > out[j].Stars
			}
			return out[i].TakenUnix > out[j].TakenUnix
		})
	case "dateasc":
		sort.SliceStable(out, func(i, j int) bool { return out[i].TakenUnix < out[j].TakenUnix })
	}
	a.photos = out
	a.photoRev++
	a.photoIndex = make(map[string]int, len(out))
	for i, p := range out {
		a.photoIndex[p.Path] = i
	}
	// A photo no longer shown cannot stay chosen. Only the chosen paths are
	// walked, not the whole library.
	for path := range a.sel.All() {
		if _, ok := a.photoIndex[path]; !ok {
			a.sel.Remove(path)
		}
	}
}

// inNav reports whether the photo belongs to the place chosen in the sidebar.
func (a *app) inNav(p *Photo) bool {
	switch {
	case a.nav == "" || a.nav == "all":
		return true
	case a.nav == "stars":
		return p.Stars > 0
	case a.nav == "recent":
		return p.TakenUnix >= time.Now().AddDate(0, 0, -30).Unix()
	case a.nav == "favorites":
		return p.Favorite
	case a.nav == "edited":
		return !p.Edit.IsZero()
	case strings.HasPrefix(a.nav, "album:"):
		return p.InAlbum(strings.TrimPrefix(a.nav, "album:"))
	case strings.HasPrefix(a.nav, "folder:"):
		return strings.HasPrefix(p.Path, strings.TrimPrefix(a.nav, "folder:")+string(filepath.Separator))
	case strings.HasPrefix(a.nav, "tag:"):
		return p.HasTag(strings.TrimPrefix(a.nav, "tag:"))
	case strings.HasPrefix(a.nav, "year:"):
		return itoa(p.Year()) == strings.TrimPrefix(a.nav, "year:")
	}
	return true
}

// inFilter reports whether the photo passes the filter chosen in the toolbar.
func (a *app) inFilter(p *Photo) bool {
	switch a.filter {
	case "favorite":
		return p.Favorite
	case "edited":
		return !p.Edit.IsZero()
	case "rated":
		return p.Stars >= 4
	}
	return true
}

// matches reports whether the photo passes the search and the star filter.
func (a *app) matches(p *Photo) bool {
	if p.Stars < a.minStars {
		return false
	}
	q := strings.ToLower(strings.TrimSpace(a.query))
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.Caption), q) {
		return true
	}
	for _, t := range p.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(filepath.Dir(p.Path)), q)
}

// containsPhoto reports whether paths holds path.
func containsPhoto(photos []*Photo, path string) bool {
	for _, p := range photos {
		if p.Path == path {
			return true
		}
	}
	return false
}

// selected returns the photos the user chose, in the order they show. It
// walks the choice, not the whole library: selecting two of ten thousand
// photos must not build ten thousand elements.
func (a *app) selected() []*Photo {
	n := a.sel.Len()
	if n == 0 {
		return nil
	}
	data := a.library()
	out := make([]*Photo, 0, n)
	for path := range a.sel.All() {
		if _, ok := a.photoIndex[path]; !ok {
			continue
		}
		if p := data.byPath[path]; p != nil {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return a.photoIndex[out[i].Path] < a.photoIndex[out[j].Path]
	})
	return out
}

// target returns the photos an action applies to: the chosen ones, or the
// photo shown alone.
func (a *app) target() []*Photo {
	if sel := a.selected(); len(sel) > 0 {
		return sel
	}
	if p := a.shownPhoto(); p != nil {
		return []*Photo{p}
	}
	return nil
}

// view builds the window: the title bar over the three panels.
func (a *app) view(c *ui.Context) {
	a.rebuild()
	a.drainOpenedFiles()
	a.applyTheme(c)
	if a.lightbox {
		a.viewerShortcuts(c)
		a.viewViewer(c)
		a.confettiLayer(c)
		return
	}
	a.shortcuts(c)
	ui.Column(c).Fill().Children(func() {
		a.titlebar(c)
		// A narrow window folds the inspector away, as the prototype does:
		// the photos need the room more than the properties do.
		winW, _ := c.Size()
		showInspector := a.inspectorOpen && winW >= 1000 && !(a.mode == 2 && a.editOpen)
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c)
			a.hairline(c, true)
			a.content(c)
			if showInspector {
				a.hairline(c, true)
				a.inspector(c)
			}
		})
		a.statusbar(c)
	})
	a.dialogs(c)
	a.commandPalette(c)
	a.confettiLayer(c)
}

// content is the middle column: the library, the timeline or one photo.
func (a *app) content(c *ui.Context) {
	switch a.mode {
	case 1:
		a.timelineView(c)
	case 2:
		a.oneUp(c)
	default:
		a.libraryView(c)
	}
}

// shortcuts handles the keyboard: the menu shows the same keys beside its
// items, where they apply to the whole app.
func (a *app) shortcuts(c *ui.Context) {
	if c.Shortcut(ui.Cmd, ui.KeyO) {
		a.addFolder()
	}
	if c.Shortcut(ui.Cmd, ui.KeyR) {
		a.refresh()
	}
	if c.Shortcut(ui.Cmd, ui.KeyE) {
		a.openExport()
	}
	if c.Shortcut(ui.Cmd, ui.KeyK) {
		// Command-K puts the keyboard in the search field, which filters the
		// library as it is typed.
		a.focusSearch = true
		if a.mode == 2 {
			a.mode = 0
		}
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyK) {
		a.openCommand()
	}
	if c.Shortcut(ui.Cmd, ui.Key1) {
		a.mode, a.layout = 0, 0
		a.saveLayout()
	}
	if c.Shortcut(ui.Cmd, ui.Key2) {
		a.mode, a.layout = 0, 1
		a.saveLayout()
	}
	if c.Shortcut(ui.Cmd, ui.Key3) {
		a.mode = 1
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyL) {
		a.toggleTheme()
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyI) {
		a.inspectorOpen = !a.inspectorOpen
	}
	if c.Shortcut(0, ui.KeyEscape) {
		if a.query != "" {
			a.query = ""
			a.viewKey = ""
		} else if a.mode == 2 {
			a.mode = 0
			a.current = ""
		}
	}
	if c.Shortcut(ui.Cmd, ui.KeyA) && a.mode != 2 {
		a.selectAll()
	}
	if c.Shortcut(0, ui.KeyF) && a.mode != 2 && !c.FocusTakesText() {
		a.toggleFavoriteCurrent(c)
	}
	if c.Shortcut(ui.Cmd, ui.KeyF) {
		a.toggleFullScreen()
	}
}

// saveLayout keeps the arrangement the user chose.
func (a *app) saveLayout() {
	name := "grid"
	if a.layout == 1 {
		name = "masonry"
	}
	if a.set.Layout != name {
		a.set.Layout = name
		a.saveSettings()
	}
}

// monthLabel is the heading a photo of the timeline shows under.
func monthLabel(p *Photo) string {
	t := p.Taken()
	return fmt.Sprintf("%d 年 %d 月", t.Year(), int(t.Month()))
}

// starString draws n stars.
func starString(n int) string {
	return strings.Repeat("★", n)
}

func main() {
	a := newApp()
	a.registerOpenFiles()
	mygo.App.WhenReady(func() {
		uiReady.Store(true)
		mygo.App.SetMenu(a.menu())
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "Toki",
			Width:     1180,
			Height:    760,
			MinWidth:  780,
			MinHeight: 520,
			StateKey:  "main",
			// The window draws its own title bar under the traffic lights, as
			// the design has it: the mark, the view, the search and import.
			TitleBarStyle:        mygo.TitleBarHidden,
			TrafficLightPosition: &mygo.Point{X: 18, Y: 18},
			Content:              ui.View(a.view),
		})
		winReady.Store(true)
		// Photos copied into a watched folder show up by themselves.
		a.watchAll()
		// Warm the thumbnails of an existing library while the window is up.
		a.prewarm(a.lib.Snapshot())
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
