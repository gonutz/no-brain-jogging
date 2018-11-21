package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
)

const (
	maxHighScores   = 5
	maxNameLen      = 20
	cursorBlinkTime = 300 * time.Millisecond
)

type deadState struct {
	caption        string
	blink          int
	restartVisible bool
	highscores     []highscore
	editing        int
	cursorBlink    int
	cursorVisible  bool
	score          int
	text           *text.Text
}

func (s *deadState) enter(oldState state) {
	if s.text == nil {
		s.text = text.New(pixel.ZV, font)
	}
	s.score = -1
	s.restartVisible = true
	s.blink = 0
	s.editing = -1
	s.highscores = loadHighScores()
	if len(s.highscores) < maxHighScores {
		s.highscores = append(s.highscores, make([]highscore, maxHighScores-len(s.highscores))...)
	}
	s.caption = "High Scores"
	if oldState == playing {
		s.caption = "You were eaten alive!"
		score := playing.score
		s.score = score
		s.highscores = append(s.highscores, highscore{
			score: score,
			name:  "",
			id:    1,
		})
		sort.Stable(byScore(s.highscores))
		if len(s.highscores) > maxHighScores {
			s.highscores = s.highscores[:maxHighScores]
		}
		saveHighScores(s.highscores)
		s.editing = -1
		for i := range s.highscores {
			if s.highscores[i].id == 1 {
				s.editing = i
			}
		}
	}
	s.restartVisible = false
	s.cursorBlink = 0
	s.cursorVisible = false
}

func (*deadState) leave() {}

func (s *deadState) update(window *pixelgl.Window) state {
	var nextState state = dead
	// handle input
	if window.JustPressed(pixelgl.KeyEscape) {
		nextState = menu
	}
	if window.JustPressed(pixelgl.KeyEnter) || window.JustPressed(pixelgl.KeyKPEnter) {
		if s.editing != -1 {
			s.editing = -1
			saveHighScores(s.highscores)
			s.restartVisible = false
			s.blink = 0
		} else {
			nextState = playing
		}
	}
	// text input if editing high score name
	if s.editing != -1 {
		score := &s.highscores[s.editing]
		typed := window.Typed()
		for _, r := range typed {
			if len(score.name) < maxNameLen && (32 <= r) && (r <= 126) {
				score.name += string(r)
			}
			s.cursorVisible = true
			s.cursorBlink = frames(cursorBlinkTime)
		}
		if window.JustPressed(pixelgl.KeyBackspace) && score.name != "" {
			_, size := utf8.DecodeLastRuneInString(score.name)
			score.name = score.name[:len(score.name)-size]
			s.cursorVisible = true
			s.cursorBlink = frames(cursorBlinkTime)
		}
	}
	// update animations
	s.blink--
	if s.blink <= 0 {
		s.restartVisible = !s.restartVisible
		if s.restartVisible {
			s.blink = frames(700 * time.Millisecond)
		} else {
			s.blink = frames(400 * time.Millisecond)
		}
	}
	s.cursorBlink--
	if s.cursorBlink < 0 {
		s.cursorVisible = !s.cursorVisible
		s.cursorBlink = frames(cursorBlinkTime)
	}
	// render
	var allText string
	if s.score >= 0 {
		suffix := "s"
		if s.score == 1 {
			suffix = ""
		}
		allText += fmt.Sprintf("You killed %d zombie%s", s.score, suffix)
	} else {
		allText += "High Scores"
	}
	allText += "\n\n\n"
	maxScore := 0
	for _, h := range s.highscores {
		if h.score > maxScore {
			maxScore = h.score
		}
	}
	scoreW := 1
	for maxScore/10 > 0 {
		scoreW++
		maxScore /= 10
	}
	for i, score := range s.highscores {
		name := score.name
		if i == s.editing {
			if s.cursorVisible {
				name += "|"
			} else {
				name += " "
			}
		}
		if len(name) < maxNameLen {
			name += strings.Repeat("_", maxNameLen-len(name))
		}
		space := " "
		if len(name) > maxNameLen {
			space = ""
		}
		scoreText := strconv.Itoa(score.score)
		for len(scoreText) < scoreW {
			scoreText = " " + scoreText
		}
		allText += fmt.Sprintf("%d. %s%s%s\n", i+1, name, space, scoreText)
	}
	allText += "\n\n"
	if s.editing == -1 && s.restartVisible {
		allText += "Press ENTER to play"
	}
	allText += "\n "

	s.text.Clear()
	for _, line := range strings.Split(allText, "\n") {
		s.text.Dot.X -= s.text.BoundsOf(line).W() / 2
		s.text.WriteString(line + "\n")
	}
	s.text.Draw(window, pixel.IM.
		Moved(pixel.ZV.Sub(s.text.Bounds().Center())).
		Scaled(pixel.ZV, 3).
		Moved(window.Bounds().Center()))

	return nextState
}
