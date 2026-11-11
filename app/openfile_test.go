package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// TestTheFolderOfAFileOpenedFromFinderIsTheFilmstrip is the third request:
// a photo opened in the file manager is looked at with the other images of
// its folder, which the viewer steps through. Nothing joins the library.
func TestTheFolderOfAFileOpenedFromFinderIsTheFilmstrip(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	first := writePNG(t, dir, "a.png", 60, 40, 10)
	writePNG(t, dir, "b.png", 60, 40, 80)
	writePNG(t, dir, "c.png", 60, 40, 150)
	// A file that is not a picture is not in the filmstrip.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	a.showOpenedFile(first)
	if got := a.lightboxCounter(); got != "1 / 3" {
		t.Fatalf("the counter says %q, want 1 / 3", got)
	}
	if got := len(a.shownPhotos()); got != 3 {
		t.Fatalf("the viewer walks %d photos, want the 3 images of the folder", got)
	}
	view := ui.NewTester(a.view, 1000, 700)
	view.Frame()
	for _, name := range []string{"a.png", "b.png", "c.png"} {
		if !view.HasText(name) {
			t.Fatalf("the filmstrip lacks %q: %q", name, view.Texts())
		}
	}

	// The arrow keys walk the folder, and the photo they reach is read too.
	a.step(1)
	if filepath.Base(a.current) != "b.png" {
		t.Fatalf("stepping once shows %q, want b.png", a.current)
	}
	if p := a.shownPhoto(); p == nil || p.Width != 60 || p.Height != 40 {
		t.Fatalf("the photo stepped to was not read: %+v", p)
	}
	a.step(-2)
	if filepath.Base(a.current) != "c.png" {
		t.Fatalf("stepping back past the start shows %q, want c.png", a.current)
	}
	if len(a.lib.Photos) != 0 {
		t.Fatalf("browsing a folder added %d photos to the library", len(a.lib.Photos))
	}

	// Closing the viewer forgets the folder it was looking at.
	a.closeLightbox()
	if a.externalFor != "" || a.external != nil || len(a.shownPhotos()) != 0 {
		t.Fatal("closing the viewer left the browsed folder behind")
	}
}

// TestFavoritingAPhotoFromFinderKeepsIt is the fourth request: a favorite is
// a photo the library keeps, so favoriting a file opened in the file manager
// adds that photo. Only that photo: the rest of its folder stays out.
func TestFavoritingAPhotoFromFinderKeepsIt(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "keep-me.png", 60, 40, 90)
	writePNG(t, dir, "leave-me.png", 60, 40, 20)

	a.showOpenedFile(path)
	p := a.shownPhoto()
	if p == nil {
		t.Fatal("the opened file is not shown")
	}
	a.setFavorite(p, true)
	a.save()

	kept := a.lib.Get(path)
	if kept == nil {
		t.Fatal("favoriting a file from the file manager did not add it to the library")
	}
	if !kept.Favorite {
		t.Fatal("the photo joined the library without its favorite mark")
	}
	if len(a.lib.Photos) != 1 {
		t.Fatalf("the library holds %d photos, want the favorited one only", len(a.lib.Photos))
	}
	if len(a.lib.FolderList()) != 0 {
		t.Fatal("favoriting one photo imported its whole folder")
	}

	// It is a favorite of the library from now on: the 收藏 view shows it.
	a.navigate("favorites")
	a.rebuild()
	if len(a.photos) != 1 || a.photos[0].Path != path {
		t.Fatalf("the favorites view shows %d photos, want the one favorited", len(a.photos))
	}

	// Its stars are the library's stars, and they are kept.
	a.showOpenedFile(path)
	a.setStars(a.shownPhoto(), 5)
	if got := a.lib.Get(path).Stars; got != 5 {
		t.Fatalf("stars are %d, want 5", got)
	}

	// Unfavoriting keeps it: leaving the library is what 「移出库」 is for.
	a.setFavorite(a.lib.Get(path), false)
	a.save()
	if a.lib.Get(path) == nil {
		t.Fatal("unfavoriting a photo emptied it out of the library")
	}
}

// TestTheAppDeclaresImageAssociations is the build metadata that lets the
// file manager send an image to Toki: the extensions, and the system types
// (UTIs) macOS resolves them to. Without the types, an "always open with"
// choice answers for one file and leaves the other files of the same kind to
// Preview.
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
		MacOS struct {
			InfoPlist struct {
				Types []struct {
					Name      string   `json:"CFBundleTypeName"`
					Role      string   `json:"CFBundleTypeRole"`
					Rank      string   `json:"LSHandlerRank"`
					Content   []string `json:"LSItemContentTypes"`
					Extension []string `json:"CFBundleTypeExtensions"`
				} `json:"CFBundleDocumentTypes"`
			} `json:"infoPlist"`
		} `json:"macos"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	// The UTIs macOS knows for the formats the app reads.
	want := map[string]string{
		"jpg": "public.jpeg", "jpeg": "public.jpeg",
		"png":  "public.png",
		"gif":  "com.compuserve.gif",
		"bmp":  "com.microsoft.bmp",
		"webp": "org.webmproject.webp",
		"tif":  "public.tiff", "tiff": "public.tiff",
	}
	seen := map[string]bool{}
	for _, assoc := range cfg.FileAssociations {
		if assoc.Role != "Viewer" {
			t.Errorf("association role is %q, want Viewer", assoc.Role)
		}
		for _, ext := range assoc.Ext {
			seen[ext] = true
		}
	}
	for ext := range want {
		if !seen[ext] {
			t.Errorf("mygo.json does not associate .%s", ext)
		}
	}

	// The document types the bundle declares, which is what LaunchServices
	// reads to know Toki opens a JPEG at all.
	types := map[string]string{} // extension -> UTI
	for _, doc := range cfg.MacOS.InfoPlist.Types {
		if doc.Role != "Viewer" {
			t.Errorf("document type %q has role %q, want Viewer", doc.Name, doc.Role)
		}
		if doc.Rank != "Alternate" {
			t.Errorf("document type %q has rank %q, want Alternate (a viewer, not the hander of the system's own types)", doc.Name, doc.Rank)
		}
		if len(doc.Content) != 1 {
			t.Errorf("document type %q declares %d content types, want 1", doc.Name, len(doc.Content))
			continue
		}
		for _, ext := range doc.Extension {
			types[ext] = doc.Content[0]
		}
	}
	for ext, uti := range want {
		if got, ok := types[ext]; !ok {
			t.Errorf("the bundle declares no document type for .%s", ext)
		} else if got != uti {
			t.Errorf(".%s declares %q, want %q", ext, got, uti)
		}
	}
}
