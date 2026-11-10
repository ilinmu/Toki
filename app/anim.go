package main

import (
	"errors"
	"image"
	"image/draw"
	"image/gif"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// The bounds of a playing animation: an animated GIF is kept small enough to
// hold every frame of it in memory, and long ones drop frames rather than
// grow without a limit.
const (
	animMaxSide   = 1200
	animMaxFrames = 120
	animMaxPixels = 40 << 20
	animMinDelay  = 20 * time.Millisecond
	animSlowDelay = 100 * time.Millisecond
)

// gifShow is the animated picture a view is playing. Only one plays at a
// time, so the frames of the photo shown are the only ones kept.
type gifShow struct {
	mu      sync.Mutex
	key     string // the photo, its stamp and the size of its frames
	frames  []*ui.Bitmap
	delays  []time.Duration
	loading bool
	start   time.Time
	started bool
}

// gifFrame is the frame of an animated photo to show now, or nil when the
// photo is not an animated GIF or its frames are not ready yet. It asks for
// the next frame when one is due.
func (a *app) gifFrame(c *ui.Context, p *Photo, want int) *ui.Bitmap {
	if !isGIFPath(p.Path) {
		return nil
	}
	size := min(want, animMaxSide)
	key := p.Path + "\x00" + itoa(int(p.ModUnix)) + "\x00" + itoa(size)

	a.anim.mu.Lock()
	if a.anim.key != key {
		old := a.anim.frames
		a.anim.key, a.anim.frames, a.anim.delays = key, nil, nil
		a.anim.loading, a.anim.started = false, false
		a.anim.mu.Unlock()
		releaseBitmaps(old)
		a.loadGIF(p.Path, key, size)
		return nil
	}
	if a.anim.loading || len(a.anim.frames) == 0 {
		a.anim.mu.Unlock()
		return nil
	}
	frames, delays := a.anim.frames, a.anim.delays
	if !a.anim.started {
		a.anim.started = true
		a.anim.start = c.Now()
	}
	started := a.anim.start
	a.anim.mu.Unlock()

	if len(frames) == 1 || len(delays) == 0 {
		return frames[0]
	}
	i, next := frameAt(c.Now().Sub(started), delays)
	if next > 0 {
		c.After(next)
	}
	return frames[i]
}

// loadGIF decodes a photo's frames away from the main thread and turns them
// into bitmaps where the window runs.
func (a *app) loadGIF(path, key string, size int) {
	a.anim.mu.Lock()
	if a.anim.key != key || a.anim.loading {
		a.anim.mu.Unlock()
		return
	}
	a.anim.loading = true
	a.anim.mu.Unlock()

	// A windowless run, as in the tests, decodes here and now: there is no
	// main loop to hand the bitmaps to.
	if !uiReady.Load() {
		frames, delays, err := readGIFFrames(path, size)
		if err == nil && len(frames) > 0 {
			bmps := make([]*ui.Bitmap, len(frames))
			for i, f := range frames {
				bmps[i] = ui.NewBitmap(f)
			}
			a.anim.frames, a.anim.delays = bmps, delays
		}
		a.anim.loading = false
		return
	}

	go func() {
		frames, delays, err := readGIFFrames(path, size)
		if err != nil || len(frames) == 0 || !uiReady.Load() {
			a.anim.mu.Lock()
			if a.anim.key == key {
				a.anim.loading = false
			}
			a.anim.mu.Unlock()
			return
		}
		mygo.RunOnMain(func() {
			bmps := make([]*ui.Bitmap, len(frames))
			for i, f := range frames {
				bmps[i] = ui.NewBitmap(f)
			}
			a.anim.mu.Lock()
			if a.anim.key != key {
				a.anim.mu.Unlock()
				releaseBitmaps(bmps)
				return
			}
			old := a.anim.frames
			a.anim.frames, a.anim.delays = bmps, delays
			a.anim.loading, a.anim.started = false, false
			a.anim.mu.Unlock()
			releaseBitmaps(old)
			a.refreshUI()
		})
	}()
}

// releaseBitmaps gives back the pixels of bitmaps no view shows any more.
func releaseBitmaps(bmps []*ui.Bitmap) {
	for _, b := range bmps {
		if b != nil {
			b.ReleasePixels()
		}
	}
}

// isGIFPath reports whether path names a GIF, the one kind of picture that
// may move.
func isGIFPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".gif")
}

// readGIFFrames decodes an animated GIF into the frames it shows, each fitted
// inside size and set on white, with how long each stays on screen.
func readGIFFrames(path string, size int) ([]*image.RGBA, []time.Duration, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, nil, err
	}
	if len(g.Image) == 0 {
		return nil, nil, errors.New("a GIF without frames")
	}
	box := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	if box.Empty() {
		box = g.Image[0].Bounds()
	}
	canvas := image.NewRGBA(box)
	var previous *image.RGBA
	w, h := fitSize(box.Dx(), box.Dy(), size, size)

	frames := make([]*image.RGBA, 0, len(g.Image))
	delays := make([]time.Duration, 0, len(g.Image))
	for i, fr := range g.Image {
		if i > 0 {
			// The frame before goes away as its disposal says: the ground
			// shows through again, or the frame before it comes back.
			switch g.Disposal[i-1] {
			case gif.DisposalBackground:
				clearRect(canvas, g.Image[i-1].Bounds())
			case gif.DisposalPrevious:
				if previous != nil {
					canvas = previous
					previous = nil
				}
			}
		}
		if i < len(g.Disposal) && g.Disposal[i] == gif.DisposalPrevious {
			previous = cloneRGBA(canvas)
		}
		draw.Draw(canvas, fr.Bounds(), fr, fr.Bounds().Min, draw.Over)
		frames = append(frames, resize(cloneRGBA(canvas), w, h))
		delays = append(delays, gifDelay(g.Delay, i))
	}
	frames, delays = fitFrames(frames, delays)
	return frames, delays, nil
}

// gifDelay is how long frame i stays, as browsers read a GIF's delay: a
// delay of none, or too short to notice, is a tenth of a second.
func gifDelay(delays []int, i int) time.Duration {
	if i >= len(delays) {
		return animSlowDelay
	}
	d := time.Duration(delays[i]) * 10 * time.Millisecond
	if d < animMinDelay {
		return animSlowDelay
	}
	return d
}

// fitFrames drops frames from a long animation until it fits the memory a
// playing picture may take, keeping each dropped frame's time on the one
// before it.
func fitFrames(frames []*image.RGBA, delays []time.Duration) ([]*image.RGBA, []time.Duration) {
	if len(frames) == 0 {
		return frames, delays
	}
	pixels := len(frames) * frames[0].Bounds().Dx() * frames[0].Bounds().Dy()
	stride := 1
	for len(frames)/stride > animMaxFrames || pixels/stride > animMaxPixels {
		stride++
	}
	if stride == 1 {
		return frames, delays
	}
	kept := make([]*image.RGBA, 0, len(frames)/stride+1)
	keptDelays := make([]time.Duration, 0, cap(kept))
	var wait time.Duration
	for i, fr := range frames {
		wait += delays[i]
		if (i+1)%stride == 0 || i == len(frames)-1 {
			kept = append(kept, fr)
			keptDelays = append(keptDelays, wait)
			wait = 0
		}
	}
	return kept, keptDelays
}

// frameAt is the frame to show after elapsed, and how long until the next one
// takes its place. A GIF plays again from its first frame.
func frameAt(elapsed time.Duration, delays []time.Duration) (int, time.Duration) {
	var total time.Duration
	for _, d := range delays {
		total += d
	}
	if total <= 0 || len(delays) == 0 {
		return 0, 0
	}
	t := elapsed % total
	if t < 0 {
		t = 0
	}
	var acc time.Duration
	for i, d := range delays {
		if t < acc+d {
			return i, acc + d - t
		}
		acc += d
	}
	return len(delays) - 1, total - t
}

// clearRect makes part of a picture transparent again.
func clearRect(img *image.RGBA, r image.Rectangle) {
	draw.Draw(img, r, image.Transparent, image.Point{}, draw.Src)
}

// cloneRGBA copies a picture.
func cloneRGBA(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Bounds())
	copy(out.Pix, img.Pix)
	return out
}
