package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
)

// exportSizes are the widths an export can scale a picture to, 0 for the
// original size, which is what most exports want.
var exportSizes = []int{0, 320, 480, 640, 800, 1024, 1200, 1600}

// exportSizeNames name the sizes for the picker.
var exportSizeNames = []string{"原始尺寸", "320 像素", "480 像素", "640 像素", "800 像素", "1024 像素", "1200 像素", "1600 像素"}

// exportQualities are the JPEG qualities an export offers.
var exportQualities = []struct {
	Name  string
	Value int
}{
	{"自动", 88},
	{"普通", 82},
	{"最高", 95},
	{"最低", 70},
}

// exportOutcome reports what an export did.
type exportOutcome struct {
	Done, Failed int
	Errors       []string
	Dir          string
}

// exportPhotos writes copies of photos with their edits applied, scaled to
// the wanted size, stamped with the watermark, into a folder.
func (a *app) exportPhotos(photos []*Photo, form exportForm, quality int) exportOutcome {
	out := exportOutcome{}
	dir := form.Dir
	if strings.TrimSpace(form.Folder) != "" {
		dir = filepath.Join(form.Dir, strings.TrimSpace(form.Folder))
	}
	out.Dir = dir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		out.Errors = append(out.Errors, err.Error())
		return out
	}
	width := 0
	if form.Size >= 0 && form.Size < len(exportSizes) {
		width = exportSizes[form.Size]
	}
	used := map[string]int{}
	for i, p := range photos {
		img, err := decodeImage(p.Path)
		if err != nil {
			out.Failed++
			out.Errors = append(out.Errors, p.Name+": "+err.Error())
			continue
		}
		info := readEXIFFromFile(p.Path)
		img = applyOrientation(img, info.Orientation)
		img = renderEdit(img, p.Edit)
		if width > 0 {
			b := img.Bounds()
			w, h := fitSize(b.Dx(), b.Dy(), width, width)
			img = resize(img, w, h)
		}
		if form.Watermark != "" {
			img = withWatermark(img, form.Watermark)
		}
		base := strings.TrimSuffix(p.Name, filepath.Ext(p.Name))
		if form.Numbers {
			base = fmt.Sprintf("%03d-%s", i+1, base)
		}
		name := uniqueName(used, dir, base)
		if err := encodeJPEGFile(filepath.Join(dir, name), img, quality); err != nil {
			out.Failed++
			out.Errors = append(out.Errors, p.Name+": "+err.Error())
			continue
		}
		out.Done++
	}
	return out
}

// uniqueName returns base.jpg, or base-2.jpg and so on when the folder or the
// export already has that name.
func uniqueName(used map[string]int, dir, base string) string {
	name := base + ".jpg"
	for {
		n := used[strings.ToLower(name)]
		if n > 0 {
			name = fmt.Sprintf("%s-%d.jpg", base, n+1)
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			used[strings.ToLower(name)] = 1
			name = fmt.Sprintf("%s-2.jpg", base)
			continue
		}
		used[strings.ToLower(name)] = 1
		return name
	}
}

// withWatermark stamps text on a picture.
func withWatermark(img image.Image, text string) image.Image {
	rgba := toRGBA(img)
	drawWatermark(rgba, text)
	return rgba
}

// runExport exports in the background and reports in the status bar.
func (a *app) runExport(photos []*Photo, form exportForm) {
	quality := exportQualities[a.set.ExportQuality%len(exportQualities)].Value
	a.exportOpen = false
	a.status = fmt.Sprintf("正在导出 %d 张…", len(photos))
	go func() {
		out := a.exportPhotos(photos, form, quality)
		mygo.RunOnMain(func() {
			if out.Failed == 0 {
				a.status = fmt.Sprintf("已导出 %d 张到 %s", out.Done, out.Dir)
			} else {
				a.status = fmt.Sprintf("导出完成：%d 成功，%d 失败（%s）", out.Done, out.Failed, out.Errors[0])
			}
			a.refreshUI()
		})
	}()
}
