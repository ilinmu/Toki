package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestAPhotoFromFinderOpensInTheViewer is the double click in the file
// manager: an image outside the library opens in the app's viewer without
// being added to the library.
func TestAPhotoFromFinderOpensInTheViewer(t *testing.T) {
	a := testApp(t)
	path := writePNG(t, t.TempDir(), "from-finder.png", 60, 40, 90)

	a.showOpenedFile(path)
	if !a.lightbox {
		t.Fatal("opening an image did not show the viewer")
	}
	if a.current != path {
		t.Fatalf("the viewer shows %q, want %q", a.current, path)
	}
	if a.external == nil || a.external.Path != path {
		t.Fatal("the opened file is not the one-off photo")
	}
	if a.lib.Get(path) != nil {
		t.Fatal("opening a file silently added it to the library")
	}

	view := ui.NewTester(a.view, 1000, 700)
	view.Frame()
	if !view.HasText("from-finder.png") {
		t.Fatalf("the viewer did not draw the file: %q", view.Texts())
	}
	if got := a.lightboxCounter(); got != "1 / 1" {
		t.Fatalf("the counter says %q, want 1 / 1", got)
	}

	a.closeLightbox()
	if a.external != nil || a.current != "" {
		t.Fatal("closing the viewer left the one-off photo behind")
	}
}

// TestAPhotoAlreadyInTheLibraryOpensWithoutAOneOff keeps the filmstrip and
// the library navigation working for a photo the library already knows.
func TestAPhotoAlreadyInTheLibraryOpensWithoutAOneOff(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "in-library.png", 60, 40, 20)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()

	a.showOpenedFile(path)
	if !a.lightbox || a.current != path {
		t.Fatal("the file in the library did not open")
	}
	if a.external != nil {
		t.Fatal("a library photo was treated as a one-off file")
	}
	if _, ok := a.photoIndex[path]; !ok {
		t.Fatal("the opened photo is not in the shown photos")
	}
}

// TestTheAppDeclaresImageAssociations is the build metadata that lets the
// file manager send an image to Toki.
func TestTheAppDeclaresImageAssociations(t *testing.T) {
	b, err := os.ReadFile("mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		FileAssociations []struct {
			Ext  []string `json:"ext"`
			Role string   `json:"role"`
		} `json:"fileAssociations"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"jpg": false, "jpeg": false, "png": false,
		"gif": false, "bmp": false, "webp": false,
		"tif": false, "tiff": false,
	}
	for _, assoc := range cfg.FileAssociations {
		if assoc.Role != "Viewer" {
			t.Errorf("association role is %q, want Viewer", assoc.Role)
		}
		for _, ext := range assoc.Ext {
			if _, ok := want[ext]; ok {
				want[ext] = true
			}
		}
	}
	for ext, seen := range want {
		if !seen {
			t.Errorf("mygo.json does not associate .%s", ext)
		}
	}
}
