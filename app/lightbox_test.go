package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// screenPoint is where a point of the photo shows: its offset from the middle
// of the photo, times the size the photo is drawn at, plus the pan.
func screenPoint(imgX float64, zoom, pan, fit float64) float64 {
	return imgX*fit*zoom + pan
}

func TestZoomAroundKeepsThePointerOnTheSamePoint(t *testing.T) {
	a := testApp(t)
	areaW, areaH := float32(1000), float32(700)
	const fit = 0.5 // the photo's points in DIPs when fitted

	// Hover a point a quarter to the right of the middle.
	cursorX := float64(areaW)/2 + 120
	imgX := (cursorX - float64(areaW)/2) / fit // the photo point under the pointer

	for _, factor := range []float64{1.15, 1.15 * 1.15, 1 / 1.15} {
		before := a.lightZoom
		a.zoomAround(factor, float32(cursorX), 400, areaW, areaH)
		if a.lightZoom == before && (a.lightZoom == 1 || a.lightZoom == lightboxMaxZoom) {
			continue // already at a limit
		}
		after := float64(areaW)/2 + screenPoint(imgX, a.lightZoom, a.lightX, fit)
		if math.Abs(after-cursorX) > 0.001 {
			t.Fatalf("zoom %v moved the point under the pointer from %v to %v", factor, cursorX, after)
		}
	}
}

func TestZoomLimitsAndReset(t *testing.T) {
	a := testApp(t)
	for i := 0; i < 30; i++ {
		a.zoomAround(2, 500, 350, 1000, 700)
	}
	if a.lightZoom != lightboxMaxZoom {
		t.Fatalf("zoom stopped at %v, want %v", a.lightZoom, lightboxMaxZoom)
	}
	for i := 0; i < 40; i++ {
		a.zoomAround(0.5, 500, 350, 1000, 700)
	}
	if a.lightZoom != lightboxMinZoom {
		t.Fatalf("zoom stopped at %v, want %v", a.lightZoom, lightboxMinZoom)
	}
	if a.lightX != 0 || a.lightY != 0 {
		t.Fatalf("smaller than the area still panned: %v %v", a.lightX, a.lightY)
	}
	// The button that fits it again comes back to 100%.
	a.lightZoom, a.lightX = 1, 0
	if a.lightZoom != 1 {
		t.Fatalf("fit is %v, want 1", a.lightZoom)
	}
}

func TestZoomAtTheMiddleDoesNotPan(t *testing.T) {
	a := testApp(t)
	a.zoomAround(2, 500, 350, 1000, 700)
	if a.lightX != 0 || a.lightY != 0 {
		t.Fatalf("zoom at the middle panned to %v %v", a.lightX, a.lightY)
	}
	if a.lightZoom != 2 {
		t.Fatalf("zoom %v, want 2", a.lightZoom)
	}
}

func TestLightboxFitKeepsTheShape(t *testing.T) {
	// A wide photo in a taller area: its width decides the fit.
	w, h := lightboxFit(2000, 1000, 800, 700, 1)
	if w != 800 || math.Abs(float64(h)-400) > 0.01 {
		t.Fatalf("fit gives %vx%v, want 800x400", w, h)
	}
	// Zooming doubles both.
	w2, h2 := lightboxFit(2000, 1000, 800, 700, 2)
	if math.Abs(float64(w2)-1600) > 0.01 || math.Abs(float64(h2)-800) > 0.01 {
		t.Fatalf("zoom 2 gives %vx%v, want 1600x800", w2, h2)
	}
	// A tall photo in a wide area: its height decides the fit.
	w3, h3 := lightboxFit(1000, 2000, 800, 700, 1)
	if math.Abs(float64(h3)-700) > 0.01 || math.Abs(float64(w3)-350) > 0.01 {
		t.Fatalf("fit gives %vx%v, want 350x700", w3, h3)
	}
}

func TestClampPanKeepsThePhotoInTheArea(t *testing.T) {
	a := testApp(t)
	// Zoomed twice: the photo is 1600x800 in an 800x700 area, so it may pan
	// 400 across and 50 down.
	a.lightX, a.lightY = 5000, -5000
	a.clampPan(1600, 800, 800, 700)
	if a.lightX != 400 {
		t.Fatalf("horizontal pan %v, want 400", a.lightX)
	}
	if a.lightY != -50 {
		t.Fatalf("vertical pan %v, want -50", a.lightY)
	}
	// A fitted photo cannot be dragged at all.
	a.lightX, a.lightY = 100, 100
	a.clampPan(800, 700, 800, 700)
	if a.lightX != 0 || a.lightY != 0 {
		t.Fatalf("fit panned to %v %v", a.lightX, a.lightY)
	}
}

func TestInsidePhoto(t *testing.T) {
	// A 400x300 photo in the middle of a 1000x700 area.
	if !insidePhoto(500, 350, 1000, 700, 400, 300, 0, 0) {
		t.Fatal("the middle of the photo is outside it")
	}
	if insidePhoto(100, 100, 1000, 700, 400, 300, 0, 0) {
		t.Fatal("a corner of the mask is inside the photo")
	}
	// Panned right by 100: what was at 350 now sits at 450.
	if insidePhoto(350, 350, 1000, 700, 400, 300, 100, 0) {
		t.Fatal("the pan was ignored")
	}
	if !insidePhoto(600, 350, 1000, 700, 400, 300, 100, 0) {
		t.Fatal("the panned photo does not reach the point under it")
	}
}

func TestLightboxRenders(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "beach.png", 80, 60, 40)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()

	a.openLightbox(path)
	if !a.lightbox {
		t.Fatal("the lightbox did not open")
	}
	view := ui.NewTester(a.view, 1180, 760)
	for _, want := range []string{"beach.png", "1 / 1", "适应窗口", "‹ 上一张", "下一张 ›"} {
		if !view.HasText(want) {
			t.Fatalf("the lightbox lacks %q: %q", want, view.Texts())
		}
	}
	// The grid is not behind it: the mask replaced the window's content.
	if view.HasText("全部照片") {
		t.Fatal("the sidebar is still drawn behind the lightbox")
	}

	// Esc, through the view, closes it again.
	view.Key(0, ui.KeyEscape)
	view.Frame()
	if a.lightbox {
		t.Fatal("Escape did not close the lightbox")
	}
	back := ui.NewTester(a.view, 1180, 760)
	if !back.HasText("全部照片") {
		t.Fatalf("the grid did not come back: %q", back.Texts())
	}
}

func TestLightboxDoubleClickOpensIt(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	writePNG(t, dir, "double.png", 60, 40, 70)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()

	view := ui.NewTester(a.view, 1100, 720)
	// The first cell's middle: a double click shows it in the lightbox.
	rect, ok := view.Find("double.png")
	if !ok {
		t.Fatalf("the thumbnail is not on the screen: %q", view.Texts())
	}
	view.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	view.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	view.Frame()
	if !a.lightbox {
		t.Fatal("a double click did not open the lightbox")
	}
	if a.current != filepath.Join(dir, "double.png") {
		t.Fatalf("the lightbox shows %q", a.current)
	}
}

func TestWheelZoomsTheLightbox(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "wheel.png", 200, 100, 90)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()
	a.openLightbox(path)

	view := ui.NewTester(a.view, 1000, 700)
	if a.lightZoom != 1 {
		t.Fatalf("a fresh lightbox is zoomed %v, want 1", a.lightZoom)
	}
	// A notch up over the middle of the picture enlarges it.
	view.Wheel(500, 350, 0, -1)
	view.Frame()
	if a.lightZoom <= 1 {
		t.Fatalf("the wheel did not zoom in: %v", a.lightZoom)
	}
	first := a.lightZoom
	// And down again, below the fit: a photo can be shown smaller than the
	// area, down to the smallest zoom.
	for i := 0; i < 3; i++ {
		view.Wheel(500, 350, 0, 1)
		view.Frame()
	}
	if a.lightZoom >= first {
		t.Fatalf("the wheel did not zoom out: %v then %v", first, a.lightZoom)
	}
	if a.lightZoom != 1 {
		// Three notches from 1.32 go below the fit.
		t.Logf("zoomed out to %v", a.lightZoom)
	}
	if a.lightX != 0 || a.lightY != 0 {
		t.Fatalf("smaller than the area panned to %v %v", a.lightX, a.lightY)
	}
	// Its button fits it again.
	if err := view.Click("适应窗口"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if a.lightZoom != 1 {
		t.Fatalf("fit left the zoom at %v, want 1", a.lightZoom)
	}
}

func TestZoomGoesBelowTheFit(t *testing.T) {
	a := testApp(t)
	a.lightZoom = 1
	for i := 0; i < 5; i++ {
		a.zoomAround(1/1.15, 500, 350, 1000, 700)
	}
	if a.lightZoom >= 1 {
		t.Fatalf("zoom is %v, want less than the fit", a.lightZoom)
	}
	w, h := lightboxFit(2000, 1000, 800, 700, a.lightZoom)
	if w >= 800 {
		t.Fatalf("the photo is %v wide, want less than the 800 area", w)
	}
	if h <= 0 || w <= 0 {
		t.Fatalf("the photo is %vx%v", w, h)
	}
}

func TestClickingAThumbnailChoosesIt(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	first := writePNG(t, dir, "one.png", 60, 40, 10)
	second := writePNG(t, dir, "two.png", 60, 40, 200)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()

	view := ui.NewTester(a.view, 1100, 720)
	rect, ok := view.Find("one.png")
	if !ok {
		t.Fatalf("no thumbnail: %q", view.Texts())
	}
	view.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	view.Frame()
	if a.current != first {
		t.Fatalf("a click chose %q, want %q", a.current, first)
	}
	if !a.sel.Has(first) || a.sel.Len() != 1 {
		t.Fatalf("a click did not choose it for batch actions: %d chosen", a.sel.Len())
	}
	if a.lightbox {
		t.Fatal("a single click opened the lightbox")
	}

	// A second click on another photo chooses that one instead.
	rect2, ok := view.Find("two.png")
	if !ok {
		t.Fatalf("no second thumbnail: %q", view.Texts())
	}
	view.ClickAt(rect2.X+rect2.W/2, rect2.Y+rect2.H/2)
	view.Frame()
	if a.current != second {
		t.Fatalf("the second click chose %q", a.current)
	}
	if a.lightbox {
		t.Fatal("a single click opened the lightbox")
	}
}

// writeJPEG writes a big photo, as a camera would.
func writeJPEG(t *testing.T, dir, name string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y += 4 {
		for x := 0; x < w; x += 4 {
			c := color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255}
			img.Set(x, y, c)
			img.Set(x+1, y, c)
			img.Set(x, y+1, c)
			img.Set(x+1, y+1, c)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

// TestLightboxOpensAtOnce guards the interaction the double click starts: the
// first frame must not decode an original nor read a large preview from the
// disk, which made opening a photo take seconds.
func TestLightboxOpensAtOnce(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writeJPEG(t, dir, "big.jpg", 4000, 3000)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()

	start := time.Now()
	a.openLightbox(path)
	view := ui.NewTester(a.view, 1180, 760)
	view.Frame()
	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Fatalf("the first lightbox frame took %v: something decoded on the main thread", elapsed)
	}
	// It shows the mask and the bars even while the picture is made.
	if !view.HasText("big.jpg") {
		t.Fatalf("the lightbox did not draw: %q", view.Texts())
	}
}

// makeThumbnailOnDisk writes the thumbnail the grid would have made, so a test
// has the bitmap the grid shows without a window to draw it.
func makeThumbnailOnDisk(t *testing.T, a *app, p *Photo, size int) {
	t.Helper()
	key := thumbKey(p.Path, p.ModUnix, int64(size), p.Edit)
	a.thumbs.make(thumbJob{key: key, path: p.Path, mod: p.ModUnix, size: size, edit: p.Edit})
	if _, ok := a.thumbs.Bitmap(p.Path, p.ModUnix, size, p.Edit); !ok {
		t.Fatal("the thumbnail was not made")
	}
}

// TestOneUpShowsAPictureAtOnce is the stepping complaint: moving to a photo
// never seen before must show what the grid has, not a blank area.
func TestOneUpShowsAPictureAtOnce(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	writePNG(t, dir, "first.png", 60, 40, 10)
	next := writePNG(t, dir, "second.png", 60, 40, 200)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()

	// The grid drew the second photo: its bitmap is in the cache.
	makeThumbnailOnDisk(t, a, a.lib.Get(next), 288)

	a.mode = 2
	a.current = a.lib.Get(next).Path
	view := ui.NewTester(a.view, 1100, 720)
	if !view.HasText("second.png") {
		t.Fatalf("the view did not draw: %q", view.Texts())
	}
	if view.HasText("载入中…") {
		t.Fatal("stepping to a photo showed a blank area although its thumbnail was cached")
	}
}

// TestPreviewWaitsForTheSharpOne checks the order: the size asked for first,
// then a smaller preview, then whatever the photo showed last.
func TestPreviewWaitsForTheSharpOne(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "photo.png", 60, 40, 30)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	p := a.lib.Get(path)

	// Nothing is known yet: nothing to show, and a request is made.
	if _, ok := a.previewBitmap(p, previewMedium); ok {
		t.Fatal("a photo the cache never saw already has a picture")
	}
	// The grid's thumbnail arrives: it stands in for the large preview.
	makeThumbnailOnDisk(t, a, p, 288)
	bmp, ok := a.previewBitmap(p, previewMedium)
	if !ok || bmp == nil {
		t.Fatal("the cached thumbnail was not used while the large preview is made")
	}
	if w, _ := bmp.Size(); w > 288 {
		t.Fatalf("the stand-in is %d wide, want the thumbnail", w)
	}
}

// TestNeighborsArePrefetched checks that the photos either side are asked for,
// so stepping to one shows it without waiting.
func TestNeighborsArePrefetched(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	writePNG(t, dir, "1.png", 40, 30, 10)
	writePNG(t, dir, "2.png", 40, 30, 60)
	writePNG(t, dir, "3.png", 40, 30, 120)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()

	mid := a.photos[1]
	if got := a.neighbor(mid, -1); got == nil || got.Path != a.photos[0].Path {
		t.Fatalf("the photo before the middle is %v", got)
	}
	if got := a.neighbor(mid, 1); got == nil || got.Path != a.photos[2].Path {
		t.Fatalf("the photo after the middle is %v", got)
	}
	if got := a.neighbor(a.photos[0], -1); got != nil {
		t.Fatal("the first photo has one before it")
	}
	if got := a.neighbor(a.photos[len(a.photos)-1], 1); got != nil {
		t.Fatal("the last photo has one after it")
	}
}

// TestToolbarCarriesThumbSizeAndSort is the second request: the two controls
// that shape the grid sit beside the folders, on the right.
func TestToolbarCarriesThumbSizeAndSort(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	writePNG(t, dir, "one.png", 40, 30, 10)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()

	view := ui.NewTester(a.view, 1180, 760)
	for _, want := range []string{"小", "中", "大", "按日期（新→旧）", "设置"} {
		if !view.HasText(want) {
			t.Fatalf("the toolbar lacks %q: %q", want, view.Texts())
		}
	}

	// A narrow window keeps them too, instead of pushing them off the bar.
	// A grid belongs to one window at a time, so this gets its own app.
	b := testApp(t)
	b.lib.AddFolder(dir)
	b.scanFolder(dir, nil)
	b.rebuild()
	narrow := ui.NewTester(b.view, 780, 560)
	narrow.Frame()
	for _, want := range []string{"大", "按日期（新→旧）", "设置"} {
		if !narrow.HasText(want) {
			t.Fatalf("a narrow toolbar lacks %q: %q", want, narrow.Texts())
		}
	}

	// The size buttons change the preference from there.
	if err := view.Click("大"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if a.set.ThumbSize != 2 {
		t.Fatalf("thumbnail size %d after 大, want 2", a.set.ThumbSize)
	}

	// And the order, through the popup.
	if err := view.Click("按日期（新→旧）"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if err := view.Click("按名称"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if a.set.Sort != "name" {
		t.Fatalf("sort %q after choosing 按名称, want name", a.set.Sort)
	}

	// The settings no longer repeat them.
	a.settingsOpen = true
	settings := ui.NewTester(a.view, 1180, 760)
	if settings.HasText("缩略图尺寸") {
		t.Fatal("the settings still offer the thumbnail size")
	}
	if !settings.HasText("扫描时跳过内容重复的照片") {
		t.Fatalf("the settings lost their own options: %q", settings.Texts())
	}
}
