package main

import "github.com/egoist/mygo/ui"

// The line icons of the prototype: 24×24, one color, 1.7 wide strokes with
// round caps and joins, filled shapes for the states that are on (a star, a
// heart). ui.Icon draws them in the color of the text they sit in, so one
// icon serves every state.
var icons = struct {
	search, plus, importFile, image, clock, heart, heartFilled      *ui.SVG
	albums, folder, sidebar, info                                   *ui.SVG
	chevronLeft, chevronRight, arrowLeft                            *ui.SVG
	star, starFilled, edit, sliders, rotateLeft, rotateRight        *ui.SVG
	undo, redo, check, x, play, pause, expand, download, trash, tag *ui.SVG
	grid, masonry, moon, sun, camera                                *ui.SVG
	gear                                                            *ui.SVG
	minus                                                           *ui.SVG
}{
	search:       svgIcon(`<circle cx="11" cy="11" r="6.5"></circle><path d="m16 16 4.2 4.2"></path>`),
	plus:         svgIcon(`<path d="M12 5v14M5 12h14"></path>`),
	importFile:   svgIcon(`<path d="M12 3v11m0 0 4-4m-4 4-4-4"></path><path d="M5 16.5V19a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-2.5"></path>`),
	image:        svgIcon(`<rect x="3.5" y="4.5" width="17" height="15" rx="2.5"></rect><circle cx="9" cy="10" r="1.5"></circle><path d="m5.5 17 4.6-4.4 3.2 3 2.5-2.2 2.7 3.6"></path>`),
	clock:        svgIcon(`<circle cx="12" cy="12" r="8.5"></circle><path d="M12 7.5V12l3 2"></path>`),
	heart:        svgIcon(`<path d="M20.2 5.7a5 5 0 0 0-7.1 0L12 6.8l-1.1-1.1a5 5 0 0 0-7.1 7.1L12 21l8.2-8.2a5 5 0 0 0 0-7.1Z"></path>`),
	heartFilled:  svgIcon(`<path d="M20.2 5.7a5 5 0 0 0-7.1 0L12 6.8l-1.1-1.1a5 5 0 0 0-7.1 7.1L12 21l8.2-8.2a5 5 0 0 0 0-7.1Z" fill="currentColor"></path>`),
	albums:       svgIcon(`<path d="M5 4.5h11.5A2.5 2.5 0 0 1 19 7v12.5H7.5A2.5 2.5 0 0 1 5 17V4.5Z"></path><path d="M8 8h7M8 11.5h7M8 15h4.5"></path>`),
	folder:       svgIcon(`<path d="M3.5 7.5A2.5 2.5 0 0 1 6 5h4l2 2h6.5A2.5 2.5 0 0 1 21 9.5v8A2.5 2.5 0 0 1 18.5 20h-13A2.5 2.5 0 0 1 3 17.5v-10Z"></path>`),
	sidebar:      svgIcon(`<rect x="3.5" y="4.5" width="17" height="15" rx="2.5"></rect><path d="M15 4.5v15M8 9h3.5M8 12h3.5M8 15h3.5"></path>`),
	info:         svgIcon(`<circle cx="12" cy="12" r="9"></circle><path d="M12 11v6M12 7.3v.2"></path>`),
	chevronLeft:  svgIcon(`<path d="m14.5 6-6 6 6 6"></path>`),
	chevronRight: svgIcon(`<path d="m9.5 6 6 6-6 6"></path>`),
	arrowLeft:    svgIcon(`<path d="m14.5 6-6 6 6 6M8.5 12H20"></path>`),
	star:         svgIcon(`<path d="m12 3.3 2.7 5.5 6.1.9-4.4 4.3 1 6-5.4-2.8-5.4 2.8 1-6-4.4-4.3 6.1-.9L12 3.3Z"></path>`),
	starFilled:   svgIcon(`<path d="m12 3.3 2.7 5.5 6.1.9-4.4 4.3 1 6-5.4-2.8-5.4 2.8 1-6-4.4-4.3 6.1-.9L12 3.3Z" fill="currentColor"></path>`),
	edit:         svgIcon(`<path d="m4 20 4.2-1 10.7-10.7a2.3 2.3 0 0 0-3.3-3.3L4.9 15.8 4 20Z"></path><path d="m14.3 6.3 3.4 3.4"></path>`),
	sliders:      svgIcon(`<path d="M4 7h7M15 7h5M4 12h3M11 12h9M4 17h9M17 17h3"></path><circle cx="13" cy="7" r="2"></circle><circle cx="9" cy="12" r="2"></circle><circle cx="15" cy="17" r="2"></circle>`),
	rotateLeft:   svgIcon(`<path d="M4 10a8 8 0 1 1 2.3 6.7M4 4v6h6"></path>`),
	rotateRight:  svgIcon(`<path d="M20 10a8 8 0 1 0-2.3 6.7M20 4v6h-6"></path>`),
	undo:         svgIcon(`<path d="M9 7 4 12l5 5"></path><path d="M5 12h8.5a5.5 5.5 0 0 1 0 11H11"></path>`),
	redo:         svgIcon(`<path d="m15 7 5 5-5 5"></path><path d="M19 12h-8.5a5.5 5.5 0 0 0 0 11H13"></path>`),
	check:        svgIcon(`<path d="m5 12.5 4.2 4.2L19 7"></path>`),
	x:            svgIcon(`<path d="m6 6 12 12M18 6 6 18"></path>`),
	play:         svgIcon(`<path d="m8 5 11 7-11 7V5Z"></path>`),
	pause:        svgIcon(`<path d="M8 5v14M16 5v14"></path>`),
	expand:       svgIcon(`<path d="M8 3H3v5M16 3h5v5M21 16v5h-5M3 16v5h5"></path>`),
	download:     svgIcon(`<path d="M12 3v11m0 0 4-4m-4 4-4-4M4 19h16"></path>`),
	trash:        svgIcon(`<path d="M4 7h16M9 7V4h6v3M6.5 7l.8 13h9.4l.8-13M10 11v5M14 11v5"></path>`),
	tag:          svgIcon(`<path d="m20 13-7 7L4 11V4h7l9 9Z"></path><circle cx="8.5" cy="8.5" r="1.2"></circle>`),
	grid:         svgIcon(`<rect x="4" y="4" width="6" height="6" rx="1"></rect><rect x="14" y="4" width="6" height="6" rx="1"></rect><rect x="4" y="14" width="6" height="6" rx="1"></rect><rect x="14" y="14" width="6" height="6" rx="1"></rect>`),
	masonry:      svgIcon(`<rect x="4" y="4" width="6" height="9" rx="1"></rect><rect x="14" y="4" width="6" height="5" rx="1"></rect><rect x="14" y="13" width="6" height="7" rx="1"></rect><rect x="4" y="17" width="6" height="3" rx="1"></rect>`),
	moon:         svgIcon(`<path d="M20 15.2A8 8 0 0 1 8.8 4 8 8 0 1 0 20 15.2Z"></path>`),
	sun:          svgIcon(`<circle cx="12" cy="12" r="4"></circle><path d="M12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.3 5.3l1.4 1.4M17.3 17.3l1.4 1.4M18.7 5.3l-1.4 1.4M6.7 17.3l-1.4 1.4"></path>`),
	camera:       svgIcon(`<path d="M4 8.5h3l1.5-2h7l1.5 2h3v10H4v-10Z"></path><circle cx="12" cy="13" r="3.2"></circle>`),
	minus:        svgIcon(`<path d="M5 12h14"></path>`),
	gear:         svgIcon(`<path d="M12.9 2.6h-1.8a1.8 1.8 0 0 0-1.8 1.8v.2a1.8 1.8 0 0 1-.9 1.6l-.4.2a1.8 1.8 0 0 1-1.8 0l-.2-.1a1.8 1.8 0 0 0-2.5.7l-.9 1.6a1.8 1.8 0 0 0 .7 2.4l.2.1a1.8 1.8 0 0 1 .9 1.6v.4a1.8 1.8 0 0 1-.9 1.6l-.2.1a1.8 1.8 0 0 0-.7 2.4l.9 1.6a1.8 1.8 0 0 0 2.5.7l.2-.1a1.8 1.8 0 0 1 1.8 0l.4.2a1.8 1.8 0 0 1 .9 1.6v.2a1.8 1.8 0 0 0 1.8 1.8h1.8a1.8 1.8 0 0 0 1.8-1.8v-.2a1.8 1.8 0 0 1 .9-1.6l.4-.2a1.8 1.8 0 0 1 1.8 0l.2.1a1.8 1.8 0 0 0 2.5-.7l.9-1.6a1.8 1.8 0 0 0-.7-2.4l-.2-.1a1.8 1.8 0 0 1-.9-1.6v-.4a1.8 1.8 0 0 1 .9-1.6l.2-.1a1.8 1.8 0 0 0 .7-2.4l-.9-1.6a1.8 1.8 0 0 0-2.5-.7l-.2.1a1.8 1.8 0 0 1-1.8 0l-.4-.2a1.8 1.8 0 0 1-.9-1.6v-.2a1.8 1.8 0 0 0-1.8-1.8Z"></path><circle cx="12" cy="12" r="2.9"></circle>`),
}

// brandMark is the window's mark: a lens, a pinwheel and a shutter, drawn
// with the same 1.7 stroke as the icons.
var brandMark = svgIcon(`<circle cx="12" cy="12" r="8.4"></circle><path d="M12 3.6 16.5 12 12 20.4 7.5 12Z"></path><circle cx="12" cy="12" r="2.4"></circle>`)

// svgIcon parses one icon of the set.
func svgIcon(body string) *ui.SVG {
	const head = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" ` +
		`stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">`
	return ui.MustParseSVG([]byte(head + body + `</svg>`))
}
