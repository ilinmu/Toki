package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writePNG writes a small picture and returns its path.
func writePNG(t *testing.T, dir, name string, w, h int, shade uint8) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: shade, G: uint8(x % 256), B: uint8(y % 256), A: 255})
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
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

// testApp makes an app with its data in a temporary folder.
func testApp(t *testing.T) *app {
	t.Helper()
	dir := t.TempDir()
	a := newAppAt(dir)
	a.set.Dedupe = true
	return a
}

func TestScanAddsAndSkips(t *testing.T) {
	a := testApp(t)
	photos := t.TempDir()
	writePNG(t, photos, "one.png", 40, 30, 10)
	writePNG(t, photos, "two.png", 20, 10, 200)
	os.MkdirAll(filepath.Join(photos, ".hidden"), 0o755)
	writePNG(t, filepath.Join(photos, ".hidden"), "skip.png", 8, 8, 1)
	writePNG(t, filepath.Join(photos, "Originals"), "old.png", 8, 8, 2)

	a.lib.AddFolder(photos)
	st := a.scanFolder(photos, nil)
	if st.Added != 2 {
		t.Fatalf("added %d photos, want 2: %+v", st.Added, st)
	}
	if got := len(a.lib.Snapshot()); got != 2 {
		t.Fatalf("library has %d photos, want 2", got)
	}
	// A second scan finds nothing new and leaves the library alone.
	st = a.scanFolder(photos, nil)
	if st.Added != 0 || st.Skipped != 2 {
		t.Fatalf("second scan added %d skipped %d, want 0 and 2", st.Added, st.Skipped)
	}
	// A picture with the same contents under another name is a duplicate.
	writePNG(t, photos, "copy.png", 40, 30, 10)
	data, err := os.ReadFile(filepath.Join(photos, "one.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(photos, "copy.png"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	st = a.scanFolder(photos, nil)
	if st.Duplicates != 1 {
		t.Fatalf("duplicates %d, want 1: %+v", st.Duplicates, st)
	}
}

func TestScanRecordsSizes(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "wide.png", 64, 16, 90)
	a.scanFolder(dir, nil)
	p := a.lib.Get(path)
	if p == nil {
		t.Fatal("photo not in library")
	}
	if p.Width != 64 || p.Height != 16 {
		t.Errorf("size %dx%d, want 64x16", p.Width, p.Height)
	}
	if p.TakenUnix == 0 {
		t.Error("no date taken")
	}
}

func TestDisplayKeepsFiles(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "keep.png", 16, 16, 30)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.lib.Forget([]string{path})
	if a.lib.Get(path) != nil {
		t.Fatal("photo stayed in the library")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the file was touched: %v", err)
	}
}

func TestPruneForgetsMissingFiles(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "gone.png", 10, 10, 3)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	os.Remove(path)
	if n := a.prune(); n != 1 {
		t.Fatalf("pruned %d photos, want 1", n)
	}
	if a.lib.Get(path) != nil {
		t.Fatal("missing photo stayed")
	}
}

func TestStarsAndTags(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "star.png", 10, 10, 3)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	p := a.lib.Get(path)
	p.Stars = 4
	p.Tags = append(p.Tags, "家庭")
	a.lib.LearnTag("家庭")
	a.save()

	again := NewLibrary(a.lib.path)
	if err := again.Load(); err != nil {
		t.Fatal(err)
	}
	loaded := again.Get(path)
	if loaded == nil || loaded.Stars != 4 || !loaded.HasTag("家庭") {
		t.Fatalf("stars and tags did not survive: %+v", loaded)
	}
	if tags := again.AllTags(); len(tags) != 1 || tags[0] != "家庭" {
		t.Fatalf("tags %v, want one", tags)
	}
}

func TestAlbums(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "album.png", 10, 10, 3)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	al := a.lib.CreateAlbum("旅行")
	a.lib.AddToAlbum(al.ID, []*Photo{a.lib.Get(path)})
	if !a.lib.Get(path).InAlbum(al.ID) {
		t.Fatal("photo not in album")
	}
	a.lib.DeleteAlbum(al.ID)
	if a.lib.Get(path).InAlbum(al.ID) {
		t.Fatal("album stayed on the photo")
	}
	if len(a.lib.AlbumList()) != 0 {
		t.Fatal("album stayed in the library")
	}
}

func TestFiltersAndSearch(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	writePNG(t, dir, "beach-sunset.png", 10, 10, 3)
	writePNG(t, dir, "office.png", 10, 10, 4)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.lib.LearnTag("假期")

	if got := len(a.photos); got != 0 {
		t.Fatalf("photos built before rebuild: %d", got)
	}
	a.rebuild()
	if got := len(a.photos); got != 2 {
		t.Fatalf("%d photos, want 2", got)
	}
	a.query = "beach"
	a.viewKey = ""
	a.rebuild()
	if got := len(a.photos); got != 1 || a.photos[0].Name != "beach-sunset.png" {
		t.Fatalf("search found %d photos", len(a.photos))
	}
	a.query = ""
	a.minStars = 1
	a.viewKey = ""
	beach := a.lib.Get(filepath.Join(dir, "beach-sunset.png"))
	beach.Stars = 3
	beach.Tags = []string{"假期"}
	a.lib.Upsert(&Photo{Path: beach.Path, Size: beach.Size, ModUnix: beach.ModUnix})
	a.rebuild()
	if got := len(a.photos); got != 1 {
		t.Fatalf("star filter shows %d photos, want 1", len(a.photos))
	}
	a.nav = "tag:假期"
	a.viewKey = ""
	a.rebuild()
	if got := len(a.photos); got != 1 {
		t.Fatalf("tag filter shows %d photos, want 1", len(a.photos))
	}
	a.nav = "year:" + itoa(time.Now().Year())
	a.viewKey = ""
	a.rebuild()
	if got := len(a.photos); got != 1 {
		t.Fatalf("year filter shows %d photos, want 1", len(a.photos))
	}
}

func TestWatchingAddsPhotos(t *testing.T) {
	if testing.Short() {
		t.Skip("watches the file system")
	}
	a := testApp(t)
	dir := t.TempDir()
	a.lib.AddFolder(dir)
	a.watchFolder(dir)
	path := writePNG(t, dir, "late.png", 20, 20, 5)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if a.lib.Get(path) != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("the watch did not add a photo written into the folder")
}

// TestDeletingATagRemovesItEverywhere takes a tag off every photo and out of
// the learned vocabulary.
func TestDeletingATagRemovesItEverywhere(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	first := writePNG(t, dir, "one.png", 30, 20, 10)
	second := writePNG(t, dir, "two.png", 30, 20, 80)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	p1, p2 := a.lib.Get(first), a.lib.Get(second)
	p1.Tags = []string{"咖啡", "旅行"}
	p2.Tags = []string{"咖啡"}
	a.lib.LearnTag("咖啡")
	a.save()

	a.deleteTag("咖啡")
	if p1.HasTag("咖啡") || p2.HasTag("咖啡") {
		t.Fatal("the tag is still on a photo")
	}
	if p1.HasTag("旅行") != true {
		t.Fatal("deleting one tag removed another")
	}
	for _, tag := range a.lib.AllTags() {
		if tag == "咖啡" {
			t.Fatal("the deleted tag is still in the vocabulary")
		}
	}
	if p1.Tags[0] != "旅行" {
		t.Fatalf("the remaining tags are %v", p1.Tags)
	}
}

// TestRemovingAFolderKeepsEveryFile takes a watched folder out of the library
// without touching the files on disk.
func TestRemovingAFolderKeepsEveryFile(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "kept.png", 30, 20, 10)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	if len(a.lib.Snapshot()) != 1 {
		t.Fatal("the photo was not imported")
	}

	a.nav = "folder:" + dir
	a.removeFolder(dir)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("removing the folder deleted the file: %v", err)
	}
	if len(a.lib.Snapshot()) != 0 {
		t.Fatal("the photos are still in the library")
	}
	if len(a.lib.FolderList()) != 0 {
		t.Fatal("the folder is still watched")
	}
	if a.nav != "all" {
		t.Fatalf("the view stayed on the removed folder: %q", a.nav)
	}
}
