package main

import (
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
)

type menuState struct {
	hotItem  int
	items    []*text.Text
	menuBeep *sound
}

func (s *menuState) enter(state) {
	if len(s.items) == 0 {
		for i, caption := range []string{
			"Start Game",
			"How to Play",
			"High Scores",
			"Quit",
		} {
			s.items = append(s.items, text.New(pixel.V(0, 0), font))
			s.items[i].WriteString(caption)
		}

		s.menuBeep = loadWav(file("menu beep.wav"))
	}
}

func (*menuState) leave() {}

func (s *menuState) update(window *pixelgl.Window) state {
	var nextState state = menu
	if window.JustPressed(pixelgl.KeyEscape) {
		window.SetClosed(true)
	}
	oldItem := s.hotItem
	if window.JustPressed(pixelgl.KeyDown) {
		s.hotItem = (s.hotItem + 1) % len(s.items)
	}
	if window.JustPressed(pixelgl.KeyUp) {
		s.hotItem = (s.hotItem + len(s.items) - 1) % len(s.items)
	}
	if s.hotItem != oldItem {
		s.menuBeep.play()
	}

	if window.JustPressed(pixelgl.KeyEnter) || window.JustPressed(pixelgl.KeyKPEnter) {
		switch s.hotItem {
		case 0:
			nextState = playing
		case 1:
			nextState = instructions
		case 2:
			nextState = dead
		case 3:
			window.SetClosed(true)
		}
	}
	// render
	const textScale = 5
	for i, item := range s.items {
		h := item.Bounds().H()
		m := pixel.IM.
			Moved(pixel.V(0, -(float64(i)-float64(len(s.items))/2)*h)).
			Scaled(item.Bounds().Center(), textScale).
			Moved(window.Bounds().Center())
		m = pixel.IM.
			Moved(pixel.ZV.Sub(item.Bounds().Center())).
			Scaled(pixel.ZV, textScale).
			Moved(window.Bounds().Center()).
			Moved(pixel.V(0, -textScale*h*(0.5+float64(i)-float64(len(s.items))/2)))
		if i == s.hotItem {
			im := imdraw.New(nil)
			im.Color = pixel.RGB(0.5, 0, 0)
			r := item.Bounds()
			im.Push(
				m.Project(r.Min).Add(pixel.V(-20, 0)),
				m.Project(r.Max).Add(pixel.V(20, 0)),
			)
			im.Rectangle(0)
			im.Draw(window)
		}
		item.Draw(window, m)
	}
	return nextState
}
