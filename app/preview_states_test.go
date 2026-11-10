package main

import (
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestPreviewStates renders the states the manual preview does not reach: a
// chosen photo, the command palette and the new album dialog. Like TestPreview
// it is a manual tool, and skips unless both paths are given.
//
//	TOKI_PREVIEW_DATA="$PWD/.preview" TOKI_PREVIEW_OUT=../docs/screenshots \
//	  go test -run TestPreviewStates -v
func TestPreviewStates(t *testing.T) {
	data := previewEnv("TOKI_PREVIEW_DATA", "LIBRA_PREVIEW_DATA")
	out := previewEnv("TOKI_PREVIEW_OUT", "LIBRA_PREVIEW_OUT")
	if data == "" || out == "" {
		t.Skip("set TOKI_PREVIEW_DATA and TOKI_PREVIEW_OUT to render the states")
	}
	a := newAppAt(data)
	a.rebuild()
	if len(a.photos) < 3 {
		t.Fatalf("few photos in %s", data)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	save := func(name string, view *ui.Tester) {
		shot := view.Image()
		f, err := os.Create(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, shot); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", filepath.Join(out, name))
	}

	// One photo chosen: the card's mark, the inspector and the batch bar.
	a.sel.Clear()
	a.sel.Add(a.photos[0].Path)
	a.current = a.photos[0].Path
	chosen := ui.NewTester(a.view, 1180, 760)
	chosen.Frame()
	save("ui-selected.png", chosen)

	// The command palette over the library.
	a.commandOpen = true
	a.commandQuery = "猫"
	command := ui.NewTester(a.view, 1180, 760)
	command.Frame()
	save("ui-command.png", command)
	a.commandOpen = false

	// The new album dialog.
	a.albumName = "海边的周末"
	a.newAlbumOpen = true
	album := ui.NewTester(a.view, 1180, 760)
	album.Frame()
	save("ui-new-album.png", album)
	a.newAlbumOpen = false

	// Many chosen: the inspector's batch panel.
	a.sel.Clear()
	for i := 0; i < 4 && i < len(a.photos); i++ {
		a.sel.Add(a.photos[i].Path)
	}
	many := ui.NewTester(a.view, 1180, 760)
	many.Frame()
	save("ui-multi.png", many)

	// The masonry with its date rail, and the rail after scrolling halfway:
	// the handle and the dates have to follow the flow.
	a.sel.Clear()
	a.mode, a.layout = 0, 1
	a.libScroll.Y = 0
	masonry := ui.NewTester(a.view, 1180, 760)
	masonry.Frame()
	masonry.Frame()
	save("ui-masonry.png", masonry)

	a.libScroll.Y = 900
	scrolled := ui.NewTester(a.view, 1180, 760)
	scrolled.Frame()
	scrolled.Frame()
	save("ui-masonry-scrolled.png", scrolled)

	// The live search the title bar's field does.
	a.layout = 0
	a.query = "猫"
	a.viewKey = ""
	search := ui.NewTester(a.view, 1180, 760)
	search.Frame()
	save("ui-search.png", search)
	a.query = ""

	// The confetti of a favorite on a card, from its heart and inside the
	// card, a moment after it was set off.
	a.mode, a.layout = 0, 1
	a.libScroll.Y = 0
	confetti := ui.NewTester(a.view, 1180, 760)
	confetti.Frame()
	if card, ok := confetti.Access("城市夜景.jpg"); ok {
		area := card.Bounds
		a.burstAt(confetti.Now(), rand.New(rand.NewSource(7)),
			area, area.W-16, 16)
		confetti.Advance(380 * time.Millisecond)
		confetti.Frame()
		save("ui-confetti.png", confetti)
		a.fetti = nil
	}
}
