package main

type rectangle struct {
	x, y, w, h int
}

func (r rectangle) contains(x, y int) bool {
	return x >= r.x && y >= r.y && x < r.x+r.w && y < r.y+r.h
}

func overlap(r, s rectangle) bool {
	return s.x+s.w >= r.x && s.y+s.h >= r.y && s.x < r.x+r.w && s.y < r.y+r.h
}
