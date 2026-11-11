package main

import (
	"sort"
	"strings"
	"time"
)

// albumRow, tagRow, folderRow and yearRow are the sidebar's places, with the
// number of photos each holds. They are counted once per library change, not
// once per frame: a library of tens of thousands of photos must not make the
// sidebar walk every photo on every redraw.
type albumRow struct {
	id, name, description, tint string
	count                       int
}

type tagRow struct {
	name  string
	count int
}

type folderRow struct {
	path  string
	count int
}

type yearRow struct {
	year  string
	count int
}

// libraryData is one immutable view of the library. All per-frame readers
// share it; a change to the library or a rebuild of the shown photos gives
// the app a new revision and the next frame builds a fresh one.
type libraryData struct {
	rev     uint64
	all     []*Photo // by date, newest first
	byPath  map[string]*Photo
	counts  libCounts
	albums  []albumRow
	tags    []tagRow
	folders []folderRow
	years   []yearRow
}

// library returns the cached view of the whole library. The cache is owned by
// the main thread; a background scan only advances the atomic revision.
func (a *app) library() *libraryData {
	rev := a.dataRev.Load()
	if a.libData != nil && a.libData.rev == rev {
		return a.libData
	}
	d := buildLibraryData(a.lib, rev)
	a.libData = d
	return d
}

func buildLibraryData(lib *Library, rev uint64) *libraryData {
	all := lib.Snapshot() // one sort per revision
	d := &libraryData{
		rev:    rev,
		all:    all,
		byPath: make(map[string]*Photo, len(all)),
	}

	albumCounts := make(map[string]int)
	tagCounts := make(map[string]int)
	tagNames := make(map[string]string)
	folderCounts := make(map[string]int)
	yearCounts := make(map[string]int)
	cutoff := time.Now().Unix() - 30*24*3600

	for _, p := range all {
		d.byPath[p.Path] = p
		d.counts.all++
		d.counts.bytes += p.Size
		if p.TakenUnix >= cutoff {
			d.counts.recent++
		}
		if p.Favorite {
			d.counts.favorites++
		}
		if p.Stars > 0 {
			d.counts.stars++
		}
		if !p.Edit.IsZero() {
			d.counts.edited++
		}
		for _, id := range p.Albums {
			albumCounts[id]++
		}
		for _, t := range p.Tags {
			key := strings.ToLower(t)
			if _, ok := tagNames[key]; !ok {
				tagNames[key] = t
			}
			tagCounts[key]++
		}
		folderCounts[p.Root]++
		yearCounts[itoa(p.Year())]++
	}

	for i, al := range lib.AlbumList() {
		tint := al.Tint
		if tint == "" {
			tint = albumTintNames[i%len(albumTintNames)]
		}
		d.albums = append(d.albums, albumRow{
			id: al.ID, name: al.Name, description: al.Description,
			tint: tint, count: albumCounts[al.ID],
		})
	}
	for _, t := range lib.TagVocabulary() {
		key := strings.ToLower(t)
		if _, ok := tagNames[key]; !ok {
			tagNames[key] = t
		}
	}
	tagKeys := make([]string, 0, len(tagNames))
	for key := range tagNames {
		tagKeys = append(tagKeys, key)
	}
	sort.Strings(tagKeys)
	for _, key := range tagKeys {
		d.tags = append(d.tags, tagRow{name: tagNames[key], count: tagCounts[key]})
	}
	for _, dir := range lib.FolderList() {
		d.folders = append(d.folders, folderRow{path: dir, count: folderCounts[dir]})
	}
	yearKeys := make([]string, 0, len(yearCounts))
	for y := range yearCounts {
		yearKeys = append(yearKeys, y)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(yearKeys)))
	for _, y := range yearKeys {
		d.years = append(d.years, yearRow{year: y, count: yearCounts[y]})
	}
	return d
}
