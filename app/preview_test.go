package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// previewEnv reads an environment variable, or the name it had before.
func previewEnv(name, oldName string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return os.Getenv(oldName)
}

// previewTheme is the appearance to render, when one is asked for. Left empty
// the preview uses the library's own setting, which is dark; TOKI_PREVIEW_THEME
// =light renders the same set in the light appearance, next to the dark one.
func previewTheme() string {
	switch t := os.Getenv("TOKI_PREVIEW_THEME"); t {
	case "dark", "light":
		return t
	}
	return ""
}

// previewName adds the appearance to a picture's name when one was asked for,
// so both sets can be written to the same folder.
func previewName(name, theme string) string {
	if theme == "" {
		return name
	}
	return strings.TrimSuffix(name, ".png") + "-" + theme + ".png"
}

// previewApp opens the demo library in the appearance the preview asked for.
func previewApp(data, theme string) *app {
	a := newAppAt(data)
	if theme != "" {
		a.set.Theme = theme
	}
	a.rebuild()
	return a
}

// TestPreview renders the window's views without a window and writes them as
// PNGs, to look at the layout away from a desktop. It is a manual tool: set
// TOKI_PREVIEW_DATA to a data folder (the one with library.json, settings and
// thumbs) and TOKI_PREVIEW_OUT to where the pictures go. The LIBRA_PREVIEW_*
// names are the ones this tool had before the app was named Toki.
//
//	TOKI_PREVIEW_DATA="$PWD/.preview" \
//	TOKI_PREVIEW_OUT=../docs/screenshots go test -run TestPreview -v
func TestPreview(t *testing.T) {
	data := previewEnv("TOKI_PREVIEW_DATA", "LIBRA_PREVIEW_DATA")
	out := previewEnv("TOKI_PREVIEW_OUT", "LIBRA_PREVIEW_OUT")
	if data == "" || out == "" {
		t.Skip("set TOKI_PREVIEW_DATA and TOKI_PREVIEW_OUT to render the views")
	}
	theme := previewTheme()
	a := previewApp(data, theme)
	if len(a.photos) == 0 {
		t.Fatalf("no photos in %s", data)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	save := func(name string, img *ui.Tester) {
		shot := img.Image()
		path := filepath.Join(out, previewName(name, theme))
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, shot); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%dx%d)", path, shot.Bounds().Dx(), shot.Bounds().Dy())
	}

	// The grid, with the demo library the app scans.
	a.mode = 0
	grid := ui.NewTester(a.view, 1180, 760)
	grid.Frame()
	save("ui-grid.png", grid)

	// The same grid at the smallest the window goes, where the toolbar is
	// tightest. A grid belongs to one window at a time, so this is a second
	// app on the same library.
	b := previewApp(data, theme)
	narrow := ui.NewTester(b.view, 780, 560)
	narrow.Frame()
	save("ui-grid-narrow.png", narrow)

	// The one-photo view with the properties panel.
	a.mode = 2
	a.current = a.photos[0].Path
	a.editOpen = false
	one := ui.NewTester(a.view, 1180, 760)
	one.Frame()
	save("ui-oneup.png", one)

	// The one-photo view with the edit panel open.
	a.openEditPanel(a.lib.Get(a.photos[0].Path))
	edit := ui.NewTester(a.view, 1180, 760)
	edit.Frame()
	save("ui-edit.png", edit)

	// The timeline.
	a.mode = 1
	a.editOpen = false
	timeline := ui.NewTester(a.view, 1180, 760)
	timeline.Frame()
	save("ui-timeline.png", timeline)

	// The export dialog.
	a.mode = 0
	a.exportOpen = true
	dialog := ui.NewTester(a.view, 1180, 760)
	dialog.Frame()
	save("ui-export.png", dialog)

	// The lightbox a double click opens: the photo over the mask.
	a.exportOpen = false
	a.openLightbox(a.photos[0].Path)
	light := ui.NewTester(a.view, 1180, 760)
	light.Frame()
	save("ui-lightbox.png", light)

	// The same photo zoomed in twice, off the middle.
	a.zoomAround(2, 760, 380, 1180-48, 760-2*44-24)
	zoomed := ui.NewTester(a.view, 1180, 760)
	zoomed.Frame()
	save("ui-lightbox-zoom.png", zoomed)

	// And smaller than the fit, which the wheel reaches too.
	a.lightZoom = 0.5
	half := ui.NewTester(a.view, 1180, 760)
	half.Frame()
	save("ui-lightbox-half.png", half)
	a.closeLightbox()
}
