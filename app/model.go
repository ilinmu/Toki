package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// EditState is one photo's non-destructive edit: a fixed set of adjustments
// plus an ordered list of effects. The original file is never touched.
type EditState struct {
	Rotate        int      `json:"rotate,omitempty"` // degrees clockwise, 0/90/180/270
	FlipH         bool     `json:"fliph,omitempty"`
	FlipV         bool     `json:"flipv,omitempty"`
	Angle         float64  `json:"angle,omitempty"` // straighten, degrees
	HasCrop       bool     `json:"hascrop,omitempty"`
	CropX         float64  `json:"cropx,omitempty"`    // 0..1, left
	CropY         float64  `json:"cropy,omitempty"`    // 0..1, top
	CropW         float64  `json:"cropw,omitempty"`    // 0..1, width
	CropH         float64  `json:"croph,omitempty"`    // 0..1, height
	Exposure      float64  `json:"exposure,omitempty"` // -1..1
	Vignette      float64  `json:"vignette,omitempty"` // 0..1
	FillLight     float64  `json:"fill,omitempty"`     // -1..1
	Highlights    float64  `json:"highlights,omitempty"`
	Shadows       float64  `json:"shadows,omitempty"`
	Temp          float64  `json:"temp,omitempty"` // -1 cool .. 1 warm
	Saturation    float64  `json:"sat,omitempty"`
	Contrast      float64  `json:"contrast,omitempty"`
	Sharpen       float64  `json:"sharpen,omitempty"` // 0..1
	Soften        float64  `json:"soften,omitempty"`  // 0..1
	LocalContrast float64  `json:"local,omitempty"`   // 0..1
	Effects       []string `json:"effects,omitempty"`
}

// Clone copies the state, including its effect list.
func (e EditState) Clone() EditState {
	out := e
	out.Effects = append([]string(nil), e.Effects...)
	return out
}

// IsZero reports whether the photo has no edit at all.
func (e EditState) IsZero() bool {
	return e.Rotate == 0 && !e.FlipH && !e.FlipV && e.Angle == 0 && !e.HasCrop &&
		e.Exposure == 0 && e.Vignette == 0 &&
		e.FillLight == 0 && e.Highlights == 0 && e.Shadows == 0 && e.Temp == 0 &&
		e.Saturation == 0 && e.Contrast == 0 && e.Sharpen == 0 && e.Soften == 0 &&
		e.LocalContrast == 0 && len(e.Effects) == 0
}

// Photo is one image in the library, with its metadata and its edit.
type Photo struct {
	Path        string      `json:"path"`
	Name        string      `json:"name"`
	Root        string      `json:"root"`
	Size        int64       `json:"size"`
	ModUnix     int64       `json:"mod"`
	TakenUnix   int64       `json:"taken"`
	Width       int         `json:"w,omitempty"`
	Height      int         `json:"h,omitempty"`
	Orientation int         `json:"orientation,omitempty"`
	Stars       int         `json:"stars,omitempty"`
	Favorite    bool        `json:"favorite,omitempty"`
	Caption     string      `json:"caption,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Albums      []string    `json:"albums,omitempty"`
	Edit        EditState   `json:"edit"`
	Undo        []EditState `json:"undo,omitempty"`
	Redo        []EditState `json:"redo,omitempty"`

	// EXIF, as a photo's properties are usually listed.
	Make         string  `json:"make,omitempty"`
	Model        string  `json:"model,omitempty"`
	ISO          int     `json:"iso,omitempty"`
	FNumber      float64 `json:"fnumber,omitempty"`
	ExposureTime string  `json:"exposure,omitempty"`
	FocalLength  int     `json:"focal,omitempty"`
	HasGPS       bool    `json:"hasgps,omitempty"`
	Lat          float64 `json:"lat,omitempty"`
	Lon          float64 `json:"lon,omitempty"`
}

// Taken returns the date the photo was taken, or the file's modification
// time when the camera gave none.
func (p *Photo) Taken() time.Time {
	if p.TakenUnix > 0 {
		return time.Unix(p.TakenUnix, 0)
	}
	return time.Unix(p.ModUnix, 0)
}

// Year returns the year the photo was taken in.
func (p *Photo) Year() int { return p.Taken().Year() }

// HasTag reports whether the photo carries tag.
func (p *Photo) HasTag(tag string) bool {
	for _, t := range p.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

// InAlbum reports whether the photo is in the album with id.
func (p *Photo) InAlbum(id string) bool {
	for _, a := range p.Albums {
		if a == id {
			return true
		}
	}
	return false
}

// Album is a named set of photos. A starred album follows the star marks
// instead of holding photos of its own.
type Album struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Tint        string `json:"tint,omitempty"`
	Starred     bool   `json:"starred,omitempty"`
	Created     int64  `json:"created"`
}

// Library is the whole photo library: watched folders, photos, albums and
// the tag vocabulary, saved as one JSON file.
type Library struct {
	Folders []string `json:"folders"`
	Photos  []*Photo `json:"photos"`
	Albums  []*Album `json:"albums"`
	Tags    []string `json:"tags"`
	// Prints maps a picture's content fingerprint to the path that has it,
	// so a scan can skip a picture the library already has elsewhere.
	Prints map[string]string `json:"prints,omitempty"`

	mu     sync.Mutex
	path   string
	byPath map[string]*Photo
}

// NewLibrary makes an empty library saved at path.
func NewLibrary(path string) *Library {
	return &Library{path: path, byPath: map[string]*Photo{}, Prints: map[string]string{}}
}

// PrintOwner returns the path that holds a picture with this fingerprint.
func (l *Library) PrintOwner(print string) (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.Prints[print]
	return p, ok
}

// RememberPrint records which path holds a picture with this fingerprint.
func (l *Library) RememberPrint(print, path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.Prints == nil {
		l.Prints = map[string]string{}
	}
	l.Prints[print] = path
}

// Load reads the library from disk, if it has been saved before.
func (l *Library) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := json.Unmarshal(b, l); err != nil {
		return err
	}
	l.byPath = make(map[string]*Photo, len(l.Photos))
	for _, p := range l.Photos {
		l.byPath[p.Path] = p
	}
	if l.Prints == nil {
		l.Prints = map[string]string{}
	}
	return nil
}

// Save writes the library, replacing the file atomically.
func (l *Library) Save() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.saveLocked()
}

func (l *Library) saveLocked() error {
	if l.path == "" {
		return nil
	}
	b, err := json.Marshal(l)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, l.path)
}

// AddFolder adds a watched folder if it is not watched yet.
func (l *Library) AddFolder(dir string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, f := range l.Folders {
		if f == dir {
			return false
		}
	}
	l.Folders = append(l.Folders, dir)
	return true
}

// RemoveFolder stops watching a folder and forgets the photos under it.
func (l *Library) RemoveFolder(dir string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.Folders[:0]
	for _, f := range l.Folders {
		if f != dir {
			kept = append(kept, f)
		}
	}
	l.Folders = kept
	photos := l.Photos[:0]
	for _, p := range l.Photos {
		if p.Root == dir {
			delete(l.byPath, p.Path)
			continue
		}
		photos = append(photos, p)
	}
	l.Photos = photos
}

// Upsert adds or refreshes a photo by path, keeping its stars, tags, albums
// and edit.
func (l *Library) Upsert(p *Photo) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if old, ok := l.byPath[p.Path]; ok {
		old.Size, old.ModUnix = p.Size, p.ModUnix
		if p.TakenUnix > 0 {
			old.TakenUnix = p.TakenUnix
		}
		old.Width, old.Height, old.Orientation = p.Width, p.Height, p.Orientation
		if p.Make != "" {
			old.Make, old.Model = p.Make, p.Model
		}
		if p.ISO > 0 {
			old.ISO, old.FNumber = p.ISO, p.FNumber
			old.ExposureTime, old.FocalLength = p.ExposureTime, p.FocalLength
		}
		if p.HasGPS {
			old.HasGPS, old.Lat, old.Lon = true, p.Lat, p.Lon
		}
		return true
	}
	l.byPath[p.Path] = p
	l.Photos = append(l.Photos, p)
	return true
}

// Forget removes photos from the library. The files stay on disk.
func (l *Library) Forget(paths []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	drop := make(map[string]bool, len(paths))
	for _, p := range paths {
		drop[p] = true
		delete(l.byPath, p)
	}
	kept := l.Photos[:0]
	for _, p := range l.Photos {
		if !drop[p.Path] {
			kept = append(kept, p)
		}
	}
	l.Photos = kept
}

// Get returns the photo at path, or nil.
func (l *Library) Get(path string) *Photo {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.byPath[path]
}

// Has reports whether the library knows a photo at path.
func (l *Library) Has(path string) bool { return l.Get(path) != nil }

// Snapshot returns the photos newest first, a copy the caller may sort.
func (l *Library) Snapshot() []*Photo {
	l.mu.Lock()
	out := make([]*Photo, len(l.Photos))
	copy(out, l.Photos)
	l.mu.Unlock()
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TakenUnix != out[j].TakenUnix {
			return out[i].TakenUnix > out[j].TakenUnix
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// FolderList returns the watched folders, sorted.
func (l *Library) FolderList() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := append([]string(nil), l.Folders...)
	sort.Strings(out)
	return out
}

// AlbumList returns the albums in creation order.
func (l *Library) AlbumList() []*Album {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]*Album(nil), l.Albums...)
}

// Album returns the album with id, or nil.
func (l *Library) Album(id string) *Album {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, a := range l.Albums {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// CreateAlbum adds an album named name and returns it, with the tint that
// follows the ones the library already has.
func (l *Library) CreateAlbum(name string) *Album {
	return l.CreateAlbumWith(name, "")
}

// CreateAlbumWith adds an album with a description, and the tint named tint,
// or the next of the six when that is empty.
func (l *Library) CreateAlbumWith(name, description string) *Album {
	l.mu.Lock()
	defer l.mu.Unlock()
	tint := albumTintNames[len(l.Albums)%len(albumTintNames)]
	a := &Album{
		ID:          "a" + itoa(len(l.Albums)+1) + "-" + strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Name:        name,
		Description: description,
		Tint:        tint,
		Created:     time.Now().Unix(),
	}
	l.Albums = append(l.Albums, a)
	return a
}

// DeleteAlbum removes an album and takes it out of every photo.
func (l *Library) DeleteAlbum(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.Albums[:0]
	for _, a := range l.Albums {
		if a.ID != id {
			kept = append(kept, a)
		}
	}
	l.Albums = kept
	for _, p := range l.Photos {
		p.Albums = without(p.Albums, id)
	}
}

// AddToAlbum puts photos in an album.
func (l *Library) AddToAlbum(albumID string, photos []*Photo) {
	for _, p := range photos {
		if !p.InAlbum(albumID) {
			p.Albums = append(p.Albums, albumID)
		}
	}
}

// TagVocabulary returns the known tags, sorted.
func (l *Library) TagVocabulary() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := append([]string(nil), l.Tags...)
	sort.Strings(out)
	return out
}

// LearnTag remembers a tag in the vocabulary.
func (l *Library) LearnTag(tag string) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, t := range l.Tags {
		if strings.EqualFold(t, tag) {
			return
		}
	}
	l.Tags = append(l.Tags, tag)
	sort.Strings(l.Tags)
}

// DeleteTag removes a tag from every photo and from the learned vocabulary.
// The photos themselves stay in the library.
func (l *Library) DeleteTag(tag string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	keptTags := l.Tags[:0]
	for _, t := range l.Tags {
		if !strings.EqualFold(t, tag) {
			keptTags = append(keptTags, t)
		}
	}
	l.Tags = keptTags
	for _, p := range l.Photos {
		kept := p.Tags[:0]
		for _, t := range p.Tags {
			if !strings.EqualFold(t, tag) {
				kept = append(kept, t)
			}
		}
		p.Tags = kept
	}
}

// AllTags returns every tag in use, learned or not.
func (l *Library) AllTags() []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range l.Snapshot() {
		for _, t := range p.Tags {
			if !seen[strings.ToLower(t)] {
				seen[strings.ToLower(t)] = true
				out = append(out, t)
			}
		}
	}
	for _, t := range l.TagVocabulary() {
		if !seen[strings.ToLower(t)] {
			seen[strings.ToLower(t)] = true
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}

// without returns list without value.
func without(list []string, value string) []string {
	out := list[:0]
	for _, s := range list {
		if s != value {
			out = append(out, s)
		}
	}
	return out
}

// itoa is a small helper so the model file keeps its imports minimal.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
