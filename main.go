package main

import (
	"math/rand"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

const (
	windowTitle      = "No-Brain Jogging"
	windowW, windowH = 1200, 600
	musicLength      = 8081 * time.Millisecond
)

type state interface {
	enter(from state)
	update(window *pixelgl.Window) state
	leave()
}

// all game states
var (
	loading      = &loadingState{}
	menu         = &menuState{}
	playing      = &playingState{}
	dead         = &deadState{}
	instructions = &instructionsState{}
)

func run() {
	rand.Seed(time.Now().UnixNano())

	var state state = loading
	state.enter(nil)

	defer cleanUpAssets()

	cfg := pixelgl.WindowConfig{
		Title:  windowTitle,
		Bounds: pixel.R(0, 0, windowW, windowH),
		VSync:  true,
	}
	window, err := pixelgl.NewWindow(cfg)
	check(err)
	window.SetCursorVisible(false)

	var sampleRate beep.SampleRate = 44100
	check(speaker.Init(sampleRate, sampleRate.N(100*time.Millisecond)))
	speaker.Play(&mixer)

	for !window.Closed() {
		window.Clear(colornames.Black)

		newState := state.update(window)
		if state != newState {
			state.leave()
			newState.enter(state)
		}
		state = newState

		window.Update()
	}
}

func main() {
	pixelgl.Run(run)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
