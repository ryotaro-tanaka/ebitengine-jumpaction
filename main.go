package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

const (
	screenWidth  = 640
	screenHeight = 360
	groundY      = 300

	playerX      = 120
	playerW      = 16
	playerH      = 24
	jumpVelocity = -10.5
	gravity      = 0.5

	obstacleW = 16
	obstacleH = 28

	baseScrollSpeed = 4.0
	maxScrollSpeed  = 11.0
)

type gameState int

const (
	stateTitle gameState = iota
	statePlaying
	stateGameOver
)

type obstacle struct {
	x float64
	y float64
	w float64
	h float64
}

type Game struct {
	state gameState

	playerY  float64
	playerV  float64
	onGround bool

	obstacles  []obstacle
	spawnTimer int

	scrollSpeed float64
	score       int

	rng *rand.Rand
}

func NewGame() *Game {
	g := &Game{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	g.reset()
	g.state = stateTitle
	return g
}

func (g *Game) reset() {
	g.playerY = groundY - playerH
	g.playerV = 0
	g.onGround = true
	g.obstacles = g.obstacles[:0]
	g.spawnTimer = g.nextSpawnFrames()
	g.scrollSpeed = baseScrollSpeed
	g.score = 0
}

func (g *Game) nextSpawnFrames() int {
	// スピードが上がるほど障害物間の距離（ピクセル）を広げる。
	// これにより高速でも「先に押しておけば避けられる」余地を残す。
	speedDelta := g.scrollSpeed - baseScrollSpeed
	minGapPx := 180.0 + speedDelta*120.0
	maxGapPx := 320.0 + speedDelta*170.0
	if maxGapPx < minGapPx+40 {
		maxGapPx = minGapPx + 40
	}

	gapPx := minGapPx + g.rng.Float64()*(maxGapPx-minGapPx)
	frames := int(gapPx / g.scrollSpeed)
	if frames < 22 {
		frames = 22
	}
	return frames
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	switch g.state {
	case stateTitle:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.reset()
			g.state = statePlaying
		}
	case statePlaying:
		g.updatePlaying()
	case stateGameOver:
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.reset()
			g.state = statePlaying
		}
	}

	return nil
}

func (g *Game) updatePlaying() {
	if ebiten.IsKeyPressed(ebiten.KeySpace) && g.onGround {
		g.playerV = jumpVelocity
		g.onGround = false
	}

	g.updateDifficulty()

	g.playerV += gravity
	g.playerY += g.playerV

	groundTop := float64(groundY - playerH)
	if g.playerY >= groundTop {
		g.playerY = groundTop
		g.playerV = 0
		g.onGround = true
	}

	g.spawnTimer--
	if g.spawnTimer <= 0 {
		g.obstacles = append(g.obstacles, obstacle{
			x: screenWidth + 10,
			y: groundY - obstacleH,
			w: obstacleW,
			h: obstacleH,
		})
		g.spawnTimer = g.nextSpawnFrames()
	}

	for i := 0; i < len(g.obstacles); {
		g.obstacles[i].x -= g.scrollSpeed
		if g.obstacles[i].x+g.obstacles[i].w < 0 {
			g.obstacles = append(g.obstacles[:i], g.obstacles[i+1:]...)
			continue
		}
		i++
	}

	if g.collides() {
		g.state = stateGameOver
	}

	g.score++
}

func (g *Game) updateDifficulty() {
	// 経過時間（score）に応じて少しずつ加速。
	targetSpeed := baseScrollSpeed + float64(g.score)*0.0008
	if targetSpeed > maxScrollSpeed {
		targetSpeed = maxScrollSpeed
	}
	g.scrollSpeed = targetSpeed
}

func (g *Game) collides() bool {
	px := float64(playerX)
	py := g.playerY
	for _, ob := range g.obstacles {
		if px < ob.x+ob.w && px+playerW > ob.x && py < ob.y+ob.h && py+playerH > ob.y {
			return true
		}
	}
	return false
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 240, G: 245, B: 255, A: 255})

	// ground
	ebitenutil.DrawRect(screen, 0, groundY, screenWidth, screenHeight-groundY, color.RGBA{R: 50, G: 60, B: 70, A: 255})
	ebitenutil.DrawLine(screen, 0, groundY, screenWidth, groundY, color.White)

	for _, ob := range g.obstacles {
		ebitenutil.DrawRect(screen, ob.x, ob.y, ob.w, ob.h, color.RGBA{R: 200, G: 60, B: 60, A: 255})
		text.Draw(screen, "#", basicfont.Face7x13, int(ob.x)+3, int(ob.y)+17, color.White)
	}

	ebitenutil.DrawRect(screen, playerX, g.playerY, playerW, playerH, color.RGBA{R: 60, G: 120, B: 220, A: 255})
	text.Draw(screen, "@", basicfont.Face7x13, playerX+4, int(g.playerY)+16, color.White)

	text.Draw(screen, fmt.Sprintf("Score: %d", g.score), basicfont.Face7x13, 12, 24, color.Black)
	text.Draw(screen, fmt.Sprintf("Speed: %.2f", g.scrollSpeed), basicfont.Face7x13, 12, 42, color.Black)

	switch g.state {
	case stateTitle:
		text.Draw(screen, "Charihashi Run (MVP)", basicfont.Face7x13, 240, 140, color.Black)
		text.Draw(screen, "Press Space to Start", basicfont.Face7x13, 242, 170, color.Black)
		text.Draw(screen, "Space: Jump  Esc: Quit", basicfont.Face7x13, 230, 198, color.Black)
	case stateGameOver:
		text.Draw(screen, "Game Over", basicfont.Face7x13, 285, 140, color.Black)
		text.Draw(screen, "Press R to Retry", basicfont.Face7x13, 260, 170, color.Black)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Charihashi Run")
	if err := ebiten.RunGame(NewGame()); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
