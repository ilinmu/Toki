package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// uiReady says whether a window is running, so a worker knows it may ask it
// to draw again.
var uiReady atomic.Bool

// thumbCache keeps thumbnails of photos on disk and their bitmaps in memory.
// Grids ask for one while they build; a photo whose thumbnail is not ready
// shows a placeholder and the cache tells the window to draw again once the
// thumbnail is made.
type thumbCache struct {
	dir    string
	notify func()

	mu          sync.Mutex
	mem         map[string]*ui.Bitmap
	latest      map[string]latestThumb // by path and stamp
	order       []string               // least recently used first
	pending     map[string]bool
	jobs        chan thumbJob
	prewarmJobs chan thumbJob
	limit       int
	preparing   atomic.Int64
	prewarmDone atomic.Int64
}

// thumbJob asks for one thumbnail: either to make it from the original, or
// only to load the one already on disk into a bitmap.
type thumbJob struct {
	key      string
	path     string
	mod      int64
	size     int
	edit     EditState
	loadOnly bool // read one already on disk instead of rendering the original
	diskOnly bool // prewarm: only write the file, skip the bitmap
	prewarm  bool // counted for PrewarmPending
}

// latestThumb is the last bitmap a photo got, whatever size it was made at:
// a view that needs a sharper one shows this until it arrives.
type latestThumb struct {
	bitmap *ui.Bitmap
	size   int
}

// newThumbCache makes a cache storing its files in dir, calling notify when
// a thumbnail arrives.
func newThumbCache(dir string, notify func()) *thumbCache {
	os.MkdirAll(dir, 0o755)
	t := &thumbCache{
		dir:         dir,
		notify:      notify,
		mem:         map[string]*ui.Bitmap{},
		latest:      map[string]latestThumb{},
		pending:     map[string]bool{},
		jobs:        make(chan thumbJob, 512),
		prewarmJobs: make(chan thumbJob, 512),
		limit:       400,
	}
	workers := max(2, min(4, runtime.NumCPU()-1))
	for i := 0; i < workers; i++ {
		go t.work()
	}
	return t
}

// key identifies a thumbnail: the file, its stamp, the size asked for and the
// edit applied to it.
func thumbKey(path string, mod, size int64, st EditState) string {
	edit, _ := json.Marshal(st)
	h := sha1.New()
	h.Write([]byte(path))
	h.Write([]byte(itoa(int(mod))))
	h.Write([]byte(itoa(int(size))))
	h.Write(edit)
	return hex.EncodeToString(h.Sum(nil))[:20]
}

// Bitmap returns the thumbnail of a photo at size pixels, and whether it is
// ready. The caller draws a placeholder when it is not. Nothing is read or
// decoded on the calling (UI) thread: a thumbnail on disk is loaded by a
// worker, so scrolling never blocks on file I/O or JPEG decoding.
func (t *thumbCache) Bitmap(path string, mod int64, size int, st EditState) (*ui.Bitmap, bool) {
	key := thumbKey(path, mod, int64(size), st)
	t.mu.Lock()
	if b, ok := t.mem[key]; ok {
		t.touchLocked(key)
		t.mu.Unlock()
		return b, true
	}
	waiting := t.pending[key]
	t.mu.Unlock()
	// Something this photo showed before, at another size, is better than a
	// placeholder while its exact size loads or is being prepared.
	if b, ok := t.Latest(path, mod); ok {
		if !waiting {
			job := thumbJob{key: key, path: path, mod: mod, size: size, edit: st}
			if t.onDisk(key) {
				job.loadOnly = true
			}
			t.requestJob(job)
		}
		return b, true
	}
	if waiting {
		return nil, false
	}
	// A windowless run, as in the screenshot tests, reads the disk here: there
	// is no main loop to hand the bitmap to.
	if !uiReady.Load() {
		if data, err := os.ReadFile(t.file(key)); err == nil {
			if b, err := ui.DecodeBitmap(data); err == nil {
				t.store(key, b)
				t.remember(path, mod, size, b)
				return b, true
			}
		}
		return nil, false
	}
	job := thumbJob{key: key, path: path, mod: mod, size: size, edit: st}
	if t.onDisk(key) {
		job.loadOnly = true
	}
	t.requestJob(job)
	return nil, false
}

// Request asks for a thumbnail without reading or decoding anything on the
// calling thread: a large preview, which the window shows once the worker
// delivers it.
func (t *thumbCache) Request(path string, mod int64, size int, st EditState) {
	key := thumbKey(path, mod, int64(size), st)
	t.mu.Lock()
	_, inMemory := t.mem[key]
	t.mu.Unlock()
	if inMemory {
		return
	}
	job := thumbJob{key: key, path: path, mod: mod, size: size, edit: st}
	if t.onDisk(key) {
		job.loadOnly = true
	}
	t.requestJob(job)
}

// Prewarm makes the thumbnail for a photo in the background, for a size the
// views will ask for soon. It skips what is already cached, and never runs on
// the UI thread: a large import is prepared while the user can keep working.
func (t *thumbCache) Prewarm(path string, mod int64, size int, st EditState, wantBitmap bool) bool {
	if !uiReady.Load() {
		return false
	}
	key := thumbKey(path, mod, int64(size), st)
	t.mu.Lock()
	_, inMemory := t.mem[key]
	if inMemory || t.pending[key] {
		t.mu.Unlock()
		return false
	}
	onDisk := t.onDisk(key)
	if !wantBitmap && onDisk {
		t.mu.Unlock()
		return false
	}
	t.pending[key] = true
	t.mu.Unlock()
	job := thumbJob{key: key, path: path, mod: mod, size: size, edit: st,
		diskOnly: !wantBitmap, loadOnly: wantBitmap && onDisk, prewarm: true}
	t.preparing.Add(1)
	t.prewarmJobs <- job
	return true
}

// SetLimitFor sizes the decoded-bitmap cache for a thumbnail size, within a
// memory budget. The current grid size is kept in memory for the whole
// library when it fits; other sizes can still be read from disk.
func (t *thumbCache) SetLimitFor(pixels int) {
	if pixels <= 0 {
		return
	}
	const budget = 384 << 20 // bytes of decoded pixels
	n := budget / (pixels * pixels * 4)
	n = min(max(n, 300), 1500)
	t.mu.Lock()
	t.limit = n
	t.mu.Unlock()
}

// PrewarmPending is how many prewarm thumbnails are still queued or being
// made.
func (t *thumbCache) PrewarmPending() int64 { return t.preparing.Load() }

// PrewarmDone is how many prewarm thumbnails have been written so far. The
// caller takes a baseline before it queues a batch.
func (t *thumbCache) PrewarmDone() int64 { return t.prewarmDone.Load() }

// onDisk reports whether a thumbnail of this key was made before.
func (t *thumbCache) onDisk(key string) bool {
	_, err := os.Stat(t.file(key))
	return err == nil
}

// requestJob queues a job unless the same one is queued already.
func (t *thumbCache) requestJob(job thumbJob) {
	if !uiReady.Load() {
		// No window, as in tests: nothing would show the result, and a
		// worker writing into a temporary folder outlives the test.
		return
	}
	t.mu.Lock()
	if t.pending[job.key] {
		t.mu.Unlock()
		return
	}
	t.pending[job.key] = true
	t.mu.Unlock()
	select {
	case t.jobs <- job:
	default:
		// The queue is full: let a later frame ask again.
		t.mu.Lock()
		delete(t.pending, job.key)
		t.mu.Unlock()
	}
}

// Latest returns the last bitmap a photo got at any size, so a view needing
// a sharper one can show something in the meantime.
func (t *thumbCache) Latest(path string, mod int64) (*ui.Bitmap, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.latest[latestKey(path, mod)]
	if !ok {
		return nil, false
	}
	return entry.bitmap, true
}

// latestKey names the last bitmap of a photo.
func latestKey(path string, mod int64) string { return path + "|" + itoa(int(mod)) }

// remember keeps a bitmap as the photo's last one, when it is larger than
// what the photo showed before.
func (t *thumbCache) remember(path string, mod int64, size int, b *ui.Bitmap) {
	if path == "" || b == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	key := latestKey(path, mod)
	if old, ok := t.latest[key]; ok && old.size >= size {
		return
	}
	t.latest[key] = latestThumb{bitmap: b, size: size}
	if len(t.latest) > t.limit*2 {
		for k := range t.latest {
			if _, ok := t.latest[k]; ok && k != key {
				delete(t.latest, k)
				break
			}
		}
	}
}

// Peek returns a thumbnail only when it is already in memory, without asking
// for one: a view that wants a sharper picture than the one it has asks here
// first, and keeps showing what it has until the sharper one arrives.
func (t *thumbCache) Peek(path string, mod int64, size int, st EditState) (*ui.Bitmap, bool) {
	key := thumbKey(path, mod, int64(size), st)
	t.mu.Lock()
	defer t.mu.Unlock()
	b, ok := t.mem[key]
	if ok {
		t.touchLocked(key)
	}
	return b, ok
}

// file is where a thumbnail is stored on disk.
func (t *thumbCache) file(key string) string { return filepath.Join(t.dir, key+".jpg") }

// work makes thumbnails until the app quits.
func (t *thumbCache) work() {
	for {
		// What a view asks for goes before what prewarm queued.
		select {
		case job := <-t.jobs:
			t.make(job)
			continue
		default:
		}
		select {
		case job := <-t.jobs:
			t.make(job)
		case job := <-t.prewarmJobs:
			t.make(job)
		}
	}
}

// make loads one thumbnail. A thumbnail already on disk is only read and
// decoded; one that is missing is decoded from the original, edited, scaled
// and written back. The bitmap itself is made on the main thread, as the
// window requires.
func (t *thumbCache) make(job thumbJob) {
	defer func() {
		t.mu.Lock()
		delete(t.pending, job.key)
		t.mu.Unlock()
		if job.prewarm {
			t.preparing.Add(-1)
			t.prewarmDone.Add(1)
		}
	}()
	if job.loadOnly {
		if !uiReady.Load() {
			return
		}
		data, err := os.ReadFile(t.file(job.key))
		if err != nil {
			return
		}
		mygo.RunOnMain(func() {
			bmp, err := ui.DecodeBitmap(data)
			if err != nil {
				return
			}
			t.store(job.key, bmp)
			t.remember(job.path, job.mod, job.size, bmp)
			if t.notify != nil {
				t.notify()
			}
		})
		return
	}
	img, err := decodeImage(job.path)
	if err != nil {
		return
	}
	info := readEXIFFromFile(job.path)
	img = applyOrientation(img, info.Orientation)
	img = renderEdit(img, job.edit)
	b := img.Bounds()
	w, h := fitSize(b.Dx(), b.Dy(), job.size, job.size)
	small := resize(img, w, h)
	if err := encodeJPEGFile(t.file(job.key), small, 82); err != nil {
		return
	}
	if job.diskOnly {
		return
	}
	if !uiReady.Load() {
		return // no window yet, as in tests: the disk cache is enough
	}
	// The bitmap has to be made where the window runs.
	mygo.RunOnMain(func() {
		data, err := os.ReadFile(t.file(job.key))
		if err != nil {
			return
		}
		if bmp, err := ui.DecodeBitmap(data); err == nil {
			t.store(job.key, bmp)
			t.remember(job.path, job.mod, job.size, bmp)
			if t.notify != nil {
				t.notify()
			}
		}
	})
}

// store keeps a bitmap, dropping the least recently used ones beyond the cap.
func (t *thumbCache) store(key string, b *ui.Bitmap) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if old, ok := t.mem[key]; ok && old != b {
		old.ReleasePixels()
	}
	t.mem[key] = b
	t.touchLocked(key)
	for len(t.order) > t.limit {
		oldest := t.order[0]
		t.order = t.order[1:]
		if old, ok := t.mem[oldest]; ok {
			old.ReleasePixels()
			delete(t.mem, oldest)
		}
	}
}

// touchLocked moves key to the most recently used end.
func (t *thumbCache) touchLocked(key string) {
	for i, k := range t.order {
		if k == key {
			t.order = append(t.order[:i], t.order[i+1:]...)
			break
		}
	}
	t.order = append(t.order, key)
}

// forget drops every cached thumbnail of a photo, as after an edit.
func (t *thumbCache) forget(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for key, b := range t.mem {
		_ = key
		b.ReleasePixels()
	}
	t.mem = map[string]*ui.Bitmap{}
	t.order = nil
}

// clearDisk empties the thumbnail folder.
func (t *thumbCache) clearDisk() error {
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		os.Remove(filepath.Join(t.dir, e.Name()))
	}
	t.forget("")
	return nil
}

// size returns how many bytes the thumbnails take on disk.
func (t *thumbCache) size() int64 {
	var total int64
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		if info, err := e.Info(); err == nil {
			total += info.Size()
		}
	}
	return total
}

// readEXIFFromFile reads the metadata of a photo, quietly. Only the start of
// the file is read: a camera writes EXIF before the picture's own data, so a
// 30 MB original costs a few kilobytes here.
func readEXIFFromFile(path string) exifInfo {
	f, err := os.Open(path)
	if err != nil {
		return exifInfo{}
	}
	defer f.Close()
	head := make([]byte, 512<<10)
	n, err := io.ReadFull(f, head)
	if n == 0 {
		return exifInfo{}
	}
	_ = err // a short read is a file shorter than the head, which is fine
	return readEXIF(head[:n])
}
