package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
)

const (
	playerSpeed          = 4
	playerW, playerH     = 172, 207
	playerHeadH          = 60
	bulletShootOffsetY   = 103
	bulletW, bulletH     = 27, 9
	zombieW, zombieH     = 116, 218
	deadHeadW, deadHeadH = 87, 103
	zombieSpawnReduction = 0.97
	zombieSpawnMin       = 1000 * time.Millisecond
	zombieSpawnMax       = 2000 * time.Millisecond
	playerWalkFrames     = 4
	bloodW, bloodH       = 24, 20
	zombieDeathSounds    = 5
)

type torsoState int

const (
	idle torsoState = iota
	reloading
	waitingToReload
	shooting
	realizing
	aimingAtHead
	bleeding
)

func dying(s torsoState) bool {
	return s >= realizing
}

type playingState struct {
	playerX, playerY int
	playerFacingLeft bool
	playerWalkFrame  int
	playerWalkTime   int
	generator        mathGenerator
	assignment       assignment
	bullets          []bullet
	zombies          []zombie
	numbers          []fadingNumber
	nextZombie       int // time until next zombie spawns
	shootBan         int // time until shooting is allowed after wrong number
	score            int
	zombieSpawnDelay struct {
		minFrames, maxFrames float32
	}
	torso          torsoState
	torsoTime      int
	blood          []bloodParticle
	leaveStateTime int
	missShot       *sound
	zombieDeath    [zombieDeathSounds]*sound
	reload         *sound
	uhOh           *sound
	shot           *sound
	sprites        map[string]*pixel.Sprite
	bloodParticle  *pixel.Sprite
	bulletLeft     *pixel.Sprite
	bulletRight    *pixel.Sprite
	deadHead       *pixel.Sprite
	question       *text.Text
	scoreText      *text.Text
	number         *text.Text
}

func (s *playingState) enter(state) {
	if s.missShot == nil {
		s.missShot = loadWav(file("miss shot.wav"))
		for i := range s.zombieDeath {
			s.zombieDeath[i] = loadWav(file(fmt.Sprintf("zombie death %d.wav", i)))
		}
		s.reload = loadWav(file("reload.wav"))
		s.uhOh = loadWav(file("uh oh.wav"))
		s.shot = loadWav(file("shot.wav"))
		s.sprites = make(map[string]*pixel.Sprite)
		for _, key := range []string{
			"hero aiming at head left",
			"hero aiming at head right",
			"hero bleeding head left",
			"hero bleeding head right",
			"hero eye blink left",
			"hero eye blink right",
			"hero left",
			"hero right",
			"hero reload left",
			"hero reload right",
			"hero shoot left",
			"hero shoot right",
			"hero legs stand left",
			"hero legs stand right",
			"hero legs walk left 0",
			"hero legs walk right 0",
			"hero legs walk left 1",
			"hero legs walk right 1",
			"hero legs walk left 2",
			"hero legs walk right 2",
			"hero legs walk left 3",
			"hero legs walk right 3",
		} {
			s.sprites[key] = loadPNG(file(key + ".png"))
		}
		for _, dir := range []string{"left", "right"} {
			for z := 0; z < 3; z++ {
				key := fmt.Sprintf("zombie %d %s", z, dir)
				s.sprites[key] = loadPNG(file(key + ".png"))
				for i := 0; i < 4; i++ {
					key := fmt.Sprintf("zombie %d %s %d", z, dir, i)
					s.sprites[key] = loadPNG(file(key + ".png"))
				}
			}
		}
		s.bloodParticle = loadPNG(file("blood particle.png"))
		s.bulletLeft = loadPNG(file("bullet left.png"))
		s.bulletRight = loadPNG(file("bullet right.png"))
		s.deadHead = loadPNG(file("dead head.png"))
		s.question = text.New(pixel.V(0, 0), font)
		s.scoreText = text.New(pixel.V(0, 0), font)
		s.scoreText.Color = pixel.RGB(1, 0, 0)
		s.number = text.New(pixel.V(0, 0), font)
	}
	s.playerX = (windowW - playerW) / 2
	s.playerY = windowH - playerH - 100
	s.playerFacingLeft = false
	s.playerWalkFrame = 0
	s.playerWalkTime = 0
	s.generator = mathGenerator{
		ops: []mathOp{add, subtract, add, subtract, multiply, divide},
		max: 9,
	}
	s.assignment = s.generator.generate(rand.Int)
	s.question.Clear()
	s.question.WriteString(s.assignment.question)
	s.bullets = nil
	s.zombies = nil
	s.numbers = nil
	s.nextZombie = 0
	s.shootBan = 0
	s.score = 0
	s.scoreText.Clear()
	s.scoreText.WriteString(romanNumeral(s.score))
	s.zombieSpawnDelay.minFrames = float32(frames(zombieSpawnMin))
	s.zombieSpawnDelay.maxFrames = float32(frames(zombieSpawnMax))
	s.newZombie()
	s.torso = idle
	s.torsoTime = 0
	s.blood = nil
	s.leaveStateTime = -1
}

func (*playingState) leave() {}

var fireKeys = [10][2]pixelgl.Button{
	{pixelgl.Key0, pixelgl.KeyKP0},
	{pixelgl.Key1, pixelgl.KeyKP1},
	{pixelgl.Key2, pixelgl.KeyKP2},
	{pixelgl.Key3, pixelgl.KeyKP3},
	{pixelgl.Key4, pixelgl.KeyKP4},
	{pixelgl.Key5, pixelgl.KeyKP5},
	{pixelgl.Key6, pixelgl.KeyKP6},
	{pixelgl.Key7, pixelgl.KeyKP7},
	{pixelgl.Key8, pixelgl.KeyKP8},
	{pixelgl.Key9, pixelgl.KeyKP9},
}

func (s *playingState) update(window *pixelgl.Window) state {
	// handle input
	if window.JustPressed(pixelgl.KeyEscape) {
		if dying(s.torso) {
			return dead
		} else {
			return menu
		}
	}
	// shoot or miss
	s.shootBan--
	if s.shootBan < 0 {
		s.shootBan = 0
	}
	if !dying(s.torso) && s.shootBan <= 0 {
		wrongNumber := false
		for n, keys := range fireKeys {
			if window.JustPressed(keys[0]) || window.JustPressed(keys[1]) {
				if n != s.assignment.answer {
					wrongNumber = true
					s.missShot.play()
					s.addFadingNumber(n, pixel.RGB(1, 0, 0))
					s.shootBan = frames(500 * time.Millisecond)
					break
				}
			}
		}
		if !wrongNumber {
			keys := fireKeys[s.assignment.answer]
			if window.JustPressed(keys[0]) || window.JustPressed(keys[1]) {
				// add the number before shooting, shooting generates a new one
				s.addFadingNumber(s.assignment.answer, pixel.RGB(0, 1, 0))
				s.shoot(window)
			}
		}
	}
	// move left/right
	walking := false
	if !dying(s.torso) {
		const margin = -50
		if window.Pressed(pixelgl.KeyLeft) || window.Pressed(pixelgl.KeyA) {
			walking = true
			s.playerX -= playerSpeed
			if s.playerX < margin {
				s.playerX = margin
			}
			s.playerFacingLeft = true
		} else if window.Pressed(pixelgl.KeyRight) || window.Pressed(pixelgl.KeyD) {
			walking = true
			s.playerX += playerSpeed
			if s.playerX+playerW > windowW-margin {
				s.playerX = windowW - margin - playerW
			}
			s.playerFacingLeft = false
		}
	}
	if walking {
		s.playerWalkTime--
		if s.playerWalkTime <= 0 {
			s.playerWalkFrame = (s.playerWalkFrame + 1) % playerWalkFrames
			s.playerWalkTime = frames(100 * time.Millisecond)
		}
	} else {
		s.playerWalkFrame = 0
		s.playerWalkTime = 0
	}

	// update world
	if s.leaveStateTime > 0 {
		s.leaveStateTime--
		if s.leaveStateTime <= 0 {
			return dead
		}
	}
	// shoot bullets
	n := 0
	for i := range s.bullets {
		b := &s.bullets[i]
		bulletHitbox := rectangle{
			x: b.x,
			y: b.y,
			w: bulletW + abs(b.dx),
			h: bulletH,
		}
		if b.dx < 0 {
			bulletHitbox.x += b.dx
		}
		b.x += b.dx
		victimIndex := -1
		for i, z := range s.zombies {
			hitbox := rectangle{
				x: z.x + zombieW/4,
				y: z.y,
				w: zombieW / 2,
				h: zombieH,
			}
			if overlap(bulletHitbox, hitbox) {
				if victimIndex == -1 ||
					(b.dx > 0 && z.x < s.zombies[victimIndex].x) ||
					(b.dx < 0 && z.x > s.zombies[victimIndex].x) {
					victimIndex = i
				}
			}
		}
		if victimIndex != -1 {
			s.killZombie(victimIndex)
			s.zombieDeath[rand.Intn(len(s.zombieDeath))].play()
		}
		if victimIndex == -1 && (-100 <= b.x) && (b.x <= windowW+100) {
			s.bullets[n] = *b
			n++
		}
	}
	s.bullets = s.bullets[:n]
	// update fading numbers
	n = 0
	for i := range s.numbers {
		num := &s.numbers[i]
		num.life -= 0.02
		if num.life > 0 {
			s.numbers[n] = *num
			n++
		}
	}
	s.numbers = s.numbers[:n]
	// update zombies
	if !dying(s.torso) {
		s.nextZombie--
		if s.nextZombie <= 0 {
			s.newZombie()
		}
		for i := range s.zombies {
			z := &s.zombies[i]
			if z.facingLeft {
				z.x -= 2
			} else {
				z.x += 2
			}
			const hitDist = 40
			if abs((s.playerX+playerW/2)-(z.x+zombieW/2)) < hitDist {
				s.torso = realizing
				s.torsoTime = frames(time.Second)
			}
			const zombieFrameCount = 4
			z.nextFrame--
			if z.nextFrame <= 0 {
				z.nextFrame = frames(250 * time.Millisecond)
				z.frame = (z.frame + 1) % zombieFrameCount
			}
		}
	}
	// update blood and gore
	{
		n := 0
		for i := range s.blood {
			b := &s.blood[i]
			b.x += b.vx
			b.y += b.vy
			b.rotation += b.dRotation
			b.vy += 0.5
			if b.y < windowH {
				s.blood[n] = *b
				n++
			}
		}
		s.blood = s.blood[:n]
	}
	// animations
	if s.torsoTime > 0 {
		s.torsoTime--
		if s.torsoTime == 0 {
			switch s.torso {
			case idle:
				// nothing to do in this case
			case shooting:
				s.torso = waitingToReload
				s.torsoTime = frames(200 * time.Millisecond)
			case reloading:
				s.torso = idle
			case waitingToReload:
				s.torso = reloading
				s.torsoTime = frames(250 * time.Millisecond)
				s.reload.play()
			case realizing:
				s.torso = aimingAtHead
				s.torsoTime = frames(time.Second)
				s.uhOh.play()
			case aimingAtHead:
				s.torso = bleeding
				s.shot.play()
				x, y := s.playerNeck()
				s.sprayBlood(x, y, 100, 200)
				s.torsoTime = frames(50 * time.Millisecond)
				s.leaveStateTime = frames(3 * time.Second)
			case bleeding:
				// nothing to do in this case
				s.torsoTime = frames(50 * time.Millisecond)
				x, y := s.playerNeck()
				s.sprayBlood(x, y, 5, 10)
			}
		}
	}

	// render
	// background
	{
		const h = 3
		im := imdraw.New(nil)
		for y := 0; y < windowH; y += h {
			im.Color = pixel.RGB(0, 0, float64(y+50)/windowH)
			im.Push(
				pixel.V(0, float64(windowH-y-h)),
				pixel.V(windowW, float64(windowH-y)),
			)
			im.Rectangle(0)
		}
		const groundH = 170
		groundCenter := pixel.RGB(135/255.0, 33/255.0, 2/255.0)
		groundEdge := pixel.RGB(95/255.0, 23/255.0, 1/255.0)
		for y := windowH - groundH; y < windowH; y += h {
			centerWeight := 1.0 - float64(abs(y-(windowH-groundH/2)))/80.0
			im.Color = pixel.RGB(
				groundCenter.R*centerWeight+groundEdge.R*(1-centerWeight),
				groundCenter.G*centerWeight+groundEdge.G*(1-centerWeight),
				groundCenter.B*centerWeight+groundEdge.B*(1-centerWeight),
			)
			im.Push(
				pixel.V(0, float64(windowH-y-h)),
				pixel.V(windowW, float64(windowH-y)),
			)
			im.Rectangle(0)
		}
		im.Draw(window)
	}
	// player
	hero := "hero "
	if s.torso == reloading {
		hero += "reload "
	}
	if s.torso == shooting {
		hero += "shoot "
	}
	if s.torso == aimingAtHead {
		hero += "aiming at head "
	}
	if s.torso == bleeding {
		hero += "bleeding head "
	}
	dir := "right"
	if s.playerFacingLeft {
		dir = "left"
	}
	hero += dir
	s.sprites[hero].Draw(window, pixel.IM.Moved(
		pixel.V(float64(s.playerX), float64(windowH-s.playerY))).
		Moved(pixel.ZV.Sub(s.sprites[hero].Picture().Bounds().Center())))
	if s.shootBan > 0 {
		head := s.sprites["hero eye blink "+dir]
		head.Draw(window, pixel.IM.Moved(
			pixel.V(float64(s.playerX), float64(windowH-s.playerY))).
			Moved(pixel.ZV.Sub(head.Picture().Bounds().Center())))
	}
	var legs *pixel.Sprite
	if walking {
		sprite := fmt.Sprintf("hero legs walk %s %d", dir, s.playerWalkFrame)
		legs = s.sprites[sprite]
	} else {
		legs = s.sprites["hero legs stand "+dir]
	}
	legs.Draw(window, pixel.IM.Moved(
		pixel.V(float64(s.playerX), float64(windowH-s.playerY))).
		Moved(pixel.ZV.Sub(legs.Picture().Bounds().Center())))
	// zombies
	for _, z := range s.zombies {
		dir := "right"
		if z.facingLeft {
			dir = "left"
		}
		var img string
		if dying(s.torso) {
			img = fmt.Sprintf("zombie %d %s", z.kind, dir)
		} else {
			img = fmt.Sprintf("zombie %d %s %d", z.kind, dir, z.frame)
		}
		zombie := s.sprites[img]
		zombie.Draw(window, pixel.IM.Moved(
			pixel.V(float64(z.x), float64(windowH-z.y))).
			Moved(pixel.ZV.Sub(zombie.Picture().Bounds().Center())))
	}
	// blood and gore
	for i := range s.blood {
		b := &s.blood[i]
		s.bloodParticle.Draw(window, pixel.IM.
			Rotated(pixel.ZV, b.rotation).
			Moved(pixel.V(b.x, windowH-b.y)).
			Moved(pixel.ZV.Sub(s.bloodParticle.Picture().Bounds().Center())))
	}
	// bullets
	for _, b := range s.bullets {
		img := s.bulletLeft
		if b.dx > 0 {
			img = s.bulletRight
		}
		img.Draw(window, pixel.IM.
			Moved(pixel.V(float64(b.x), float64(windowH-b.y))))
	}
	// score
	{
		b := s.deadHead.Picture().Bounds()
		s.deadHead.Draw(window, pixel.IM.Moved(b.Center().Add(pixel.V(0, windowH-b.H()))))
		const textScale = 4
		s.scoreText.Draw(window, pixel.IM.
			Moved(pixel.ZV.Sub(s.scoreText.Bounds().Center())).
			Scaled(pixel.ZV, textScale).
			Moved(pixel.V(s.scoreText.Bounds().W()*textScale/2+deadHeadW, windowH-deadHeadH/2)))
	}
	// fading numbers from the past
	for _, num := range s.numbers {
		scale := 4 + 7*(1-num.life)
		color := num.color
		color.A = num.life
		s.number.Clear()
		s.number.Color = color
		s.number.WriteString(num.text)
		c := s.number.Bounds().Center()
		s.number.DrawColorMask(window, pixel.IM.Moved(pixel.ZV.Sub(c)).
			Scaled(pixel.ZV, scale).
			Moved(window.Bounds().Center().Add(pixel.V(0, windowH/2-100))),
			color)
	}
	// assigment
	const mathScale = 3
	s.question.Draw(window, pixel.IM.
		Moved(pixel.ZV.Sub(s.question.Bounds().Center())).
		Scaled(pixel.ZV, mathScale).
		Moved(pixel.V(float64(s.playerX+playerW/2), float64(windowH-s.playerY)+s.question.Bounds().H()*4)))

	return playing
}

func (s *playingState) shoot(window *pixelgl.Window) {
	s.shot.play()
	const bulletSpeed = 30
	var b bullet
	b.y = s.playerY + bulletShootOffsetY
	if s.playerFacingLeft {
		b.x = s.playerX
		b.dx = -bulletSpeed
	} else {
		b.x = s.playerX + playerW - bulletW
		b.dx = bulletSpeed
	}
	s.bullets = append(s.bullets, b)
	s.assignment = s.generator.generate(rand.Int)
	s.question.Clear()
	s.question.WriteString(s.assignment.question)
	s.torso = shooting
	s.torsoTime = frames(100 * time.Millisecond)
}

func (s *playingState) killZombie(i int) {
	// spray blood
	z := s.zombies[i]
	cx, cy := z.x+zombieW/2, z.y+zombieH/2
	s.sprayBlood(cx, cy, 10, 30)

	// remove zombie from list
	copy(s.zombies[i:], s.zombies[i+1:])
	s.zombies = s.zombies[:len(s.zombies)-1]
	s.score++
	s.scoreText.Clear()
	s.scoreText.WriteString(romanNumeral(s.score))
	min, max := s.zombieSpawnDelay.minFrames, s.zombieSpawnDelay.maxFrames
	s.zombieSpawnDelay.minFrames = min * zombieSpawnReduction
	if s.score%2 == 1 {
		s.zombieSpawnDelay.maxFrames = max * zombieSpawnReduction
	}
}

func (s *playingState) sprayBlood(x, y, min, max int) {
	count := min + rand.Intn(max-min)
	for i := 0; i < count; i++ {
		s.blood = append(s.blood, bloodParticle{
			x:         float64(x - bloodW/2),
			y:         float64(y - bloodH/2),
			vx:        3 - 6*rand.Float64(),
			vy:        -10 - 5*rand.Float64(),
			rotation:  2 * math.Pi * rand.Float64(),
			dRotation: 0.035 - 0.07*rand.Float64(),
		})
	}
}

func (s *playingState) addFadingNumber(n int, color pixel.RGBA) {
	s.numbers = append(s.numbers, fadingNumber{
		text:  fmt.Sprintf("%d", n),
		life:  1.0,
		color: color,
	})
}

func (s *playingState) newZombie() {
	var z zombie
	z.facingLeft = rand.Intn(2) == 0
	z.y = s.playerY + playerH - zombieH - 10 + rand.Intn(30)
	if z.facingLeft {
		z.x = windowW
	} else {
		z.x = -zombieW
	}
	const zombieKindCount = 3
	z.kind = rand.Intn(zombieKindCount)
	s.zombies = append(s.zombies, z)
	min := round(s.zombieSpawnDelay.minFrames)
	max := round(s.zombieSpawnDelay.maxFrames)
	s.nextZombie = min + rand.Intn(max-min)
}

func (s *playingState) playerNeck() (x, y int) {
	dx := -6
	if s.playerFacingLeft {
		dx = -dx
	}
	return s.playerX + playerW/2 + dx, s.playerY + playerHeadH
}

type fadingNumber struct {
	text  string
	life  float64
	color pixel.RGBA
}

type bullet struct {
	x, y int
	dx   int
}

type zombie struct {
	x, y       int
	facingLeft bool
	frame      int
	nextFrame  int
	kind       int
}

type bloodParticle struct {
	x, y      float64
	vx, vy    float64
	rotation  float64
	dRotation float64
}
