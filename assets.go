package main

import (
	"image/png"
	"os"

	"github.com/faiface/beep"
	"github.com/faiface/beep/wav"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/text"
)

var (
	// file is the function to be called with a file name to create the absolute
	// path for assets. Only use paths created with this file.
	// In dev mode this will resolve to files in the local resource folder.
	// In release mode this will resolve to files in the executable's data blob.
	file func(filename string) string

	cleanUpAssets func() = func() {}

	font  *text.Atlas
	mixer beep.Mixer
	music *sound
)

type sound beep.Buffer

func (w *sound) play() {
	buf := (*beep.Buffer)(w)
	mixer.Add(buf.Streamer(0, buf.Len()))
}

func (w *sound) loop() {
	buf := (*beep.Buffer)(w)
	mixer.Add(beep.Loop(-1, buf.Streamer(0, buf.Len())))
}

func loadWav(path string) *sound {
	f, err := os.Open(path)
	check(err)
	defer f.Close()
	stream, format, err := wav.Decode(f)
	check(err)
	buf := beep.NewBuffer(format)
	buf.Append(stream)
	return (*sound)(buf)
}

func loadPNG(path string) *pixel.Sprite {
	f, err := os.Open(path)
	check(err)
	defer f.Close()
	img, err := png.Decode(f)
	check(err)
	pic := pixel.PictureDataFromImage(img)
	return pixel.NewSprite(pic, pic.Bounds())
}
