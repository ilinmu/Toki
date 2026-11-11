package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// flowApp makes an app showing two photos, and a window to drive it.
func flowApp(t *testing.T) (*app, *ui.Tester, string, string) {
	t.Helper()
	a := testApp(t)
	dir := t.TempDir()
	first := writePNG(t, dir, "one.png", 80, 60, 40)
	second := writePNG(t, dir, "two.png", 80, 60, 200)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()
	a.inspectorOpen = false
	view := ui.NewTester(a.view, 1100, 720)
	view.Frame()
	return a, view, first, second
}

// TestTheHeartWorksOnACardThatIsNotChosen is the bug the card's heart had: it
// showed while the picture was under the pointer and went away as the pointer
// reached it, so the click fell through to the card and only chose the photo.
func TestTheHeartWorksOnACardThatIsNotChosen(t *testing.T) {
	a, view, first, second := flowApp(t)

	// Rest the pointer on the second photo, away from its heart.
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

	photo := a.lib.Get(second)
	if photo == nil || !photo.Favorite {
		t.Fatal("clicking the heart of an unchosen card did not favorite it")
	}
	if a.lib.Get(first).Favorite {
		t.Fatal("the other photo was favorited too")
	}
	// Choosing it was not the point of the click.
	if a.sel.Has(second) {
		t.Fatal("the heart's click also chose the photo")
	}
}

// TestTheCheckBoxChoosesOneMorePhoto is what the mark on a card does: it adds
// the photo to the choice or takes it out, and leaves the others chosen.
func TestTheCheckBoxChoosesOneMorePhoto(t *testing.T) {
	a, view, first, second := flowApp(t)

	// Choose the first photo by its picture.
	rect, ok := view.Find("one.png")
	if !ok {
		t.Fatalf("no card for the first photo: %q", view.Texts())
	}
	view.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	view.Frame()
	if !a.sel.Has(first) || a.sel.Len() != 1 {
		t.Fatalf("a click chose %d photos", a.sel.Len())
	}

	// Hover the second card so its check box shows, then tick it.
	rect2, ok := view.Find("two.png")
	if !ok {
		t.Fatalf("no card for the second photo: %q", view.Texts())
	}
	view.Move(rect2.X+rect2.W/2, rect2.Y+rect2.H/2)
	view.Frame()
	box, ok := view.Access("选择这张照片")
	if !ok {
		t.Fatalf("no check box on the card under the pointer: %q", view.Texts())
	}
	view.ClickAt(box.Bounds.X+box.Bounds.W/2, box.Bounds.Y+box.Bounds.H/2)
	view.Frame()

	if !a.sel.Has(first) {
		t.Fatal("ticking the second card cleared the first")
	}
	if !a.sel.Has(second) || a.sel.Len() != 2 {
		t.Fatalf("ticking the second card chose %d photos, want 2", a.sel.Len())
	}

	// The same box ticks it off again, at the same place.
	view.ClickAt(box.Bounds.X+box.Bounds.W/2, box.Bounds.Y+box.Bounds.H/2)
	view.Frame()
	if a.sel.Has(second) {
		t.Fatal("ticking the box again did not untick the second photo")
	}
	if !a.sel.Has(first) || a.sel.Len() != 1 {
		t.Fatalf("unticking the second left %d chosen, want the first only", a.sel.Len())
	}
}

// TestTheDateRailFollowsTheFlow checks what the rail stands for: a day sits at
// the scroll offset that shows its first photo, not at a fraction of the
// content's height.
func TestTheDateRailFollowsTheFlow(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	for i, name := range []string{"a.png", "b.png", "c.png", "d.png"} {
		writePNG(t, dir, name, 80, 60+i*20, uint8(i*40))
	}
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()
	// Four days, so the masonry has a date for each.
	for i, p := range a.lib.Snapshot() {
		p.TakenUnix -= int64(i * 86400)
	}
	a.set.ThumbSize = 2
	a.mode, a.layout = 0, 1
	a.viewKey = ""

	view := ui.NewTester(a.view, 1180, 760)
	view.Frame()
	view.Frame()

	// The dates the rail shows, newest first. The rail is one target: the
	// date nearest the pointer takes a click.
	var days []string
	for _, p := range a.lib.Snapshot() {
		if label := dayLabel(p.Taken()); len(days) == 0 || days[len(days)-1] != label {
			days = append(days, label)
		}
	}
	if len(days) < 2 {
		t.Fatalf("the rail shows %d dates, want at least 2", len(days))
	}
	rail, ok := view.Access("时间轴")
	if !ok {
		t.Fatalf("the rail is not reachable: %q", view.Texts())
	}
	railX := rail.Bounds.X + rail.Bounds.W/2

	// The bottom of the rail is the oldest photos, and the top the newest.
	view.ClickAt(railX, rail.Bounds.Y+rail.Bounds.H-6)
	view.Advance(time.Second)
	view.Frame()
	atBottom := a.libScroll.Y
	if atBottom <= 0 {
		t.Fatalf("the bottom of the rail scrolled to %v, want the oldest photos", atBottom)
	}
	view.ClickAt(railX, rail.Bounds.Y+6)
	view.Advance(time.Second)
	view.Frame()
	if a.libScroll.Y >= atBottom {
		t.Fatalf("the top of the rail scrolled to %v, not before %v", a.libScroll.Y, atBottom)
	}

	// A click between two ticks still picks the nearer date: the whole
	// track is a target, not only the pixels of a tick.
	view.ClickAt(railX, rail.Bounds.Y+rail.Bounds.H*0.75)
	view.Advance(time.Second)
	view.Frame()
	if a.libScroll.Y <= 0 {
		t.Fatalf("a click on the rail between ticks did nothing: %v", a.libScroll.Y)
	}
	if filepath.Dir(a.current) != dir && a.current != "" {
		t.Fatalf("the click opened something else: %q", a.current)
	}
}

// TestCommandKPutsTheKeyboardInTheSearch is the title bar's search responding
// to the keyboard: Command-K focuses the field and what is typed filters the
// library, Escape clears it.
func TestCommandKPutsTheKeyboardInTheSearch(t *testing.T) {
	a, view, first, second := flowApp(t)

	view.Key(ui.Cmd, ui.KeyK)
	view.Frame()
	view.Type("one")
	view.Frame()
	if a.query != "one" {
		t.Fatalf("typing after ⌘K left the query %q", a.query)
	}
	if len(a.photos) != 1 || a.photos[0].Path != first {
		t.Fatalf("the search shows %d photos, want the first only", len(a.photos))
	}

	// Escape empties the field, and the second photo comes back.
	view.Key(0, ui.KeyEscape)
	view.Frame()
	if a.query != "" {
		t.Fatalf("Escape left the query %q", a.query)
	}
	if len(a.photos) != 2 || !containsPhoto(a.photos, second) {
		t.Fatalf("the cleared search shows %d photos, want both", len(a.photos))
	}
}

// TestTheTimelineComesFromTheToolbarAlone is the sidebar no longer repeating
// the timeline: the arrangement buttons in the toolbar and View > 时间轴
// (⌘3) switch to it, and the sidebar is left to the places of the library.
func TestTheTimelineComesFromTheToolbarAlone(t *testing.T) {
	a, view, _, _ := flowApp(t)

	if err := view.Click("时间轴"); err == nil {
		t.Fatalf("the sidebar still has a timeline entry: %q", view.Texts())
	}
	if err := view.Click("时间轴列表 (⌘3)"); err != nil {
		t.Fatalf("the toolbar's timeline button is gone: %v", err)
	}
	view.Frame()
	if a.mode != 1 {
		t.Fatalf("mode %d after the toolbar's timeline button, want 1", a.mode)
	}
}

// TestTheTitleBarNamesTheAppOnce is the brand: one name, the Chinese one,
// where the mark is; the Latin spelling waits for a translated UI.
func TestTheTitleBarNamesTheAppOnce(t *testing.T) {
	_, view, _, _ := flowApp(t)

	if !view.HasText(brandName) {
		t.Fatalf("the title bar has no brand name: %q", view.Texts())
	}
	if view.HasText(brandNameLatin) {
		t.Fatalf("the title bar shows the Latin name too: %q", view.Texts())
	}
}

// TestTheDateRailDragsAndWheelScrolls is the rail's pointer contract: the big
// handle follows a drag, and a wheel over the rail scrolls the flow in the
// same direction.
func TestTheDateRailDragsAndWheelScrolls(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	for i, name := range []string{"a.png", "b.png", "c.png", "d.png", "e.png", "f.png",
		"g.png", "h.png", "i.png", "j.png", "k.png", "l.png"} {
		writePNG(t, dir, name, 80, 60+i*10, uint8(i*20))
	}
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()
	for i, p := range a.lib.Snapshot() {
		p.TakenUnix -= int64(i * 86400)
	}
	a.set.ThumbSize = 2
	a.mode, a.layout = 0, 1
	a.viewKey = ""

	view := ui.NewTester(a.view, 1180, 760)
	view.Frame()
	view.Frame()
	rail, ok := view.Access("时间轴")
	if !ok {
		t.Fatalf("the rail is not reachable: %q", view.Texts())
	}
	handle, ok := view.Access("滚动手柄")
	if !ok {
		t.Fatalf("the handle is not reachable: %q", view.Texts())
	}
	firstY := handle.Bounds.Y
	x, y := handle.Bounds.X+8, handle.Bounds.Y+14
	view.Press(x, y)
	view.Frame()
	view.Move(x, y+100)
	view.Frame()
	view.Release(x, y+100)
	view.Frame()
	if a.libScroll.Y <= 0 {
		t.Fatalf("dragging the handle did not scroll: %v", a.libScroll.Y)
	}
	after, _ := view.Access("滚动手柄")
	if after.Bounds.Y <= firstY {
		t.Fatalf("the handle did not follow the drag: %v then %v", firstY, after.Bounds.Y)
	}

	// A wheel over the rail scrolls down, and the handle moves down with it.
	before := a.libScroll.Y
	view.Wheel(rail.Bounds.X+12, rail.Bounds.Y+rail.Bounds.H/2, 0, 5)
	view.Frame()
	if a.libScroll.Y <= before {
		t.Fatalf("the wheel over the rail did not scroll: %v then %v", before, a.libScroll.Y)
	}
	wheeled, _ := view.Access("滚动手柄")
	if wheeled.Bounds.Y <= after.Bounds.Y {
		t.Fatalf("the handle moved against the wheel: %v then %v", after.Bounds.Y, wheeled.Bounds.Y)
	}
}
