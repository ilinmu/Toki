package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// exifJPEG builds a JPEG of the smallest kind that carries an EXIF segment
// with a camera's fields, the way a camera writes them.
func exifJPEG(t *testing.T) []byte {
	t.Helper()
	const (
		ifd0Offset = 8
		ifd0Size   = 2 + 4*12 + 4
		exifOffset = ifd0Offset + ifd0Size
		exifSize   = 2 + 5*12 + 4
		dataOffset = exifOffset + exifSize
	)
	buf := &bytes.Buffer{}
	le := binary.LittleEndian
	// TIFF header.
	buf.WriteString("II")
	binary.Write(buf, le, uint16(42))
	binary.Write(buf, le, uint32(ifd0Offset))

	entry := func(tag, typ uint16, count uint32, value uint32) {
		binary.Write(buf, le, tag)
		binary.Write(buf, le, typ)
		binary.Write(buf, le, count)
		binary.Write(buf, le, value)
	}
	// IFD0: orientation, make, model, and where the EXIF IFD is.
	binary.Write(buf, le, uint16(4))
	entry(0x0112, 3, 1, 6)           // Orientation: rotate 90 to show
	entry(0x010F, 2, 4, 0x00434241)  // Make: "ABC\0"
	entry(0x0110, 2, 3, 0x00005958)  // Model: "XY\0"
	entry(0x8769, 4, 1, exifOffset)  // ExifIFD
	binary.Write(buf, le, uint32(0)) // no next IFD

	// The EXIF IFD.
	binary.Write(buf, le, uint16(5))
	entry(0x9003, 2, 20, dataOffset)   // DateTimeOriginal
	entry(0x8827, 3, 1, 400)           // ISO
	entry(0x829D, 5, 1, dataOffset+20) // FNumber
	entry(0x829A, 5, 1, dataOffset+28) // ExposureTime
	entry(0x920A, 5, 1, dataOffset+36) // FocalLength
	binary.Write(buf, le, uint32(0))

	// The values too long to sit in an entry.
	buf.WriteString("2024:05:06 07:08:09\x00")
	for _, pair := range [][2]uint32{{28, 10}, {1, 125}, {50, 1}} {
		binary.Write(buf, le, pair[0])
		binary.Write(buf, le, pair[1])
	}

	tiff := buf.Bytes()
	payload := append([]byte("Exif\x00\x00"), tiff...)
	size := len(payload) + 2
	out := []byte{0xFF, 0xD8, 0xFF, 0xE1, byte(size >> 8), byte(size)}
	out = append(out, payload...)
	out = append(out, 0xFF, 0xD9)
	return out
}

func TestReadEXIF(t *testing.T) {
	info := readEXIF(exifJPEG(t))
	if info.Orientation != 6 {
		t.Errorf("orientation %d, want 6", info.Orientation)
	}
	if info.Make != "ABC" || info.Model != "XY" {
		t.Errorf("camera %q %q, want ABC XY", info.Make, info.Model)
	}
	if got := info.DateTimeOriginal.Format("2006-01-02 15:04:05"); got != "2024-05-06 07:08:09" {
		t.Errorf("date %q", got)
	}
	if info.ISO != 400 {
		t.Errorf("ISO %d, want 400", info.ISO)
	}
	if info.FNumber < 2.79 || info.FNumber > 2.81 {
		t.Errorf("f-number %.2f, want 2.8", info.FNumber)
	}
	if info.ExposureTime != "1/125 s" {
		t.Errorf("exposure %q", info.ExposureTime)
	}
	if info.FocalLength != 50 {
		t.Errorf("focal length %d, want 50", info.FocalLength)
	}
}

func TestParseEXIFTime(t *testing.T) {
	got := parseEXIFTime("2019:12:31 23:59:58")
	if got.Year() != 2019 || got.Month() != time.December || got.Second() != 58 {
		t.Fatalf("parsed %v", got)
	}
	if !parseEXIFTime("garbage").IsZero() {
		t.Fatal("garbage parsed as a date")
	}
}

func TestMetadataRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(path, exifJPEG(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeMetadata(path, `A <caption> & "quotes"`, []string{"家庭", "旅行"}, 5); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data[:2]), "\xff\xd8") {
		t.Fatal("the file is no longer a JPEG")
	}
	if n := bytes.Count(data, []byte(xmpHeader)); n != 1 {
		t.Fatalf("%d XMP segments, want 1", n)
	}
	for _, want := range []string{"&lt;caption&gt;", "&amp;", "家庭", "旅行", "<xmp:Rating>5</xmp:Rating>"} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("metadata lacks %q", want)
		}
	}
	// The camera's own EXIF survives the write.
	info := readEXIF(data)
	if info.Make != "ABC" || info.Orientation != 6 {
		t.Fatalf("EXIF lost: %+v", info)
	}
	// Writing again replaces the packet instead of adding another one.
	if err := writeMetadata(path, "第二版", nil, 1); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(data, []byte(xmpHeader)); n != 1 {
		t.Fatalf("%d XMP segments after the second write, want 1", n)
	}
	if !bytes.Contains(data, []byte("第二版")) {
		t.Error("the new caption is missing")
	}
}

func TestMetadataSidecarForOtherKinds(t *testing.T) {
	dir := t.TempDir()
	path := writePNG(t, dir, "shot.png", 8, 8, 4)
	if err := writeMetadata(path, "sidecar", []string{"tag"}, 3); err != nil {
		t.Fatal(err)
	}
	side := sidecarPath(path)
	data, err := os.ReadFile(side)
	if err != nil {
		t.Fatalf("no sidecar: %v", err)
	}
	if !strings.Contains(string(data), "sidecar") {
		t.Error("the sidecar lacks the caption")
	}
}

func TestViewsRender(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	path := writePNG(t, dir, "beach.png", 60, 40, 55)
	a.lib.AddFolder(dir)
	a.scanFolder(dir, nil)
	a.rebuild()

	grid := ui.NewTester(a.view, 1100, 720)
	if !grid.HasText("beach.png") {
		t.Fatalf("the grid does not show the photo: %q", grid.Texts())
	}
	if !grid.HasText("全部照片") {
		t.Fatal("the sidebar is missing")
	}

	a.mode = 1
	timeline := ui.NewTester(a.view, 1100, 720)
	if !timeline.HasText("beach.png") {
		t.Fatalf("the timeline does not show the photo: %q", timeline.Texts())
	}
	if !timeline.HasText(monthLabel(a.lib.Get(path))) {
		t.Fatalf("the timeline has no month heading: %q", timeline.Texts())
	}

	a.mode = 2
	a.current = path
	one := ui.NewTester(a.view, 1100, 720)
	for _, want := range []string{"beach.png", "信息", "文件名", "拍摄时间"} {
		if !one.HasText(want) {
			t.Fatalf("the one-photo view lacks %q: %q", want, one.Texts())
		}
	}

	// The search box filters what the grid shows.
	a.mode = 0
	a.query = "nothing-matches"
	a.viewKey = ""
	empty := ui.NewTester(a.view, 1100, 720)
	if !empty.HasText("这里还没有照片") {
		t.Fatalf("an empty search does not say so: %q", empty.Texts())
	}
}
