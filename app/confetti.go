package main

import (
	"math"
	"math/rand"
	"time"

	"github.com/egoist/mygo/ui"
)

// The confetti of a favorite: the pieces fly up from the middle of the field,
// turn, and fall away, as the canvas-confetti library does on the web.
const (
	confettiLife   = 1300 * time.Millisecond
	confettiPieces = 44
)

// confettiColors are the colors of the pieces: the app's own accents and a
// few festive ones, so a burst looks like a celebration and not a warning.
var confettiColors = []ui.Color{
	ui.Hex("#5b9bff"), ui.Hex("#f1b24a"), ui.Hex("#f27a9b"),
	ui.Hex("#46c08a"), ui.Hex("#a25afd"), ui.Hex("#ff8a3d"),
	ui.Hex("#26c6da"),
}

// confettiPiece is one piece of a burst: where it starts, which way it flies
// and how it turns.
type confettiPiece struct {
	vx, vy float32 // pixels per second
	spin   float32 // degrees per second
	w, h   float32
	delay  time.Duration
	color  ui.Color
	life   time.Duration
}

// confettiBurst is one celebration, in the box it may cover: the field is as
// large as the box, so a burst on a card dances the same dance as one on a
// stage, only smaller, and its pieces fade before they reach the edge.
type confettiBurst struct {
	at      time.Time
	area    ui.Rect
	x, y    float32 // the field's middle, from the area's top left corner
	gravity float32
	pieces  []confettiPiece
}

// confettiSquare is how wide the field of a burst is when the celebration has
// no card to play in, as in the viewer: a small patch around the heart.
const confettiSquare float32 = 200

// celebrate starts the confetti of a photo just made a favorite, from the
// control that was pressed and inside area, a box of the window: the pieces
// never leave the card they celebrate, and they do not vanish at its edge
// either -- the whole fall happens inside it.
func (a *app) celebrate(c *ui.Context, el ui.Element, area ui.Rect) {
	w, h := c.Size()
	if area.W <= 0 || area.H <= 0 {
		area = ui.Rect{X: w/2 - confettiSquare/2, Y: h/2 - confettiSquare/2, W: confettiSquare, H: confettiSquare}
	}
	// A card half scrolled out of the window keeps only the part shown, and
	// one too small to dance in gets a square of its own.
	x0, y0 := clamp32(area.X, 0, w), clamp32(area.Y, 0, h)
	x1, y1 := clamp32(area.X+area.W, 0, w), clamp32(area.Y+area.H, 0, h)
	if x1-x0 < 40 || y1-y0 < 40 {
		x0, y0 = w/2-confettiSquare/2, h/2-confettiSquare/2
		x1, y1 = x0+confettiSquare, y0+confettiSquare
	}
	area = ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
	// The burst starts where the heart was pressed, kept just inside the area.
	ox, oy := area.W/2, area.H/2
	if bx := el.Bounds(); bx.W > 0 {
		px, py, ok := el.PointerPosition()
		if !ok || px < 0 || py < 0 {
			px, py = bx.W/2, bx.H/2
		}
		ox = clamp32(bx.X+px-area.X, 6, area.W-6)
		oy = clamp32(bx.Y+py-area.Y, 6, area.H-6)
	}
	a.burstAt(c.Now(), rand.New(rand.NewSource(c.Now().UnixNano())), area, ox, oy)
}

// celebrateHere is a burst in a small square around the element pressed: the
// viewer and the keyboard have no card to hold the pieces.
func (a *app) celebrateHere(c *ui.Context, el ui.Element) {
	w, h := c.Size()
	cx, cy := w/2, h/2
	if bx := el.Bounds(); bx.W > 0 {
		cx, cy = bx.X+bx.W/2, bx.Y+bx.H/2
	}
	area := ui.Rect{X: cx - confettiSquare/2, Y: cy - confettiSquare/2, W: confettiSquare, H: confettiSquare}
	// Keep the field in the window.
	area.X = clamp32(area.X, 0, w-confettiSquare)
	area.Y = clamp32(area.Y, 0, h-confettiSquare)
	a.celebrate(c, el, area)
}

// burstAt is the burst itself, in an area and from a sequence of chance. The
// speeds, the gravity and the size of the pieces all follow the area's, so a
// small field gets a small, complete dance instead of a big one cut off. The
// pieces fly away from the point pressed and into the room around it: a heart
// in the corner of a card sprays across the card, not into its edge.
func (a *app) burstAt(now time.Time, r *rand.Rand, area ui.Rect, ox, oy float32) {
	radius := min(area.W, area.H) / 2
	if radius < 24 {
		radius = 24
	}
	scale := clamp32(radius/100, 0.55, 1.5)
	gravity := radius * 3.4
	// The way from the pressed point to the middle of the area: the pieces
	// fan out around it, where there is room to fly.
	dx, dy := area.W/2-ox, area.H/2-oy
	toMiddle := float32(math.Hypot(float64(dx), float64(dy)))
	towards := float32(math.Atan2(float64(dy), float64(dx)))
	pieces := make([]confettiPiece, confettiPieces)
	for i := range pieces {
		angle := towards + float32((r.Float64()*2-1)*0.8)
		if toMiddle < 4 {
			// Pressed in the middle: the classic burst, straight up.
			angle = float32(math.Pi * (-0.5 - 0.42*r.Float64()))
		}
		speed := float64(radius) * (0.9 + 1.2*r.Float64())
		pieces[i] = confettiPiece{
			vx:    float32(math.Cos(float64(angle)) * speed),
			vy:    float32(math.Sin(float64(angle)) * speed),
			spin:  float32((r.Float64()*2 - 1) * 540),
			w:     (6 + float32(r.Float64())*6) * scale,
			h:     (4 + float32(r.Float64())*4) * scale,
			delay: time.Duration(r.Float64() * 90 * float64(time.Millisecond)),
			color: confettiColors[r.Intn(len(confettiColors))],
			life:  confettiLife - time.Duration(r.Float64()*400*float64(time.Millisecond)),
		}
	}
	a.fetti = &confettiBurst{
		at: now, area: area,
		x: ox, y: oy, gravity: gravity,
		pieces: pieces,
	}
}

// confettiLayer draws the pieces of the burst that is running, in its own
// area over every panel, and asks for the frames that move them. It takes no
// input, and the pieces are clipped to the area as a last guard: the ones
// that would cross its edge have already faded.
func (a *app) confettiLayer(c *ui.Context) {
	b := a.fetti
	if b == nil {
		return
	}
	now := c.Now()
	if now.Sub(b.at) > confettiLife {
		a.fetti = nil
		return
	}
	c.AnimationFrame()
	ui.Box(c).Absolute().Left(b.area.X).Top(b.area.Y).
		Size(b.area.W, b.area.H).Clip().IgnoreInput(true).AccessHidden(true).Children(func() {
		for _, p := range b.pieces {
			t := now.Sub(b.at) - p.delay
			if t < 0 || t > p.life {
				continue
			}
			secs := float32(t.Seconds())
			x := b.x + p.vx*secs
			y := b.y + p.vy*secs + float32(0.5*b.gravity)*secs*secs
			// A piece turns edge on and shows as a thin line twice a turn.
			spin := float32(math.Abs(math.Cos(float64(p.spin*secs) * math.Pi / 180)))
			w := p.w * (0.25 + 0.75*spin)
			fade := float32(1)
			// It fades as it comes near the edge, so none is cut in half.
			edge := maxFloat32(abs32(x/b.area.W*2-1), abs32(y/b.area.H*2-1))
			if edge > 0.9 {
				fade *= clamp32((1-edge)/0.1, 0, 1)
			}
			// And again at the end of its fall.
			if left := p.life - t; left < 300*time.Millisecond {
				fade *= float32(left) / float32(300*time.Millisecond)
			}
			col := ui.RGBA(p.color.R, p.color.G, p.color.B, float32(p.color.A)/255*fade)
			ui.Box(c).Absolute().Left(x).Top(y).Size(w, p.h).
				Radius(1).Background(col).Rotate(p.spin * secs)
		}
	})
}

// abs32 is the distance of v from zero.
func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// maxFloat32 is the larger of two numbers.
func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
