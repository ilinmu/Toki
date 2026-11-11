package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// skipDirs are folders a library scan leaves out: system folders, other
// applications and their caches.
var skipDirs = map[string]bool{
	"windows": true, "winnt": true, "temp": true, "program files": true,
	"program files (x86)": true, "originals": true, "node_modules": true,
	".git": true, ".svn": true, "appdata": true, "$recycle.bin": true,
	"system volume information": true, ".trash": true, ".thumbnails": true,
}

// scanStats reports what one scan of a folder did.
type scanStats struct {
	Added, Updated, Skipped, Duplicates int
	Errors                              []string
}

// photoFromFile reads one image's metadata without adding it to the library.
// It is how a file opened from the file manager becomes a photo the viewer
// can show at once.
func photoFromFile(path string) (*Photo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() || !isImagePath(path) {
		return nil, errors.New("不是支持的图片格式")
	}
	p := &Photo{
		Path:      path,
		Name:      filepath.Base(path),
		Root:      filepath.Dir(path),
		Size:      info.Size(),
		ModUnix:   info.ModTime().Unix(),
		TakenUnix: info.ModTime().Unix(),
	}
	exif := readEXIFFromFile(path)
	if !exif.DateTimeOriginal.IsZero() {
		p.TakenUnix = exif.DateTimeOriginal.Unix()
	}
	p.Orientation = exif.Orientation
	p.Make, p.Model = exif.Make, exif.Model
	p.ISO, p.FNumber = exif.ISO, exif.FNumber
	p.ExposureTime, p.FocalLength = exif.ExposureTime, exif.FocalLength
	p.HasGPS, p.Lat, p.Lon = exif.HasGPS, exif.Lat, exif.Lon
	if w, h, err := imageSize(path); err == nil {
		p.Width, p.Height = w, h
	}
	return p, nil
}

// scanFolder walks a watched folder and adds what the library does not know
// yet, skipping files it already has and pictures whose contents it has under
// another path. progress is called with the number of pictures looked at.
func (a *app) scanFolder(root string, progress func(done int)) scanStats {
	var st scanStats
	done := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			st.Errors = append(st.Errors, path)
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path == root {
				return nil
			}
			if strings.HasPrefix(name, ".") || skipDirs[strings.ToLower(name)] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !isImagePath(path) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		done++
		if progress != nil && done%25 == 0 {
			progress(done)
		}
		if old := a.lib.Get(path); old != nil && old.Size == info.Size() && old.ModUnix == info.ModTime().Unix() {
			st.Skipped++
			return nil
		}
		print, err := fingerprint(path)
		if err == nil && a.set.Dedupe {
			if other, ok := a.lib.PrintOwner(print); ok && other != path {
				st.Duplicates++
				return nil
			}
		}
		p := &Photo{
			Path:      path,
			Name:      name,
			Root:      root,
			Size:      info.Size(),
			ModUnix:   info.ModTime().Unix(),
			TakenUnix: info.ModTime().Unix(),
		}
		exif := readEXIFFromFile(path)
		if !exif.DateTimeOriginal.IsZero() {
			p.TakenUnix = exif.DateTimeOriginal.Unix()
		}
		p.Orientation = exif.Orientation
		p.Make, p.Model = exif.Make, exif.Model
		p.ISO, p.FNumber = exif.ISO, exif.FNumber
		p.ExposureTime, p.FocalLength = exif.ExposureTime, exif.FocalLength
		p.HasGPS, p.Lat, p.Lon = exif.HasGPS, exif.Lat, exif.Lon
		if w, h, err := imageSize(path); err == nil {
			p.Width, p.Height = w, h
		}
		if old := a.lib.Get(path); old != nil {
			st.Updated++
		} else {
			st.Added++
		}
		a.lib.Upsert(p)
		if print != "" {
			a.lib.RememberPrint(print, path)
		}
		return nil
	})
	if progress != nil {
		progress(done)
	}
	return st
}

// prune forgets photos whose files are gone.
func (a *app) prune() int {
	var gone []string
	for _, p := range a.lib.Snapshot() {
		if _, err := os.Stat(p.Path); err != nil {
			gone = append(gone, p.Path)
		}
	}
	if len(gone) > 0 {
		a.lib.Forget(gone)
	}
	return len(gone)
}

// imageSize reads a picture's size without decoding its pixels.
func imageSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := imageDecodeConfig(f, filepath.Ext(strings.ToLower(path)))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
