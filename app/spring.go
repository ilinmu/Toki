package main

// The springs that move the thumbnail width and the viewer's zoom. They follow
// a mass-spring-damper model, as MyGo's own spring animations do: a little
// overshoot makes the motion feel elastic instead of starting and stopping
// abruptly. The step is split so a long frame cannot make the spring explode.
const (
	thumbSpringK = 230.0
	thumbSpringC = 24.0
	zoomSpringK  = 190.0
	zoomSpringC  = 23.0
)

// springStep advances one value toward target over dt seconds.
func springStep(v, target, vel, dt, k, c float64) (float64, float64) {
	if dt <= 0 {
		dt = 1.0 / 60
	}
	if dt > 0.05 {
		dt = 0.05
	}
	for dt > 0 {
		h := min(dt, 1.0/240)
		accel := -k*(v-target) - c*vel
		vel += accel * h
		v += vel * h
		dt -= h
	}
	return v, vel
}

// springAtRest reports whether the value has settled, with a tolerance in its
// own units.
func springAtRest(v, target, vel float64) bool {
	eps := max(0.0005, absFloat(target)*0.0005)
	return absFloat(v-target) < eps && absFloat(vel) < eps*8
}

// absFloat is the distance of v from zero.
func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
