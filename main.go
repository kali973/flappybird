package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ─────────────────────────────────────────────────────────────
// CONSTANTS
// ─────────────────────────────────────────────────────────────

const (
	ScreenW     = 480
	ScreenH     = 720
	BirdX       = 110
	PipeWidth   = 74
	PipeGap     = 188
	GroundH     = 90
	PipeSpeed   = 2.9
	PipeSpacing = 285
	Gravity     = 0.43
	JumpForce   = -9.4
	MaxFallVel  = 13.0
)

type GameState int

const (
	StateMenu GameState = iota
	StatePlaying
	StateDying
	StateGameOver
)

// ─────────────────────────────────────────────────────────────
// COLORS
// ─────────────────────────────────────────────────────────────

var (
	colSkyTop     = color.RGBA{72, 154, 210, 255}
	colSkyMid     = color.RGBA{135, 206, 235, 255}
	colSkyBot     = color.RGBA{175, 228, 240, 255}
	colGrassTop   = color.RGBA{108, 195, 78, 255}
	colGrass      = color.RGBA{92, 175, 60, 255}
	colDirtLight  = color.RGBA{200, 150, 90, 255}
	colDirtMid    = color.RGBA{175, 125, 65, 255}
	colDirtDark   = color.RGBA{155, 105, 48, 255}
	colPipeHL     = color.RGBA{120, 215, 85, 255}
	colPipeMid    = color.RGBA{88, 188, 56, 255}
	colPipeDark   = color.RGBA{58, 148, 32, 255}
	colPipeEdge   = color.RGBA{42, 118, 22, 255}
	colCapTop     = color.RGBA{110, 210, 76, 255}
	colCapBot     = color.RGBA{68, 162, 40, 255}
	colBirdYellow = color.RGBA{255, 218, 38, 255}
	colBirdLight  = color.RGBA{255, 238, 120, 255}
	colBirdDark   = color.RGBA{220, 170, 20, 255}
	colBirdWing   = color.RGBA{235, 190, 25, 255}
	colBeak1      = color.RGBA{255, 135, 0, 255}
	colBeak2      = color.RGBA{225, 100, 0, 255}
	colWhite      = color.RGBA{255, 255, 255, 255}
	colBlack      = color.RGBA{0, 0, 0, 255}
	colGold       = color.RGBA{255, 210, 50, 255}
)

// ─────────────────────────────────────────────────────────────
// FONTS
// ─────────────────────────────────────────────────────────────

var (
	fontFaceSmall *text.GoTextFace
	fontFaceMid   *text.GoTextFace
	fontFaceLarge *text.GoTextFace
	fontFaceXL    *text.GoTextFace
	fontFaceXXL   *text.GoTextFace
)

func initFonts() {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal("font load:", err)
	}
	fontFaceSmall = &text.GoTextFace{Source: src, Size: 12}
	fontFaceMid = &text.GoTextFace{Source: src, Size: 16}
	fontFaceLarge = &text.GoTextFace{Source: src, Size: 24}
	fontFaceXL = &text.GoTextFace{Source: src, Size: 32}
	fontFaceXXL = &text.GoTextFace{Source: src, Size: 48}
}

// ─────────────────────────────────────────────────────────────
// STRUCTS
// ─────────────────────────────────────────────────────────────

type Particle struct {
	x, y    float64
	vx, vy  float64
	life    float64
	size    float64
	r, g, b uint8
}

type Cloud struct {
	x, y  float64
	scale float64
	speed float64
	alpha float64
}

type Pipe struct {
	x      float64
	gapY   float64
	scored bool
}

type Bird struct {
	y         float64
	vy        float64
	angle     float64
	wingPhase float64
}

type Game struct {
	state GameState
	bird  Bird
	pipes []Pipe

	particles []Particle
	clouds    []Cloud

	score int
	best  int
	tick  int

	groundOff  float64
	shakeX     float64
	shakeY     float64
	flashAlpha float64
	deathTimer int

	birdFrames [8]*ebiten.Image
	bgImg      *ebiten.Image
}

// ─────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────

func main() {
	initFonts()
	g := newGame()
	ebiten.SetWindowSize(ScreenW, ScreenH)
	ebiten.SetWindowTitle("Flappy Bird Pro - Go Edition")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetTPS(60)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// ─────────────────────────────────────────────────────────────
// INIT
// ─────────────────────────────────────────────────────────────

func newGame() *Game {
	g := &Game{}
	g.initBackground()
	g.initBirdFrames()
	g.initClouds()
	g.resetBird()
	return g
}

func (g *Game) resetBird() {
	g.bird = Bird{y: float64(ScreenH)/2 - 40}
}

func (g *Game) startGame() {
	g.state = StatePlaying
	g.score = 0
	g.pipes = nil
	g.particles = nil
	g.groundOff = 0
	g.tick = 0
	g.shakeX, g.shakeY = 0, 0
	g.flashAlpha = 0
	g.resetBird()
}

func (g *Game) initBackground() {
	img := ebiten.NewImage(ScreenW, ScreenH-GroundH)
	skyH := ScreenH - GroundH
	for y := 0; y < skyH; y++ {
		t := float64(y) / float64(skyH)
		var r, gr, b uint8
		if t < 0.5 {
			tt := t * 2
			r = lerp8(colSkyTop.R, colSkyMid.R, tt)
			gr = lerp8(colSkyTop.G, colSkyMid.G, tt)
			b = lerp8(colSkyTop.B, colSkyMid.B, tt)
		} else {
			tt := (t - 0.5) * 2
			r = lerp8(colSkyMid.R, colSkyBot.R, tt)
			gr = lerp8(colSkyMid.G, colSkyBot.G, tt)
			b = lerp8(colSkyMid.B, colSkyBot.B, tt)
		}
		ebitenutil.DrawRect(img, 0, float64(y), ScreenW, 1, color.RGBA{r, gr, b, 255})
	}
	g.bgImg = img
}

func (g *Game) initBirdFrames() {
	for i := 0; i < 8; i++ {
		g.birdFrames[i] = renderBirdFrame(i)
	}
}

func renderBirdFrame(frame int) *ebiten.Image {
	const W, H = 52, 42
	img := ebiten.NewImage(W, H)
	wingAngle := math.Sin(float64(frame) * math.Pi / 4)
	wingOff := float32(wingAngle * 9)
	cx := float32(W / 2)
	cy := float32(H / 2)
	// Tail
	vector.DrawFilledRect(img, cx-20, cy-4, 12, 8, colBirdDark, true)
	vector.DrawFilledRect(img, cx-22, cy-6, 8, 4, colBirdWing, true)
	vector.DrawFilledRect(img, cx-22, cy+2, 8, 4, colBirdWing, true)
	// Wing shadow + wing
	vector.DrawFilledCircle(img, cx-2, cy+wingOff, 13, colBirdDark, true)
	vector.DrawFilledCircle(img, cx-2, cy+wingOff, 11, colBirdWing, true)
	// Body
	vector.DrawFilledCircle(img, cx, cy, 17, colBirdYellow, true)
	vector.DrawFilledCircle(img, cx+2, cy+3, 11, colBirdLight, true)
	// Eye
	vector.DrawFilledCircle(img, cx+9, cy-6, 8, colBirdDark, true)
	vector.DrawFilledCircle(img, cx+8, cy-7, 7, colWhite, true)
	vector.DrawFilledCircle(img, cx+9, cy-7, 4, colBlack, true)
	vector.DrawFilledCircle(img, cx+11, cy-9, 2, colWhite, true)
	// Beak
	vector.DrawFilledRect(img, cx+14, cy-5, 13, 6, colBeak1, true)
	vector.DrawFilledRect(img, cx+14, cy+1, 11, 5, colBeak2, true)
	vector.DrawFilledRect(img, cx+14, cy, 12, 1, colBirdDark, true)
	return img
}

func (g *Game) initClouds() {
	g.clouds = make([]Cloud, 9)
	for i := range g.clouds {
		g.clouds[i] = Cloud{
			x:     float64(rand.Intn(ScreenW + 200)),
			y:     float64(15 + rand.Intn(220)),
			scale: 0.45 + rand.Float64()*0.9,
			speed: 0.25 + rand.Float64()*0.45,
			alpha: 0.55 + rand.Float64()*0.45,
		}
	}
}

func (g *Game) Layout(_, _ int) (int, int) { return ScreenW, ScreenH }

// ─────────────────────────────────────────────────────────────
// UPDATE
// ─────────────────────────────────────────────────────────────

func (g *Game) Update() error {
	g.tick++
	jumped := inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) ||
		inpututil.IsKeyJustPressed(ebiten.KeyW) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.AppendJustPressedTouchIDs(nil)) > 0

	switch g.state {
	case StateMenu:
		g.bird.y = float64(ScreenH)/2 - 40 + math.Sin(float64(g.tick)*0.065)*14
		g.bird.angle = math.Sin(float64(g.tick)*0.05) * 0.12
		g.bird.wingPhase = float64(g.tick) * 0.18
		g.updateClouds()
		if jumped {
			g.startGame()
		}

	case StatePlaying:
		if jumped {
			g.bird.vy = JumpForce
			g.bird.wingPhase = 0
		}
		g.bird.vy += Gravity
		if g.bird.vy > MaxFallVel {
			g.bird.vy = MaxFallVel
		}
		g.bird.y += g.bird.vy
		target := g.bird.vy * 0.082
		if target > math.Pi/3 {
			target = math.Pi / 3
		}
		if target < -math.Pi/5 {
			target = -math.Pi / 5
		}
		g.bird.angle += (target - g.bird.angle) * 0.12
		g.bird.wingPhase += 0.20
		g.groundOff -= PipeSpeed
		if g.groundOff < -float64(ScreenW) {
			g.groundOff += float64(ScreenW)
		}
		g.updateClouds()
		g.updateParticles()

		// Spawn pipes
		if len(g.pipes) == 0 || g.pipes[len(g.pipes)-1].x < float64(ScreenW-PipeSpacing) {
			minGY := 145.0
			maxGY := float64(ScreenH-GroundH) - 145.0
			g.pipes = append(g.pipes, Pipe{
				x:    float64(ScreenW + 30),
				gapY: minGY + rand.Float64()*(maxGY-minGY),
			})
		}

		// Update pipes
		alive := g.pipes[:0]
		for i := range g.pipes {
			g.pipes[i].x -= PipeSpeed
			p := &g.pipes[i]
			if !p.scored && p.x+PipeWidth < BirdX {
				p.scored = true
				g.score++
			}
			const hitR = 12.0
			bL := float64(BirdX) - hitR
			bR := float64(BirdX) + hitR
			bT := g.bird.y - hitR
			bB := g.bird.y + hitR
			pL := p.x + 4
			pR := p.x + PipeWidth - 4
			gTop := p.gapY - PipeGap/2
			gBot := p.gapY + PipeGap/2
			if bR > pL && bL < pR {
				if bT < gTop || bB > gBot {
					g.die()
					return nil
				}
			}
			if p.x+PipeWidth > -50 {
				alive = append(alive, *p)
			}
		}
		g.pipes = alive

		if g.bird.y < 14 {
			g.bird.y = 14
			g.bird.vy = 0
		}
		if g.bird.y > float64(ScreenH-GroundH)-14 {
			g.die()
		}

	case StateDying:
		g.deathTimer--
		g.bird.vy += Gravity * 2.2
		if g.bird.vy > 18 {
			g.bird.vy = 18
		}
		g.bird.y += g.bird.vy
		g.bird.angle += 0.14
		g.shakeX *= 0.78
		g.shakeY *= 0.78
		g.flashAlpha *= 0.88
		g.updateParticles()
		if g.deathTimer <= 0 || g.bird.y > float64(ScreenH) {
			g.state = StateGameOver
			g.deathTimer = -45
		}

	case StateGameOver:
		g.updateParticles()
		if g.deathTimer < 0 {
			g.deathTimer++
		}
		if jumped && g.deathTimer == 0 {
			g.startGame()
		}
	}
	return nil
}

func (g *Game) die() {
	g.state = StateDying
	g.deathTimer = 72
	g.shakeX, g.shakeY = 14, 12
	g.flashAlpha = 1.0
	if g.score > g.best {
		g.best = g.score
	}
	g.spawnDeathParticles()
}

func (g *Game) updateClouds() {
	for i := range g.clouds {
		c := &g.clouds[i]
		c.x -= c.speed
		if c.x < -200 {
			c.x = float64(ScreenW + 80)
			c.y = float64(15 + rand.Intn(220))
			c.scale = 0.45 + rand.Float64()*0.9
			c.alpha = 0.55 + rand.Float64()*0.45
		}
	}
}

func (g *Game) spawnDeathParticles() {
	clrs := []color.RGBA{colBirdYellow, colBirdLight, colBirdDark, colBirdWing, colBeak1, colWhite}
	for i := 0; i < 48; i++ {
		angle := rand.Float64() * math.Pi * 2
		speed := 1.8 + rand.Float64()*7.0
		c := clrs[rand.Intn(len(clrs))]
		g.particles = append(g.particles, Particle{
			x: float64(BirdX), y: g.bird.y,
			vx: math.Cos(angle) * speed, vy: math.Sin(angle)*speed - 3.0,
			life: 1.0, size: 2.0 + rand.Float64()*5.5,
			r: c.R, g: c.G, b: c.B,
		})
	}
}

func (g *Game) updateParticles() {
	alive := g.particles[:0]
	for i := range g.particles {
		p := &g.particles[i]
		p.x += p.vx
		p.y += p.vy
		p.vy += 0.18
		p.vx *= 0.97
		p.life -= 0.018
		if p.life > 0 && p.y < float64(ScreenH) {
			alive = append(alive, *p)
		}
	}
	g.particles = alive
}

// ─────────────────────────────────────────────────────────────
// DRAW
// ─────────────────────────────────────────────────────────────

func (g *Game) Draw(screen *ebiten.Image) {
	var sx, sy float64
	if g.shakeX > 0.5 {
		sx = (rand.Float64()*2 - 1) * g.shakeX
	}
	if g.shakeY > 0.5 {
		sy = (rand.Float64()*2 - 1) * g.shakeY
	}

	// Background
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx*0.2, sy*0.2)
	screen.DrawImage(g.bgImg, op)

	// Clouds
	for _, c := range g.clouds {
		drawCloud(screen, c.x+sx*0.35, c.y+sy*0.35, c.scale, c.alpha)
	}

	// Mountains
	drawMountains(screen, sx*0.5, sy*0.5)

	// Pipes
	if g.state != StateMenu {
		for _, p := range g.pipes {
			drawPipe(screen, p.x+sx, p.gapY+sy)
		}
	}

	// Ground
	drawGround(screen, g.groundOff+sx, sy)

	// Particles
	for _, p := range g.particles {
		a := uint8(p.life * 255)
		vector.DrawFilledCircle(screen, float32(p.x+sx), float32(p.y+sy),
			float32(p.size*p.life), color.RGBA{p.r, p.g, p.b, a}, true)
	}

	// Bird
	if g.state != StateGameOver {
		g.drawBird(screen, sx, sy)
	}

	// Flash
	if g.flashAlpha > 0.01 {
		fl := ebiten.NewImage(1, 1)
		fl.Fill(color.RGBA{255, 255, 255, uint8(g.flashAlpha * 210)})
		fop := &ebiten.DrawImageOptions{}
		fop.GeoM.Scale(ScreenW, ScreenH)
		screen.DrawImage(fl, fop)
	}

	// UI
	switch g.state {
	case StateMenu:
		g.drawMenu(screen)
	case StatePlaying:
		g.drawHUD(screen)
	case StateDying:
		g.drawHUD(screen)
	case StateGameOver:
		g.drawGameOver(screen)
	}
}

func (g *Game) drawBird(screen *ebiten.Image, sx, sy float64) {
	frameIdx := int(g.bird.wingPhase*2) % 8
	if g.state == StateDying {
		frameIdx = 0
	}
	birdImg := g.birdFrames[frameIdx]
	const bW, bH = 52, 42
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(bW)/2, -float64(bH)/2)
	op.GeoM.Rotate(g.bird.angle)
	op.GeoM.Translate(float64(BirdX)+sx, g.bird.y+sy)
	screen.DrawImage(birdImg, op)
}

// ─────────────────────────────────────────────────────────────
// PIPES
// ─────────────────────────────────────────────────────────────

func drawPipe(screen *ebiten.Image, x, gapCY float64) {
	gapTop := gapCY - PipeGap/2
	gapBot := gapCY + PipeGap/2
	capH := 28.0
	capW := PipeWidth + 14.0
	capX := x - 7.0
	if gapTop > 0 {
		drawPipeBody(screen, x, 0, PipeWidth, gapTop)
		drawPipeCap(screen, capX, gapTop-capH, capW, capH)
	}
	if gapBot < ScreenH {
		drawPipeCap(screen, capX, gapBot, capW, capH)
		drawPipeBody(screen, x, gapBot+capH, PipeWidth, float64(ScreenH)-gapBot-capH)
	}
}

func drawPipeBody(screen *ebiten.Image, x, y, w, h float64) {
	ebitenutil.DrawRect(screen, x, y, 10, h, colPipeHL)
	ebitenutil.DrawRect(screen, x+10, y, w-20, h, colPipeMid)
	ebitenutil.DrawRect(screen, x+w-10, y, 10, h, colPipeDark)
	ebitenutil.DrawRect(screen, x+w-3, y, 3, h, colPipeEdge)
}

func drawPipeCap(screen *ebiten.Image, x, y, w, h float64) {
	ebitenutil.DrawRect(screen, x, y+h-4, w, 4, colPipeEdge)
	ebitenutil.DrawRect(screen, x, y, 12, h-4, colPipeHL)
	ebitenutil.DrawRect(screen, x+12, y, w-24, h-4, colCapTop)
	ebitenutil.DrawRect(screen, x+w-12, y, 12, h-4, colCapBot)
	ebitenutil.DrawRect(screen, x+w-4, y, 4, h-4, colPipeEdge)
	ebitenutil.DrawRect(screen, x, y, w, 3, colPipeHL)
}

// ─────────────────────────────────────────────────────────────
// CLOUDS
// ─────────────────────────────────────────────────────────────

func drawCloud(screen *ebiten.Image, x, y, scale, alpha float64) {
	a := uint8(alpha * 235)
	c := color.RGBA{255, 255, 255, a}
	cs := color.RGBA{200, 225, 240, a}
	s := float32(scale)
	fx, fy := float32(x), float32(y)
	vector.DrawFilledCircle(screen, fx+2*s, fy+6*s, 26*s, cs, true)
	vector.DrawFilledCircle(screen, fx+36*s, fy+6*s, 20*s, cs, true)
	vector.DrawFilledCircle(screen, fx-28*s, fy+6*s, 16*s, cs, true)
	vector.DrawFilledCircle(screen, fx, fy, 26*s, c, true)
	vector.DrawFilledCircle(screen, fx+34*s, fy, 20*s, c, true)
	vector.DrawFilledCircle(screen, fx-28*s, fy, 17*s, c, true)
	vector.DrawFilledCircle(screen, fx+17*s, fy-14*s, 20*s, c, true)
	vector.DrawFilledCircle(screen, fx-10*s, fy-10*s, 16*s, c, true)
	vector.DrawFilledRect(screen, fx-44*s, fy, 98*s, 26*s, c, false)
}

// ─────────────────────────────────────────────────────────────
// MOUNTAINS
// ─────────────────────────────────────────────────────────────

func drawMountains(screen *ebiten.Image, sx, sy float64) {
	drawMountainLayer(screen, sx*0.15, sy*0.15,
		color.RGBA{160, 195, 220, 200}, 0.65,
		[]float64{0, 80, 60, 130, 100, 200, 80, 270, 110, 340, 90, 420, 120, 490})
	drawMountainLayer(screen, sx*0.3, sy*0.3,
		color.RGBA{120, 165, 195, 230}, 0.85,
		[]float64{0, 100, 80, 160, 130, 250, 100, 330, 140, 400, 110, 480})
}

func drawMountainLayer(screen *ebiten.Image, sx, sy float64, clr color.RGBA, hm float64, pts []float64) {
	baseY := float64(ScreenH-GroundH) + sy
	for i := 0; i < len(pts)-2; i += 2 {
		x1, h1 := pts[i]+sx, pts[i+1]*hm
		x2, h2 := pts[i+2]+sx, pts[i+3]*hm
		steps := int(math.Abs(x2-x1)) + 1
		for s := 0; s < steps; s++ {
			t := float64(s) / float64(steps)
			cx := x1 + t*(x2-x1)
			ch := h1 + t*(h2-h1)
			ebitenutil.DrawRect(screen, cx, baseY-ch, 1, ch, clr)
		}
	}
}

// ─────────────────────────────────────────────────────────────
// GROUND
// ─────────────────────────────────────────────────────────────

func drawGround(screen *ebiten.Image, offset, sy float64) {
	groundY := float64(ScreenH-GroundH) + sy
	ebitenutil.DrawRect(screen, 0, groundY+18, ScreenW, float64(GroundH-18), colDirtLight)
	ebitenutil.DrawRect(screen, 0, groundY+36, ScreenW, float64(GroundH-36), colDirtMid)
	ebitenutil.DrawRect(screen, 0, groundY+54, ScreenW, float64(GroundH-54), colDirtDark)
	tileW := 48.0
	off := math.Mod(offset, tileW)
	for x := off - tileW; x < float64(ScreenW)+tileW; x += tileW {
		ebitenutil.DrawRect(screen, x, groundY+20, 1.5, float64(GroundH-20), colDirtMid)
		ebitenutil.DrawRect(screen, x+14, groundY+28, 5, 4, colDirtDark)
		ebitenutil.DrawRect(screen, x+28, groundY+44, 6, 3, colDirtMid)
	}
	ebitenutil.DrawRect(screen, 0, groundY, ScreenW, 18, colGrass)
	ebitenutil.DrawRect(screen, 0, groundY, ScreenW, 5, colGrassTop)
	for x := off - tileW; x < float64(ScreenW)+tileW; x += tileW {
		for j := 0; j < 3; j++ {
			tx := x + float64(j)*16 + 4
			ebitenutil.DrawRect(screen, tx, groundY-4, 3, 6, colGrassTop)
			ebitenutil.DrawRect(screen, tx+4, groundY-5, 3, 7, colGrass)
			ebitenutil.DrawRect(screen, tx+8, groundY-3, 3, 5, colGrassTop)
		}
	}
}

// ─────────────────────────────────────────────────────────────
// UI
// ─────────────────────────────────────────────────────────────

func (g *Game) drawMenu(screen *ebiten.Image) {
	// Title card
	drawPanel(screen, ScreenW/2-170, 105, 340, 120, color.RGBA{0, 40, 80, 210})
	drawTextShadow(screen, "FLAPPY", ScreenW/2, 158, fontFaceXXL, colGold, color.RGBA{140, 80, 0, 255})
	drawTextShadow(screen, "BIRD PRO", ScreenW/2, 202, fontFaceLarge, colWhite, color.RGBA{0, 0, 0, 200})

	// Blink prompt
	if (g.tick/30)%2 == 0 {
		drawPanel(screen, ScreenW/2-160, ScreenH-195, 320, 48, color.RGBA{255, 255, 255, 35})
		drawTextShadow(screen, "PRESS SPACE", ScreenW/2, ScreenH-163, fontFaceMid, colWhite, colBlack)
	}

	// Best
	if g.best > 0 {
		drawPanel(screen, ScreenW/2-90, ScreenH-130, 180, 40, color.RGBA{0, 0, 0, 140})
		drawTextShadow(screen, fmt.Sprintf("BEST %d", g.best), ScreenW/2, ScreenH-103, fontFaceMid, colGold, colBlack)
	}
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	scoreStr := fmt.Sprintf("%d", g.score)
	w, _ := text.Measure(scoreStr, fontFaceXL, 0)
	padX := w/2 + 30
	drawPanel(screen, float64(ScreenW)/2-padX, 24, padX*2, 58, color.RGBA{0, 0, 0, 160})
	drawTextShadow(screen, scoreStr, ScreenW/2, 70, fontFaceXL, colWhite, colBlack)
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	// Overlay
	overlay := ebiten.NewImage(ScreenW, ScreenH)
	overlay.Fill(color.RGBA{0, 0, 0, 120})
	screen.DrawImage(overlay, nil)

	panelX := float64(ScreenW)/2 - 185
	panelY := float64(ScreenH)/2 - 210
	drawPanel(screen, panelX, panelY, 370, 420, color.RGBA{15, 40, 70, 240})

	// GAME OVER
	drawTextShadow(screen, "GAME OVER", ScreenW/2, int(panelY)+70,
		fontFaceLarge, color.RGBA{255, 80, 80, 255}, color.RGBA{100, 0, 0, 255})

	// Divider
	ebitenutil.DrawRect(screen, panelX+20, panelY+90, 330, 2, color.RGBA{255, 255, 255, 60})

	// SCORE label + value
	drawTextShadow(screen, "SCORE", ScreenW/2, int(panelY)+130, fontFaceSmall,
		color.RGBA{180, 220, 255, 200}, colBlack)
	drawTextShadow(screen, fmt.Sprintf("%d", g.score), ScreenW/2, int(panelY)+185,
		fontFaceXXL, colWhite, colBlack)

	// Medal
	if g.score >= 1 {
		drawMedal(screen, ScreenW/2, int(panelY)+240, g.score)
	}

	// Divider 2
	ebitenutil.DrawRect(screen, panelX+20, panelY+270, 330, 2, color.RGBA{255, 255, 255, 60})

	// BEST
	drawTextShadow(screen, fmt.Sprintf("BEST  %d", g.best), ScreenW/2, int(panelY)+310,
		fontFaceMid, colGold, colBlack)

	// NEW BEST
	if g.score > 0 && g.score == g.best && (g.tick/20)%2 == 0 {
		drawPanel(screen, float64(ScreenW)/2-80, panelY+320, 160, 30, color.RGBA{255, 200, 0, 200})
		drawTextShadow(screen, "NEW BEST!", ScreenW/2, int(panelY)+344, fontFaceSmall,
			color.RGBA{80, 30, 0, 255}, colBlack)
	}

	// Retry
	if (g.tick/28)%2 == 0 && g.deathTimer == 0 {
		drawPanel(screen, float64(ScreenW)/2-145, panelY+360, 290, 42, color.RGBA{255, 255, 255, 30})
		drawTextShadow(screen, "SPACE TO RETRY", ScreenW/2, int(panelY)+388,
			fontFaceSmall, colWhite, colBlack)
	}
}

// ─────────────────────────────────────────────────────────────
// MEDAL
// ─────────────────────────────────────────────────────────────

func drawMedal(screen *ebiten.Image, cx, cy int, score int) {
	var outer, inner color.RGBA
	var label string
	switch {
	case score >= 40:
		outer = color.RGBA{180, 180, 200, 255}
		inner = color.RGBA{220, 220, 240, 255}
		label = "PLAT"
	case score >= 20:
		outer = color.RGBA{210, 160, 20, 255}
		inner = color.RGBA{255, 210, 60, 255}
		label = "GOLD"
	case score >= 10:
		outer = color.RGBA{160, 160, 170, 255}
		inner = color.RGBA{200, 200, 215, 255}
		label = "SILV"
	default:
		outer = color.RGBA{160, 90, 40, 255}
		inner = color.RGBA{200, 130, 70, 255}
		label = "BRNZ"
	}
	vector.DrawFilledCircle(screen, float32(cx), float32(cy), 22, outer, true)
	vector.DrawFilledCircle(screen, float32(cx), float32(cy), 17, inner, true)
	drawTextShadow(screen, label, cx, cy+5, fontFaceSmall, color.RGBA{60, 30, 0, 255}, colBlack)
}

// ─────────────────────────────────────────────────────────────
// TEXT HELPERS
// ─────────────────────────────────────────────────────────────

func drawTextShadow(screen *ebiten.Image, str string, cx, cy int, face *text.GoTextFace, clr, shadow color.RGBA) {
	w, h := text.Measure(str, face, 0)
	x := float64(cx) - w/2
	y := float64(cy) - h/2

	// Shadow
	op := &text.DrawOptions{}
	op.GeoM.Translate(x+2, y+2)
	op.ColorScale.ScaleWithColor(shadow)
	text.Draw(screen, str, face, op)

	// Main
	op2 := &text.DrawOptions{}
	op2.GeoM.Translate(x, y)
	op2.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, str, face, op2)
}

// ─────────────────────────────────────────────────────────────
// PANEL
// ─────────────────────────────────────────────────────────────

func drawPanel(screen *ebiten.Image, x, y, w, h float64, clr color.RGBA) {
	r := float32(12)
	fx, fy, fw, fh := float32(x), float32(y), float32(w), float32(h)
	vector.DrawFilledRect(screen, fx+r, fy, fw-2*r, fh, clr, false)
	vector.DrawFilledRect(screen, fx, fy+r, fw, fh-2*r, clr, false)
	vector.DrawFilledCircle(screen, fx+r, fy+r, r, clr, true)
	vector.DrawFilledCircle(screen, fx+fw-r, fy+r, r, clr, true)
	vector.DrawFilledCircle(screen, fx+r, fy+fh-r, r, clr, true)
	vector.DrawFilledCircle(screen, fx+fw-r, fy+fh-r, r, clr, true)
}

// ─────────────────────────────────────────────────────────────
// UTILITY
// ─────────────────────────────────────────────────────────────

func lerp8(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + t*float64(int(b)-int(a)))
}
