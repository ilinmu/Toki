package main

import (
	"image"
	"image/color"
	"image/gif"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/tiff"
)

// writeGIF writes an animated GIF of transparent frames: a red ground on the
// first, a blue square with a clear corner on the second.
func writeGIF(t *testing.T, dir, name string) string {
	t.Helper()
	pal := color.Palette{
		color.RGBA{}, // transparent
		color.RGBA{255, 0, 0, 255},
		color.RGBA{0, 0, 255, 255},
	}
	red := image.NewPaletted(image.Rect(0, 0, 10, 10), pal)
	for i := range red.Pix {
		red.Pix[i] = 1
	}
	blue := image.NewPaletted(image.Rect(0, 0, 10, 10), pal)
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			blue.SetColorIndex(x, y, 2)
		}
	}
	g := &gif.GIF{
		Image: []*image.Paletted{red, blue},
		Delay: []int{10, 20},
		Config: image.Config{
			ColorModel: pal,
			Width:      10,
			Height:     10,
		},
	}
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := gif.EncodeAll(f, g); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestTheFramesOfAGIFAreComposited checks what an animated picture is made
// of: each frame on the one before it, and its delay in hundredths.
func TestTheFramesOfAGIFAreComposited(t *testing.T) {
	dir := t.TempDir()
	path := writeGIF(t, dir, "move.gif")
	frames, delays, err := readGIFFrames(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 || len(delays) != 2 {
		t.Fatalf("read %d frames with %d delays, want 2 and 2", len(frames), len(delays))
	}
	if delays[0] != 100*time.Millisecond || delays[1] != 200*time.Millisecond {
		t.Fatalf("delays %v, want 100ms and 200ms", delays)
	}
	// The second frame keeps the red ground of the first, puts a blue square
	// on it, and its clear corner is left alone rather than made black.
	if got := frames[1].RGBAAt(1, 1); got != (color.RGBA{0, 0, 255, 255}) {
		t.Fatalf("the blue square is %v", got)
	}
	if got := frames[1].RGBAAt(8, 8); got != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("the red ground is %v", got)
	}
	// The rail plays the frames in turn and comes round again.
	if i, _ := frameAt(0, delays); i != 0 {
		t.Fatalf("at the start frame %d, want 0", i)
	}
	if i, _ := frameAt(150*time.Millisecond, delays); i != 1 {
		t.Fatalf("150ms in frame %d, want 1", i)
	}
	if i, _ := frameAt(300*time.Millisecond, delays); i != 0 {
		t.Fatalf("300ms in frame %d, want the first again", i)
	}
}

// TestATransparentPictureGoesOnWhite is the background of a clear GIF: a JPEG
// has no transparency, and the clear pixels used to encode as black.
func TestATransparentPictureGoesOnWhite(t *testing.T) {
	pal := color.Palette{color.RGBA{}, color.RGBA{255, 0, 0, 255}}
	img := image.NewPaletted(image.Rect(0, 0, 4, 4), pal)
	img.SetColorIndex(0, 0, 1)

	out := onWhite(img)
	if got := out.At(1, 1); got != (color.RGBA{255, 255, 255, 255}) {
		t.Fatalf("the clear pixel is %v, want white", got)
	}
	if got := out.At(0, 0); got != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("the red pixel is %v", got)
	}
}

// TestTIFFPhotosAreTakenIn is a format the library reads besides the common
// ones: a TIFF is scanned, sized and opened like a JPEG.
func TestTIFFPhotosAreTakenIn(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 24, 16))
	for i := range img.Pix {
		img.Pix[i] = 90
	}
	path := filepath.Join(dir, "scan.tif")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := tiff.Encode(f, img, nil); err != nil {
		t.Fatal(err)
	}
	f.Close()

	a.lib.AddFolder(dir)
	if st := a.scanFolder(dir, nil); st.Added != 1 {
		t.Fatalf("scan added %d photos, want the TIFF: %+v", st.Added, st)
	}
	p := a.lib.Get(path)
	if p == nil {
		t.Fatal("the TIFF is not in the library")
	}
	if p.Width != 24 || p.Height != 16 {
		t.Fatalf("the TIFF is %d×%d, want 24×16", p.Width, p.Height)
	}
	if _, err := decodeImage(path); err != nil {
		t.Fatalf("the TIFF does not open: %v", err)
	}
}

// TestATagRemovedLeavesItsTagView is the tag panel following the inspector:
// taking a tag off the photo shown takes the photo out of that tag's view.
func TestATagRemovedLeavesItsTagView(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "one.png", 40, 30, 10)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.lib.LearnTag("咖啡")
	p := a.lib.Get(path)
	p.Tags = []string{"咖啡", "旅行"}
	a.save()

	a.navigate("tag:咖啡")
	a.rebuild()
	if len(a.photos) != 1 {
		t.Fatalf("the tag view shows %d photos, want 1", len(a.photos))
	}

	a.removeTag(p, 0)
	a.rebuild()
	if len(a.photos) != 0 {
		t.Fatalf("the photo is still in the tag view after losing the tag: %d", len(a.photos))
	}
	if !p.HasTag("旅行") {
		t.Fatal("removing one tag took the others away")
	}

	// Putting it back brings the photo back, without a reload.
	p.Tags = append(p.Tags, "咖啡")
	a.viewKey = ""
	a.rebuild()
	if len(a.photos) != 1 {
		t.Fatalf("the tag view shows %d photos after the tag came back, want 1", len(a.photos))
	}
}

// TestFavoritingBurstsConfetti is the celebration of a favorite: pieces from
// the heart that was pressed, and none once they have fallen.
func TestFavoritingBurstsConfetti(t *testing.T) {
	a, view, _, second := flowApp(t)

	rect, ok := view.Find("two.png")
	if !ok {
		t.Fatalf("no card for the second photo: %q", view.Texts())
	}
	view.Move(rect.X+rect.W/2, rect.Y+rect.H/2)
	view.Frame()
	heart, ok := view.Access("收藏这张照片")
	if !ok {
		t.Fatalf("the heart does not show on the card under the pointer: %q", view.Texts())
	}
	view.ClickAt(heart.Bounds.X+heart.Bounds.W/2, heart.Bounds.Y+heart.Bounds.H/2)
	view.Frame()

	if p := a.lib.Get(second); p == nil || !p.Favorite {
		t.Fatal("the click did not favorite the photo")
	}
	if a.fetti == nil {
		t.Fatal("favoriting raised no confetti")
	}
	if len(a.fetti.pieces) == 0 {
		t.Fatal("the burst has no pieces")
	}
	// It plays on the card, not over the window.
	area := a.fetti.area
	if area.W > 320 || area.H > 340 {
		t.Fatalf("the burst covers %v×%v, more than the card", area.W, area.H)
	}
	if !heartIn(area, heart.Bounds) {
		t.Fatalf("the burst plays in %+v, away from the heart %+v", area, heart.Bounds)
	}
	// It starts where the heart was pressed, not in the middle of the card.
	x, y := area.X+a.fetti.x, area.Y+a.fetti.y
	if x < heart.Bounds.X-8 || x > heart.Bounds.X+heart.Bounds.W+8 ||
		y < heart.Bounds.Y-8 || y > heart.Bounds.Y+heart.Bounds.H+8 {
		t.Fatalf("the burst starts at (%v, %v), away from the heart %+v", x, y, heart.Bounds)
	}

	// Unfavoriting does not celebrate, and the pieces fall away in time.
	view.ClickAt(heart.Bounds.X+heart.Bounds.W/2, heart.Bounds.Y+heart.Bounds.H/2)
	view.Frame()
	view.Advance(2 * time.Second)
	view.Frame()
	if a.fetti != nil {
		t.Fatal("the confetti did not end")
	}
}

// TestTheConfettiStaysInItsArea is the layer's box: the pieces paint over the
// photos while the burst runs, and nothing outside the area it was given.
func TestTheConfettiStaysInItsArea(t *testing.T) {
	a, view, _, _ := flowApp(t)

	rect, ok := view.Find("one.png")
	if !ok {
		t.Fatalf("no card for the first photo: %q", view.Texts())
	}
	area := ui.Rect{X: rect.X - 6, Y: rect.Y - 6, W: rect.W + 12, H: rect.H + 12}
	before := view.Image()
	a.burstAt(view.Now(), rand.New(rand.NewSource(3)), area, 24, area.H-24)
	view.Advance(120 * time.Millisecond)
	view.Frame()
	after := view.Image()

	changed, outside := 0, 0
	for y := 0; y < before.Bounds().Dy(); y++ {
		for x := 0; x < before.Bounds().Dx(); x++ {
			if before.RGBAAt(x, y) == after.RGBAAt(x, y) {
				continue
			}
			changed++
			if float32(x) < area.X || float32(x) > area.X+area.W ||
				float32(y) < area.Y || float32(y) > area.Y+area.H {
				outside++
			}
		}
	}
	if changed == 0 {
		t.Fatal("the confetti painted nothing")
	}
	if outside != 0 {
		t.Fatalf("%d pixels of confetti fell outside their area %+v", outside, area)
	}
}

// TestTheViewerPlaysAGIF is the wiring of the frames: the viewer shows the
// first frame of an animated GIF and the next one when its time comes.
func TestTheViewerPlaysAGIF(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writeGIF(t, dir, "move.gif")
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()
	a.openLightbox(path)

	view := ui.NewTester(a.view, 1180, 760)
	view.Frame()
	view.Frame()
	if len(a.anim.frames) != 2 {
		t.Fatalf("the viewer holds %d frames, want 2", len(a.anim.frames))
	}
	first := view.Image()

	view.Advance(120 * time.Millisecond)
	view.Frame()
	second := view.Image()
	changed := 0
	for y := 0; y < first.Bounds().Dy(); y++ {
		for x := 0; x < first.Bounds().Dx(); x++ {
			if first.RGBAAt(x, y) != second.RGBAAt(x, y) {
				changed++
			}
		}
	}
	if changed == 0 {
		t.Fatal("the GIF did not move on to its second frame")
	}
}

// TestTheViewerConfettiIsASmallField is the full screen picture: the pieces
// play in a little square around the heart, not across the stage.
func TestTheViewerConfettiIsASmallField(t *testing.T) {
	a, _, first, _ := flowApp(t)

	a.openLightbox(first)
	full := ui.NewTester(a.view, 1180, 760)
	full.Frame()
	heart, ok := full.Access("收藏 (F)")
	if !ok {
		t.Fatalf("the viewer has no heart: %q", full.Texts())
	}
	full.ClickAt(heart.Bounds.X+heart.Bounds.W/2, heart.Bounds.Y+heart.Bounds.H/2)
	full.Frame()

	if a.fetti == nil {
		t.Fatal("favoriting in the viewer raised no confetti")
	}
	if a.fetti.area.W > confettiSquare || a.fetti.area.H > confettiSquare {
		t.Fatalf("the viewer burst covers %v×%v, more than %v", a.fetti.area.W, a.fetti.area.H, confettiSquare)
	}
	if a.fetti.area.W < 100 || a.fetti.area.H < 100 {
		t.Fatalf("the viewer burst is only %v×%v", a.fetti.area.W, a.fetti.area.H)
	}
}

// heartIn reports whether the heart lies in the area of a burst.
func heartIn(area, heart ui.Rect) bool {
	return heart.X+heart.W >= area.X && heart.X <= area.X+area.W &&
		heart.Y+heart.H >= area.Y && heart.Y <= area.Y+area.H
}
