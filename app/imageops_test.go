package main

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// solid makes a picture of one color.
func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestRotateQuarterTurns(t *testing.T) {
	src := solid(40, 20, color.RGBA{10, 20, 30, 255})
	for _, tc := range []struct {
		turn int
		w, h int
	}{
		{90, 20, 40},
		{180, 40, 20},
		{270, 20, 40},
	} {
		out := renderEdit(src, EditState{Rotate: tc.turn})
		if b := out.Bounds(); b.Dx() != tc.w || b.Dy() != tc.h {
			t.Errorf("rotate %d gives %dx%d, want %dx%d", tc.turn, b.Dx(), b.Dy(), tc.w, tc.h)
		}
	}
}

func TestFlipMirrorsPixels(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	src.Set(3, 0, color.RGBA{0, 0, 255, 255})
	out := toRGBA(renderEdit(src, EditState{FlipH: true}))
	if r, _, _, _ := out.At(3, 0).RGBA(); r>>8 != 255 {
		t.Fatalf("horizontal flip did not move the red pixel: got %v", out.At(3, 0))
	}
	out = toRGBA(renderEdit(src, EditState{FlipV: true}))
	if r, _, _, _ := out.At(0, 1).RGBA(); r>>8 != 255 {
		t.Fatalf("vertical flip did not move the red pixel: got %v", out.At(0, 1))
	}
}

func TestCropTakesTheAskedPart(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 80))
	src.Set(50, 40, color.RGBA{9, 9, 9, 255})
	out := renderEdit(src, EditState{HasCrop: true, CropX: 0.25, CropY: 0.25, CropW: 0.5, CropH: 0.5})
	b := out.Bounds()
	if b.Dx() != 50 || b.Dy() != 40 {
		t.Fatalf("crop gives %dx%d, want 50x40", b.Dx(), b.Dy())
	}
	if r, _, _, _ := out.At(25, 20).RGBA(); r>>8 != 9 {
		t.Fatalf("crop lost the middle pixel: %v", out.At(25, 20))
	}
}

func TestCropShapes(t *testing.T) {
	x, y, w, h := cropForShape(4000, 3000, aspectRatio{"16:9", 16, 9})
	if w != 1 || y <= 0 {
		t.Fatalf("16:9 in a 4:3 picture: %v %v %v %v", x, y, w, h)
	}
	if got := w * 4000 / (h * 3000); got < 1.75 || got > 1.81 {
		t.Fatalf("16:9 crop has shape %.2f", got)
	}
	x, y, w, h = cropForShape(4000, 3000, aspectRatio{"1:1 方形", 1, 1})
	if x <= 0 || w != 0.75 || h != 1 {
		t.Fatalf("square crop: %v %v %v %v", x, y, w, h)
	}
	x, y, w, h = cropForShape(4000, 3000, aspectRatio{"原始比例", 0, 0})
	if x != 0 || y != 0 || w != 1 || h != 1 {
		t.Fatalf("original crop: %v %v %v %v", x, y, w, h)
	}
}

func TestStraightenKeepsThePicture(t *testing.T) {
	src := solid(100, 50, color.RGBA{120, 130, 140, 255})
	out := renderEdit(src, EditState{Angle: 10})
	b := out.Bounds()
	if b.Dx() <= 100 || b.Dy() <= 50 {
		t.Fatalf("straighten did not enlarge the frame: %dx%d", b.Dx(), b.Dy())
	}
}

func TestResizeAndFit(t *testing.T) {
	src := solid(1000, 500, color.RGBA{1, 2, 3, 255})
	w, h := fitSize(1000, 500, 400, 400)
	if w != 400 || h != 200 {
		t.Fatalf("fitSize gives %dx%d, want 400x200", w, h)
	}
	w, h = fitSize(100, 100, 400, 400)
	if w != 100 || h != 100 {
		t.Fatalf("fitSize scaled up to %dx%d", w, h)
	}
	small := resize(src, 100, 50)
	if b := small.Bounds(); b.Dx() != 100 || b.Dy() != 50 {
		t.Fatalf("resize gives %dx%d", b.Dx(), b.Dy())
	}
}

func TestOrientationTurnsUpright(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 40, 20))
	src.Set(0, 0, color.RGBA{255, 255, 255, 255})
	// Orientation 6 is a picture shot with the camera turned: it shows
	// 20 wide and 40 high.
	out := applyOrientation(src, 6)
	if b := out.Bounds(); b.Dx() != 20 || b.Dy() != 40 {
		t.Fatalf("orientation 6 gives %dx%d, want 20x40", b.Dx(), b.Dy())
	}
}

func TestExportWritesSizedCopies(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	writePNG(t, dir, "export-me.png", 800, 400, 60)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	p := a.lib.Get(filepath.Join(dir, "export-me.png"))
	p.Stars = 5
	p.Tags = []string{"导出"}

	out := t.TempDir()
	form := exportForm{Dir: out, Folder: "round", Size: 0, Numbers: true, Watermark: "Libra"}
	res := a.exportPhotos([]*Photo{p}, form, 88)
	if res.Done != 1 || res.Failed != 0 {
		t.Fatalf("export: %+v", res)
	}
	exported := filepath.Join(out, "round", "001-export-me.jpg")
	f, err := os.Open(exported)
	if err != nil {
		t.Fatalf("exported file missing: %v", err)
	}
	defer f.Close()
	cfg, _, err := imageDecodeConfig(f, ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 800 || cfg.Height != 400 {
		t.Fatalf("exported %dx%d, want the original 800x400", cfg.Width, cfg.Height)
	}

	// A size preset scales the copy down.
	res = a.exportPhotos([]*Photo{p}, exportForm{Dir: out, Folder: "small", Size: 1}, 82)
	if res.Done != 1 {
		t.Fatalf("second export: %+v", res)
	}
	f2, err := os.Open(filepath.Join(out, "small", "export-me.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	cfg2, _, err := imageDecodeConfig(f2, ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Width != 320 || cfg2.Height != 160 {
		t.Fatalf("320 preset gives %dx%d", cfg2.Width, cfg2.Height)
	}
}

func TestExportAppliesEdits(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	writePNG(t, dir, "turn.png", 400, 200, 70)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	p := a.lib.Get(filepath.Join(dir, "turn.png"))
	p.Edit = EditState{Rotate: 90}

	out := t.TempDir()
	res := a.exportPhotos([]*Photo{p}, exportForm{Dir: out, Folder: "turned", Size: 0}, 80)
	if res.Done != 1 {
		t.Fatalf("export: %+v", res)
	}
	f, err := os.Open(filepath.Join(out, "turned", "turn.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := imageDecodeConfig(f, ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 200 || cfg.Height != 400 {
		t.Fatalf("rotated export is %dx%d, want 200x400", cfg.Width, cfg.Height)
	}
}

func TestUndoRestoresEdits(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	writePNG(t, dir, "undo.png", 100, 50, 20)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	p := a.lib.Get(filepath.Join(dir, "undo.png"))

	a.draft = p.Edit.Clone()
	a.draft.Rotate = 90
	a.applyEdit(p)
	if p.Edit.Rotate != 90 || len(p.Undo) != 1 {
		t.Fatalf("apply: rotate %d undo %d", p.Edit.Rotate, len(p.Undo))
	}
	if !a.undoEdit(p) || p.Edit.Rotate != 0 {
		t.Fatalf("undo: rotate %d", p.Edit.Rotate)
	}
	if !a.redoEdit(p) || p.Edit.Rotate != 90 {
		t.Fatalf("redo: rotate %d", p.Edit.Rotate)
	}
	if a.undoEdit(p); a.undoEdit(p) {
		t.Fatal("undo went past the first edit")
	}
}
