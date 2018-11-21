package main

import (
	"strings"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
)

type instructionsState struct {
	lines *text.Text
}

func (s *instructionsState) enter(state) {
	if s.lines == nil {
		const instructions = `
Solve math problems.
Shoot zombies.
Survive!

Enter the solution to
the calculation above your
head to shoot your rifle.

Failing delays your next
shot.

Use the Left/Right arrow 
keys or A/D to move.


Press ENTER to play
`
		s.lines = text.New(pixel.ZV, font)
		for _, line := range strings.Split(instructions, "\n") {
			s.lines.Dot.X -= s.lines.BoundsOf(line).W() / 2
			s.lines.WriteString(line + "\n")
		}
	}
}

func (*instructionsState) leave() {}

func (s *instructionsState) update(window *pixelgl.Window) state {
	if window.JustPressed(pixelgl.KeyEscape) {
		return menu
	}
	if window.JustPressed(pixelgl.KeyEnter) || window.JustPressed(pixelgl.KeyKPEnter) {
		return playing
	}
	s.lines.Draw(window, pixel.IM.
		Moved(pixel.ZV.Sub(s.lines.Bounds().Center())).
		Scaled(pixel.ZV, 2.5).
		Moved(window.Bounds().Center()))

	return instructions
}
