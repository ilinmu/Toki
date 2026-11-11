package main

import "github.com/egoist/mygo/ui"

// Preview sizes the cache keeps for the views that show one photo large.
const (
	previewSmall  = 800
	previewMedium = 1600
	previewLarge  = 2600
)

// previewBitmap is the picture a view shows for a photo: the size it asked
// for when that is to hand, else a sharper or smaller one already made, and
// meanwhile it asks the cache to make the size it wants. Nothing is decoded
// and no file is read on the calling thread, so swapping to a photo that was
// never shown draws its bars and its mask at once, with the picture the grid
// already has, and the large one arrives when it is ready.
func (a *app) previewBitmap(p *Photo, want int) (*ui.Bitmap, bool) {
	st := a.viewEdit(p)
	if b, ok := a.thumbs.Peek(p.Path, p.ModUnix, want, st); ok {
		return b, true
	}
	// Ask for the size wanted, and for a smaller one too when the screen is
	// large: the smaller arrives sooner.
	a.thumbs.Request(p.Path, p.ModUnix, want, st)
	if b, ok := a.thumbs.Peek(p.Path, p.ModUnix, previewSmall, st); ok && want > previewSmall {
		return b, true
	}
	// Whatever this photo showed last, even the grid's thumbnail.
	if b, ok := a.thumbs.Latest(p.Path, p.ModUnix); ok {
		return b, true
	}
	return nil, false
}

// prefetchNeighbors asks the cache for the pictures either side of the one
// shown, so stepping to them shows a picture at once instead of waiting.
func (a *app) prefetchNeighbors(p *Photo, want int) {
	for _, delta := range []int{-1, 1} {
		n := a.neighbor(p, delta)
		if n == nil {
			continue
		}
		a.thumbs.Request(n.Path, n.ModUnix, want, n.Edit)
	}
}

// neighbor is the photo delta places from p in the photos being browsed: the
// library's, or the folder a file was opened from.
func (a *app) neighbor(p *Photo, delta int) *Photo {
	i, ok := a.shownIndex(p.Path)
	if !ok {
		return nil
	}
	photos := a.shownPhotos()
	j := i + delta
	if j < 0 || j >= len(photos) {
		return nil
	}
	return photos[j]
}

// previewFor returns the size to make for an area at a zoom: enough pixels
// for a Retina screen, as one of the sizes the cache keeps.
func previewFor(areaW, areaH float32, zoom float64) int {
	need := float64(max(areaW, areaH)) * 2 * maxFloat(zoom, 1)
	switch {
	case need <= 900:
		return previewSmall
	case need <= 1700:
		return previewMedium
	}
	return previewLarge
}

// maxFloat is the larger of two numbers.
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
