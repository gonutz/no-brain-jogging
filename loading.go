package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/gonutz/blob"
	"github.com/gonutz/payload"
	"golang.org/x/image/font/basicfont"
)

type loadingState struct {
	assetsLoaded bool
	text         *text.Text
}

func (s *loadingState) enter(state) {
	font = text.NewAtlas(basicfont.Face7x13, text.ASCII)
	s.text = text.New(pixel.V(0, 0), font)
	s.text.WriteString("Loading...")
	load, err := payload.Open()
	if err == nil {
		// this executable has a blob of assets attached, write them to disk for
		// the prototype library to use them
		data, err := blob.Open(load)
		check(err)
		dir, err := ioutil.TempDir("", "ld41_")
		check(err)
		cleanUpAssets = func() {
			os.RemoveAll(dir)
		}
		file = func(filename string) string {
			return filepath.Join(dir, filename)
		}
		go func() {
			defer func() { s.assetsLoaded = true }()
			for i := 0; i < data.ItemCount(); i++ {
				id := data.GetIDAtIndex(i)
				r, _ := data.GetByIndex(i)
				func() {
					f, err := os.Create(file(id))
					check(err)
					defer f.Close()
					_, err = io.Copy(f, r)
					check(err)
				}()
			}
		}()
	} else {
		// no payload in this executable, load from files
		file = func(f string) string { return filepath.Join("rsc", f) }
		s.assetsLoaded = true
	}
}

func (*loadingState) leave() {}

func (s *loadingState) update(window *pixelgl.Window) state {
	s.text.Draw(window, pixel.IM.
		Scaled(s.text.Bounds().Center(), 5).
		Moved(window.Bounds().Center()),
	)
	if s.assetsLoaded {
		music = loadWav(file("music.wav"))
		music.loop()
		return menu
	} else {
		return loading
	}
}
