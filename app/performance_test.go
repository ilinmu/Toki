package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// manyPhotosApp makes a library of n photos without touching the disk: the
// views only need their metadata.
func manyPhotosApp(t *testing.T, n int) *app {
	t.Helper()
	a := testApp(t)
	base := time.Now().Unix()
	for i := 0; i < n; i++ {
		a.lib.Upsert(&Photo{
			Path:      fmt.Sprintf("/tmp/toki-%05d.jpg", i),
			Name:      fmt.Sprintf("photo-%05d.jpg", i),
			Root:      "/tmp",
			TakenUnix: base - int64(i*3600),
			Width:     1200,
			Height:    800,
		})
	}
	a.rebuild()
	return a
}

func countNamed(view *ui.Tester, prefix string) int {
	n := 0
	for _, s := range view.Texts() {
		if strings.HasPrefix(s, prefix) {
			n++
		}
	}
	return n
}

// TestALargeGridBuildsOnlyVisibleCards guards the virtualised flow: thousands
// of photos must not become thousands of elements in one frame.
func TestALargeGridBuildsOnlyVisibleCards(t *testing.T) {
	a := manyPhotosApp(t, 5000)
	view := ui.NewTester(a.view, 1180, 760)
	view.Frame()
	view.Frame()
	if got := countNamed(view, "photo-"); got > 80 {
		t.Fatalf("a grid of 5000 photos built %d card labels, want only the visible ones", got)
	}
	// Scrolling to the end must stay just as small.
	a.libScroll.Y = a.libScroll.MaxY
	view.Frame()
	if got := countNamed(view, "photo-"); got > 80 {
		t.Fatalf("the end of the grid built %d card labels", got)
	}
}

// TestALargeTimelineBuildsOnlyVisibleRows is the same guarantee for the
// one-column timeline.
func TestALargeTimelineBuildsOnlyVisibleRows(t *testing.T) {
	a := manyPhotosApp(t, 5000)
	a.mode = 1
	view := ui.NewTester(a.view, 1180, 760)
	view.Frame()
	view.Frame()
	if got := countNamed(view, "photo-"); got > 100 {
		t.Fatalf("a timeline of 5000 photos built %d labels, want only the visible rows", got)
	}
}

// TestALargeFilmstripBuildsOnlyVisibleThumbs keeps the viewer's filmstrip
// from building one button per photo of the library.
func TestALargeFilmstripBuildsOnlyVisibleThumbs(t *testing.T) {
	a := manyPhotosApp(t, 5000)
	a.openLightbox(a.photos[0].Path)
	view := ui.NewTester(a.view, 1180, 760)
	view.Frame()
	if got := countNamed(view, "photo-"); got > 60 {
		t.Fatalf("the filmstrip built %d thumbnails, want only the visible ones", got)
	}
}

// TestTheSidebarIsIndexedOnce checks the cached library index: a redraw reads
// the same data until something changes.
func TestTheSidebarIsIndexedOnce(t *testing.T) {
	a := manyPhotosApp(t, 200)
	first := a.library()
	if again := a.library(); again != first {
		t.Fatal("the library index was rebuilt without a change")
	}
	a.save()
	if third := a.library(); third == first {
		t.Fatal("a saved change did not invalidate the library index")
	}
}

// TestTheThumbnailSizeEases checks the small / medium / large switch: the
// layout width moves toward the new size instead of jumping.
func TestTheThumbnailSizeEases(t *testing.T) {
	a := manyPhotosApp(t, 40)
	a.set.ThumbSize = 0
	view := ui.NewTester(a.view, 1180, 760)
	view.Frame()
	if a.thumbShown != a.thumbTarget() {
		t.Fatalf("the first layout is %v, want %v", a.thumbShown, a.thumbTarget())
	}
	a.thumbAnimLength = 5 * time.Millisecond
	a.set.ThumbSize = 2
	view.Frame()
	time.Sleep(20 * time.Millisecond)
	view.Frame()
	if a.thumbShown != a.thumbTarget() {
		t.Fatalf("the size settled at %v, want %v", a.thumbShown, a.thumbTarget())
	}
}
