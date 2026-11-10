package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

// imageExts are the kinds of file the library takes in.
var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".jpe": true, ".jfif": true,
	".png": true, ".gif": true, ".bmp": true, ".webp": true,
	".tif": true, ".tiff": true,
}

// isImagePath reports whether path names a picture the library reads.
func isImagePath(path string) bool {
	return imageExts[strings.ToLower(filepath.Ext(path))]
}

// decodeImage reads a picture of any supported kind.
func decodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return decode(f, strings.ToLower(filepath.Ext(path)))
}

// decode reads a picture from r, as its extension says.
func decode(r io.Reader, ext string) (image.Image, error) {
	switch ext {
	case ".jpg", ".jpeg", ".jpe", ".jfif":
		return jpeg.Decode(r)
	case ".png":
		return png.Decode(r)
	case ".gif":
		return gif.Decode(r)
	case ".bmp":
		return bmp.Decode(r)
	case ".webp":
		return webp.Decode(r)
	case ".tif", ".tiff":
		return tiff.Decode(r)
	}
	return nil, errors.New("unsupported picture: " + ext)
}

// imageDecodeConfig reads a picture's size and kind without decoding its
// pixels.
func imageDecodeConfig(r io.Reader, ext string) (image.Config, string, error) {
	var cfg image.Config
	var err error
	switch ext {
	case ".jpg", ".jpeg", ".jpe", ".jfif":
		cfg, err = jpeg.DecodeConfig(r)
	case ".png":
		cfg, err = png.DecodeConfig(r)
	case ".gif":
		cfg, err = gif.DecodeConfig(r)
	case ".bmp":
		cfg, err = bmp.DecodeConfig(r)
	case ".webp":
		cfg, err = webp.DecodeConfig(r)
	case ".tif", ".tiff":
		cfg, err = tiff.DecodeConfig(r)
	default:
		return image.Config{}, "", errors.New("unsupported picture: " + ext)
	}
	return cfg, ext, err
}

// applyOrientation turns a picture upright as its EXIF orientation says.
func applyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return transform(img, func(w, h, x, y int) (int, int) { return w - 1 - x, y }, true)
	case 3:
		return transform(img, func(w, h, x, y int) (int, int) { return w - 1 - x, h - 1 - y }, true)
	case 4:
		return transform(img, func(w, h, x, y int) (int, int) { return x, h - 1 - y }, true)
	case 5:
		return transform(img, func(w, h, x, y int) (int, int) { return y, x }, false)
	case 6:
		return transform(img, func(w, h, x, y int) (int, int) { return h - 1 - y, x }, false)
	case 7:
		return transform(img, func(w, h, x, y int) (int, int) { return h - 1 - y, w - 1 - x }, false)
	case 8:
		return transform(img, func(w, h, x, y int) (int, int) { return y, w - 1 - x }, false)
	}
	return img
}

// transform copies img into a new picture, placing source pixel (x,y) at the
// place mapTo gives. swapped says the result has the source's width and
// height exchanged.
func transform(img image.Image, mapTo func(w, h, x, y int) (int, int), same bool) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	outW, outH := w, h
	if !same {
		outW, outH = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := mapTo(w, h, x, y)
			if dx < 0 || dy < 0 || dx >= outW || dy >= outH {
				continue
			}
			dst.Set(dx, dy, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// fitSize scales w×h down to fit maxW×maxH, keeping its shape, and never
// scales up.
func fitSize(w, h, maxW, maxH int) (int, int) {
	if w <= 0 || h <= 0 || maxW <= 0 || maxH <= 0 {
		return w, h
	}
	scale := math.Min(float64(maxW)/float64(w), float64(maxH)/float64(h))
	if scale >= 1 {
		return w, h
	}
	return int(math.Max(1, math.Round(float64(w)*scale))), int(math.Max(1, math.Round(float64(h)*scale)))
}

// resize scales img to w×h with an area filter, so pictures shown small keep
// their detail instead of dropping pixels.
func resize(img image.Image, w, h int) *image.RGBA {
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	src := toRGBA(img)
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw == w && sh == h {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	scaleX := float64(sw) / float64(w)
	scaleY := float64(sh) / float64(h)
	for y := 0; y < h; y++ {
		y0 := int(float64(y) * scaleY)
		y1 := int(math.Ceil(float64(y+1) * scaleY))
		if y1 <= y0 {
			y1 = y0 + 1
		}
		if y1 > sh {
			y1 = sh
		}
		for x := 0; x < w; x++ {
			x0 := int(float64(x) * scaleX)
			x1 := int(math.Ceil(float64(x+1) * scaleX))
			if x1 <= x0 {
				x1 = x0 + 1
			}
			if x1 > sw {
				x1 = sw
			}
			var r, g, b, a, n uint32
			for yy := y0; yy < y1; yy++ {
				row := src.Pix[yy*src.Stride:]
				for xx := x0; xx < x1; xx++ {
					p := row[xx*4:]
					r += uint32(p[0])
					g += uint32(p[1])
					b += uint32(p[2])
					a += uint32(p[3])
					n++
				}
			}
			if n == 0 {
				continue
			}
			o := dst.Pix[y*dst.Stride+x*4:]
			o[0] = uint8(r / n)
			o[1] = uint8(g / n)
			o[2] = uint8(b / n)
			o[3] = uint8(a / n)
		}
	}
	return dst
}

// toRGBA converts any picture to RGBA once, so pixel loops stay simple.
func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok {
		return r
	}
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
}

// renderEdit applies a photo's edit to a picture: its rotation and flip, its
// straighten angle, and its crop. The adjustments of the picker's own engine
// are not part of this selection of features.
func renderEdit(src image.Image, st EditState) image.Image {
	out := applyOrientation(src, 0)
	switch st.Rotate % 360 {
	case 90:
		out = transform(out, func(w, h, x, y int) (int, int) { return h - 1 - y, x }, false)
	case 180:
		out = transform(out, func(w, h, x, y int) (int, int) { return w - 1 - x, h - 1 - y }, true)
	case 270:
		out = transform(out, func(w, h, x, y int) (int, int) { return y, w - 1 - x }, false)
	}
	if st.FlipH || st.FlipV {
		b := out.Bounds()
		w, h := b.Dx(), b.Dy()
		flipH, flipV := st.FlipH, st.FlipV
		out = transform(out, func(w, h, x, y int) (int, int) {
			if flipH {
				x = w - 1 - x
			}
			if flipV {
				y = h - 1 - y
			}
			return x, y
		}, true)
		_ = w
		_ = h
	}
	if st.Angle != 0 {
		out = rotate(out, st.Angle)
	}
	if st.HasCrop {
		out = cropNorm(out, st.CropX, st.CropY, st.CropW, st.CropH)
	}
	if st.Exposure != 0 || st.Contrast != 0 || st.Saturation != 0 || st.Temp != 0 || st.Vignette != 0 {
		out = applyColor(out, st)
	}
	return out
}

// applyColor renders the light and color adjustments of an edit: exposure,
// contrast, saturation, the warmth of the light and a vignette. They are
// simple pixel operations with the same values the editor's sliders show, so
// what the panel previews is what an export writes.
func applyColor(src image.Image, st EditState) image.Image {
	rgba := toRGBA(src)
	b := rgba.Bounds()
	w, h := b.Dx(), b.Dy()
	exposure := math.Pow(2, st.Exposure) // 0.5× .. 2×
	contrast := 1 + st.Contrast          // 0 .. 2 around the middle
	saturation := 1 + st.Saturation      // 0 (gray) .. 2
	warm := clamp(-1, 1, st.Temp)        // -1 cool .. 1 warm
	vignette := clamp(0, 1, st.Vignette) // 0 none .. 1 strong
	cx, cy := float64(w)/2, float64(h)/2
	radius := math.Hypot(cx, cy)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := rgba.PixOffset(x+b.Min.X, y+b.Min.Y)
			r := float64(rgba.Pix[i]) / 255
			g := float64(rgba.Pix[i+1]) / 255
			bl := float64(rgba.Pix[i+2]) / 255

			// Exposure first: light, then the curve.
			r *= exposure
			g *= exposure
			bl *= exposure
			r = (r-0.5)*contrast + 0.5
			g = (g-0.5)*contrast + 0.5
			bl = (bl-0.5)*contrast + 0.5

			// Saturation around the luma of the pixel.
			luma := 0.2126*r + 0.7152*g + 0.0722*bl
			r = luma + (r-luma)*saturation
			g = luma + (g-luma)*saturation
			bl = luma + (bl-luma)*saturation

			// Warmth: more red and less blue, or the other way round.
			r += warm * 0.10
			bl -= warm * 0.10

			// Vignette: the corners darker, the middle untouched.
			if vignette > 0 {
				d := math.Hypot(float64(x)-cx, float64(y)-cy) / radius
				fall := 1 - vignette*0.85*smoothstep(0.35, 1, d)
				r *= fall
				g *= fall
				bl *= fall
			}
			rgba.Pix[i] = clampByte(r)
			rgba.Pix[i+1] = clampByte(g)
			rgba.Pix[i+2] = clampByte(bl)
		}
	}
	return rgba
}

// smoothstep eases t from 0 to 1 between lo and hi.
func smoothstep(lo, hi, t float64) float64 {
	if hi <= lo {
		return 0
	}
	t = clamp(lo, hi, t)
	t = (t - lo) / (hi - lo)
	return t * t * (3 - 2*t)
}

// clampByte turns a 0..1 channel into a byte.
func clampByte(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	}
	return uint8(v*255 + 0.5)
}

// rotate turns a picture by angle degrees, enlarging it to keep every corner.
func rotate(img image.Image, angle float64) image.Image {
	src := toRGBA(img)
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	rad := angle * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	// The turned bounding box.
	outW := int(math.Ceil(math.Abs(float64(sw)*cos) + math.Abs(float64(sh)*sin)))
	outH := int(math.Ceil(math.Abs(float64(sw)*sin) + math.Abs(float64(sh)*cos)))
	if outW <= 0 || outH <= 0 {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	cx, cy := float64(sw)/2, float64(sh)/2
	ocx, ocy := float64(outW)/2, float64(outH)/2
	for y := 0; y < outH; y++ {
		for x := 0; x < outW; x++ {
			// Rotate the destination pixel back into the source.
			dx, dy := float64(x)-ocx, float64(y)-ocy
			sx := cos*dx + sin*dy + cx
			sy := -sin*dx + cos*dy + cy
			if sx < 0 || sy < 0 || sx >= float64(sw)-1 || sy >= float64(sh)-1 {
				continue
			}
			dst.Set(x, y, bilinear(src, sx, sy))
		}
	}
	return dst
}

// bilinear samples src at (x, y).
func bilinear(src *image.RGBA, x, y float64) color.RGBA {
	x0, y0 := int(x), int(y)
	fx, fy := x-float64(x0), y-float64(y0)
	at := func(xx, yy int) (float64, float64, float64, float64) {
		p := src.Pix[yy*src.Stride+xx*4:]
		return float64(p[0]), float64(p[1]), float64(p[2]), float64(p[3])
	}
	r00, g00, b00, a00 := at(x0, y0)
	r10, g10, b10, a10 := at(x0+1, y0)
	r01, g01, b01, a01 := at(x0, y0+1)
	r11, g11, b11, a11 := at(x0+1, y0+1)
	mix := func(v00, v10, v01, v11 float64) uint8 {
		top := v00*(1-fx) + v10*fx
		bottom := v01*(1-fx) + v11*fx
		return uint8(math.Round(top*(1-fy) + bottom*fy))
	}
	return color.RGBA{R: mix(r00, r10, r01, r11), G: mix(g00, g10, g01, g11), B: mix(b00, b10, b01, b11), A: mix(a00, a10, a01, a11)}
}

// cropNorm cuts the rectangle (x, y, w, h) of img, given in fractions of it.
func cropNorm(img image.Image, x, y, w, h float64) image.Image {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	cx := int(math.Round(clamp01(x) * float64(iw)))
	cy := int(math.Round(clamp01(y) * float64(ih)))
	cw := int(math.Round(clamp01(w) * float64(iw)))
	ch := int(math.Round(clamp01(h) * float64(ih)))
	if cw < 1 {
		cw = 1
	}
	if ch < 1 {
		ch = 1
	}
	if cx+cw > iw {
		cx = iw - cw
	}
	if cy+ch > ih {
		cy = ih - ch
	}
	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}
	return cropRect(img, image.Rect(b.Min.X+cx, b.Min.Y+cy, b.Min.X+cx+cw, b.Min.Y+cy+ch))
}

// cropRect copies part of a picture.
func cropRect(img image.Image, r image.Rectangle) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), img, r.Min, draw.Src)
	return out
}

// clamp01 keeps v within 0..1.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// encodeJPEGFile writes img as a JPEG of the given quality, replacing path
// atomically. A JPEG has no transparency, so a picture with a see-through
// background is set on white first: left alone, its clear pixels encode as
// black.
func encodeJPEGFile(path string, img image.Image, quality int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	img = onWhite(img)
	tmp, err := os.CreateTemp(filepath.Dir(path), ".export-*.jpg")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if err := jpeg.Encode(tmp, img, &jpeg.Options{Quality: quality}); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, path)
}

// onWhite sets img on a white ground, the way a photo viewer shows a
// transparent picture, and returns it unchanged when it is opaque.
func onWhite(img image.Image) image.Image {
	if o, ok := img.(interface{ Opaque() bool }); ok && o.Opaque() {
		return img
	}
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Over)
	return dst
}

// watermark is the text stamped on an exported photo, drawn small in the
// bottom right corner.
func drawWatermark(img *image.RGBA, text string) {
	if text == "" {
		return
	}
	face := basicfont.Face7x13
	w := font.MeasureString(face, text).Ceil()
	b := img.Bounds()
	x := b.Max.X - w - 12
	y := b.Max.Y - 8
	drawer := &font.Drawer{Dst: img, Src: image.NewUniform(color.RGBA{0, 0, 0, 140}), Face: face,
		Dot: fixed.P(x+1, y+1)}
	drawer.DrawString(text)
	drawer.Src = image.NewUniform(color.RGBA{255, 255, 255, 230})
	drawer.Dot = fixed.P(x, y)
	drawer.DrawString(text)
}

// fingerprint identifies a picture's contents cheaply: its size and the first
// 64 KiB hashed. The importer uses it to skip pictures it already has.
func fingerprint(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.CopyN(h, f, 64<<10); err != nil && err != io.EOF {
		return "", err
	}
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)) + ":" + itoa(int(st.Size())), nil
}

// aspectRatios are the crop shapes the picker offers, with their width over
// height, 0 for the picture's own shape.
type aspectRatio struct {
	Name string
	W, H float64
}

var aspectRatios = []aspectRatio{
	{"原始比例", 0, 0}, {"1:1 方形", 1, 1}, {"4:3", 4, 3}, {"3:2", 3, 2}, {"16:9", 16, 9}, {"3:4 竖幅", 3, 4},
}

// cropForShape returns the largest rectangle of the given shape inside a
// picture of width w and height h, centered.
func cropForShape(w, h int, ar aspectRatio) (x, y, cw, ch float64) {
	if ar.W == 0 || ar.H == 0 || w <= 0 || h <= 0 {
		return 0, 0, 1, 1
	}
	target := ar.W / ar.H
	imgRatio := float64(w) / float64(h)
	if target > imgRatio {
		// Wider than the picture: full width, shorter height.
		fh := imgRatio / target
		return 0, (1 - fh) / 2, 1, fh
	}
	fw := target / imgRatio
	return (1 - fw) / 2, 0, fw, 1
}
