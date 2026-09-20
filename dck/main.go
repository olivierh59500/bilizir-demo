// Package bilizir implements the Bilizir demo for desktop and mobile frontends.
package bilizir

import originalassets "bilizir-demo"

import (
	"bytes"

	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	kit "github.com/olivierh59500/democonstructionkit"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/scrolling"

	"bilizir-demo/dck/internal/ymaudio"
)

const (
	screenWidth  = 800
	screenHeight = 600
	nbCubes      = 12
	scrollHeight = 64 // Increased from 50 to 64 for 2x font
	scrollSpeed  = 4.0
	sampleRate   = 44100
	copperBars   = screenHeight / 2
)

// Embed all assets
var logoImg = originalassets.
	DCKAssetLogoImg()

var barsImg = originalassets.
	DCKAssetBarsImg()

var scrollFontData = originalassets.
	DCKAssetScrollFontData()

var musicData = originalassets.

	// ScrollText manages the scrolling text with deformation effects
	DCKAssetMusicData()

type ScrollText struct {
	renderer   *scrolling.Scrolling
	x          float64
	workBuffer *ebiten.Image
	warp       *composite.StripWarp
}

// Cube3D represents a rotating 3D cube
type Cube3D struct {
	angleX       float64
	angleY       float64
	angleZ       float64
	size         float64
	drawVertices [cubeFaceCount * cubeVerticesPerFace]ebiten.Vertex
	drawIndices  [cubeFaceCount * cubeIndicesPerFace]uint16
}

const (
	cubeFaceCount       = 6
	cubeVerticesPerFace = 20 // Four for the face and four for each edge.
	cubeIndicesPerFace  = 30 // Two triangles for the face and two per edge.
)

var cubeFaces = [cubeFaceCount][4]int{
	{0, 1, 2, 3}, // Back
	{4, 5, 6, 7}, // Front
	{0, 1, 5, 4}, // Bottom
	{2, 3, 7, 6}, // Top
	{0, 3, 7, 4}, // Left
	{1, 2, 6, 5}, // Right
}

var cubeFaceColors = [cubeFaceCount]color.RGBA{
	{R: 255, G: 80, B: 160, A: 255},
	{R: 255, G: 120, B: 200, A: 255},
	{R: 200, G: 60, B: 140, A: 255},
	{R: 255, G: 100, B: 180, A: 255},
	{R: 220, G: 80, B: 160, A: 255},
	{R: 255, G: 140, B: 200, A: 255},
}

type point2D struct {
	x float32
	y float32
}

type faceDepth struct {
	index int
	depth float64
}

// NewCube3D creates a new 3D cube
func NewCube3D(size float64) *Cube3D {
	return &Cube3D{
		size: size,
	}
}

// Rotate updates the cube rotation angles
func (c *Cube3D) Rotate(dx, dy, dz float64) {
	c.angleX += dx
	c.angleY += dy
	c.angleZ += dz
}

// Project3D projects 3D coordinates to 2D
func project3D(x, y, z float64) (float64, float64) {
	// Simple perspective projection
	perspective := 200.0
	factor := perspective / (perspective + z)
	return x * factor, y * factor
}

// Draw draws the 3D cube at the specified position
func (c *Cube3D) Draw(screen, solidImage *ebiten.Image, centerX, centerY float64) {
	vertices := [8][3]float64{
		{-c.size / 2, -c.size / 2, -c.size / 2}, // 0
		{c.size / 2, -c.size / 2, -c.size / 2},  // 1
		{c.size / 2, c.size / 2, -c.size / 2},   // 2
		{-c.size / 2, c.size / 2, -c.size / 2},  // 3
		{-c.size / 2, -c.size / 2, c.size / 2},  // 4
		{c.size / 2, -c.size / 2, c.size / 2},   // 5
		{c.size / 2, c.size / 2, c.size / 2},    // 6
		{-c.size / 2, c.size / 2, c.size / 2},   // 7
	}

	var rotated [8][3]float64
	var projected [8]point2D
	sinX, cosX := math.Sincos(c.angleX)
	sinY, cosY := math.Sincos(c.angleY)
	sinZ, cosZ := math.Sincos(c.angleZ)
	for i, v := range vertices {
		x, y, z := v[0], v[1], v[2]

		y1 := y*cosX - z*sinX
		z1 := y*sinX + z*cosX
		y, z = y1, z1

		x1 := x*cosY + z*sinY
		z2 := -x*sinY + z*cosY
		x, z = x1, z2

		x2 := x*cosZ - y*sinZ
		y2 := x*sinZ + y*cosZ
		x, y = x2, y2

		rotated[i] = [3]float64{x, y, z}
		x2d, y2d := project3D(x, y, z)
		projected[i] = point2D{float32(centerX + x2d), float32(centerY + y2d)}
	}

	var depths [cubeFaceCount]faceDepth
	for i, face := range cubeFaces {
		centerZ := 0.0
		for _, vi := range face {
			centerZ += rotated[vi][2]
		}
		depths[i] = faceDepth{index: i, depth: centerZ / 4}
	}

	for i := 0; i < len(depths)-1; i++ {
		for j := i + 1; j < len(depths); j++ {
			if depths[i].depth > depths[j].depth {
				depths[i], depths[j] = depths[j], depths[i]
			}
		}
	}

	drawVertices := c.drawVertices[:0]
	drawIndices := c.drawIndices[:0]
	for _, fd := range depths {
		face := cubeFaces[fd.index]
		faceColor := cubeFaceColors[fd.index]
		points := [4]point2D{
			projected[face[0]], projected[face[1]], projected[face[2]], projected[face[3]],
		}
		drawVertices, drawIndices = appendColoredQuad(drawVertices, drawIndices, points, faceColor)

		edgeColor := color.RGBA{
			R: faceColor.R * 3 / 4,
			G: faceColor.G * 3 / 4,
			B: faceColor.B * 3 / 4,
			A: 255,
		}
		for i := 0; i < 4; i++ {
			drawVertices, drawIndices = appendColoredLine(
				drawVertices, drawIndices, points[i], points[(i+1)%4], 1, edgeColor,
			)
		}
	}

	screen.DrawTriangles(drawVertices, drawIndices, solidImage, nil)
}

func appendColoredQuad(vertices []ebiten.Vertex, indices []uint16, points [4]point2D, clr color.RGBA) ([]ebiten.Vertex, []uint16) {
	base := uint16(len(vertices))
	r := float32(clr.R) / 255
	g := float32(clr.G) / 255
	b := float32(clr.B) / 255
	a := float32(clr.A) / 255
	for _, point := range points {
		vertices = append(vertices, ebiten.Vertex{
			DstX: point.x, DstY: point.y,
			SrcX: 1, SrcY: 1,
			ColorR: r, ColorG: g, ColorB: b, ColorA: a,
		})
	}
	indices = append(indices, base, base+1, base+2, base, base+2, base+3)
	return vertices, indices
}

func appendColoredLine(vertices []ebiten.Vertex, indices []uint16, from, to point2D, width float32, clr color.RGBA) ([]ebiten.Vertex, []uint16) {
	dx := to.x - from.x
	dy := to.y - from.y
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length == 0 {
		return vertices, indices
	}
	halfWidth := width / (2 * length)
	nx, ny := -dy*halfWidth, dx*halfWidth
	points := [4]point2D{
		{x: from.x + nx, y: from.y + ny},
		{x: to.x + nx, y: to.y + ny},
		{x: to.x - nx, y: to.y - ny},
		{x: from.x - nx, y: from.y - ny},
	}
	return appendColoredQuad(vertices, indices, points, clr)
}

// Game represents the main game state
type Game struct {
	logoOptions            LogoWarpOptions
	logoMargin             int
	deformation            composite.StripWarpConfig
	logoWarp               *composite.StripWarp
	logoBuffer             *ebiten.Image
	logoDeformationEnabled bool
	// Demo assets
	cubes      [nbCubes]*Cube3D
	spritePos  [nbCubes]float64
	logo       *ebiten.Image
	logoPos    float64
	logoWidth  int
	bars       *ebiten.Image
	solidImage *ebiten.Image

	// Copper bars animation
	rasterBatch *composite.QuadBatch
	copperSin   []int
	cnt         int
	cnt2        int

	// Scroll integration
	scrollText *ScrollText
	scrollX    []float64
	scrollXMod int
	vbl        int
	offsetScr  float64
	scrollFont *ebiten.Image

	// Audio
	audioContext *audio.Context
	audioPlayer  *audio.Player
	ymPlayer     *ymaudio.Player

	// Speed control
	speedMultiplier float64

	// Initialization flag
	initialized bool
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		logoOptions:            DefaultLogoWarpOptions(),
		logoMargin:             deformationMargin,
		speedMultiplier:        1.0,
		cnt:                    0,
		cnt2:                   0,
		rasterBatch:            composite.NewQuadBatch(copperBars),
		logoDeformationEnabled: true,
	}

	// Initialize scroll deformation data
	g.initScrollX()
	g.configureDeformation()

	// Initialize copper bars sine table
	g.initCopperSin()

	return g
}

// initCopperSin initializes the sine table for copper bars animation
func (g *Game) initCopperSin() {
	// This is the sine table from the JavaScript code
	g.copperSin = []int{
		264, 264, 268, 272, 276, 280, 280, 284, 288, 292, 296, 296, 300, 304, 308, 312, 312, 316, 320, 324, 328, 328, 332, 336, 340, 340, 344, 348, 352, 352, 356, 360, 364, 364, 368, 372, 376, 376, 380, 384, 388, 388, 392, 396, 396, 400, 404, 404, 408, 412, 412, 416, 420, 420, 424, 428, 428, 432, 436, 436, 440, 440, 444, 448, 448, 452, 452, 456, 456, 460, 460, 464, 464, 468, 472, 472, 472, 476, 476, 480, 480, 484, 484, 488, 488, 488, 492, 492, 496, 496, 496, 500, 500, 500, 504, 504, 504, 508, 508, 508, 512, 512, 512, 512, 516, 516, 516, 516, 520, 520, 520, 520, 520, 520, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 520, 520, 520, 520, 520, 520, 516, 516, 516, 516, 512, 512, 512, 512, 508, 508, 508, 508, 504, 504, 504, 500, 500, 500, 496, 496, 492, 492, 492, 488, 488, 484, 484, 480, 480, 480, 476, 476, 472, 472, 468, 468, 464, 464, 460, 456, 456, 452, 452, 448, 448, 444, 444, 440, 436, 436, 432, 428, 428, 424, 424, 420, 416, 416, 412, 408, 408, 404, 400, 400, 396, 392, 388, 388, 384, 380, 380, 376, 372, 368, 368, 364, 360, 356, 356, 352, 348, 344, 344, 340, 336, 332, 328, 328, 324, 320, 316, 316, 312, 308, 304, 300, 300, 296, 292, 288, 284, 284, 280, 276, 272, 268, 264, 264, 264, 260, 256, 252, 252, 248, 244, 240, 236, 236, 232, 228, 224, 220, 220, 216, 212, 208, 204, 204, 200, 196, 192, 192, 188, 184, 180, 176, 176, 172, 168, 164, 164, 160, 156, 152, 152, 148, 144, 144, 140, 136, 132, 132, 128, 124, 124, 120, 116, 116, 112, 108, 108, 104, 100, 100, 96, 96, 92, 88, 88, 84, 84, 80, 76, 76, 72, 72, 68, 68, 64, 64, 60, 60, 56, 56, 52, 52, 48, 48, 44, 44, 40, 40, 40, 36, 36, 32, 32, 32, 28, 28, 28, 24, 24, 24, 20, 20, 20, 16, 16, 16, 16, 12, 12, 12, 12, 12, 8, 8, 8, 8, 8, 8, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 8, 8, 8, 8, 8, 8, 12, 12, 12, 12, 12, 16, 16, 16, 20, 20, 20, 20, 24, 24, 24, 28, 28, 28, 32, 32, 36, 36, 36, 40, 40, 44, 44, 44, 48, 48, 52, 52, 56, 56, 60, 60, 64, 64, 68, 68, 72, 72, 76, 80, 80, 84, 84, 88, 92, 92, 96, 96, 100, 104, 104, 108, 112, 112, 116, 120, 120, 124, 128, 128, 132, 136, 136, 140, 144, 148, 148, 152, 156, 156, 160, 164, 168, 168, 172, 176, 180, 180, 184, 188, 192, 196, 196, 200, 204, 208, 212, 212, 216, 220, 224, 224, 228, 232, 236, 240, 244, 244, 248, 252, 256, 260, 260, 264, 264, 268, 272, 276, 280, 280, 284, 288, 292, 296, 296, 300, 304, 308, 312, 312, 316, 320, 324, 328, 328, 332, 336, 340, 340, 344, 348, 352, 352, 356, 360, 364, 364, 368, 372, 376, 376, 380, 384, 388, 388, 392, 396, 396, 400, 404, 404, 408, 412, 412, 416, 420, 420, 424, 428, 428, 432, 436, 436, 440, 440, 444, 448, 448, 452, 452, 456, 456, 460, 460, 464, 464, 468, 472, 472, 472, 476, 476, 480, 480, 484, 484, 488, 488, 488, 492, 492, 496, 496, 496, 500, 500, 500, 504, 504, 504, 508, 508, 508, 512, 512, 512, 512, 516, 516, 516, 516, 520, 520, 520, 520, 520, 520, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 520, 520, 520, 520, 520, 520, 516, 516, 516, 516, 512, 512, 512, 512, 508, 508, 508, 508, 504, 504, 504, 500, 500, 500, 496, 496, 492, 492, 492, 488, 488, 484, 484, 480, 480, 480, 476, 476, 472, 472, 468, 468, 464, 464, 460, 456, 456, 452, 452, 448, 448, 444, 444, 440, 436, 436, 432, 428, 428, 424, 424, 420, 416, 416, 412, 408, 408, 404, 400, 400, 396, 392, 388, 388, 384, 380, 380, 376, 372, 368, 368, 364, 360, 356, 356, 352, 348, 344, 344, 340, 336, 332, 328, 328, 324, 320, 316, 316, 312, 308, 304, 300, 300, 296, 292, 288, 284, 284, 280, 276, 272, 268, 264, 264, 264, 260, 256, 252, 252, 248, 244, 240, 236, 236, 232, 228, 224, 220, 220, 216, 212, 208, 204, 204, 200, 196, 192, 192, 188, 184, 180, 176, 176, 172, 168, 164, 164, 160, 156, 152, 152, 148, 144, 144, 140, 136, 132, 132, 128, 124, 124, 120, 116, 116, 112, 108, 108, 104, 100, 100, 96, 96, 92, 88, 88, 84, 84, 80, 76, 76, 72, 72, 68, 68, 64, 64, 60, 60, 56, 56, 52, 52, 48, 48, 44, 44, 40, 40, 40, 36, 36, 32, 32, 32, 28, 28, 28, 24, 24, 24, 20, 20, 20, 16, 16, 16, 16, 12, 12, 12, 12, 12, 8, 8, 8, 8, 8, 8, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 8, 8, 8, 8, 8, 8, 12, 12, 12, 12, 12, 16, 16, 16, 20, 20, 20, 20, 24, 24, 24, 28, 28, 28, 32, 32, 36, 36, 36, 40, 40, 44, 44, 44, 48, 48, 52, 52, 56, 56, 60, 60, 64, 64, 68, 68, 72, 72, 76, 80, 80, 84, 84, 88, 92, 92, 96, 96, 100, 104, 104, 108, 112, 112, 116, 120, 120, 124, 128, 128, 132, 136, 136, 140, 144, 148, 148, 152, 156, 156, 160, 164, 168, 168, 172, 176, 180, 180, 184, 188, 192, 196, 196, 200, 204, 208, 212, 212, 216, 220, 224, 224, 228, 232, 236, 240, 244, 244, 248, 252, 256, 260, 260,
	}
}

// initScrollX initializes the scroll deformation wave patterns
func (g *Game) initScrollX() {
	const waveLength = 389 + 120 + 68 + 389 + 36 + 189
	g.scrollX = make([]float64, 0, waveLength)

	// First wave pattern
	stp1 := 7.0 / 180.0 * math.Pi
	stp2 := 3.0 / 180.0 * math.Pi
	for i := 0; i < 389; i++ {
		x := 20*math.Sin(float64(i)*stp1) + 30*math.Cos(float64(i)*stp2)
		g.scrollX = append(g.scrollX, x)
	}

	// Second wave pattern
	stp1 = 72.0 / 180.0 * math.Pi
	for i := 0; i < 120; i++ {
		x := 4 * math.Sin(float64(i)*stp1)
		g.scrollX = append(g.scrollX, x)
	}

	// Third wave pattern
	stp1 = 8.0 / 180.0 * math.Pi
	for i := 0; i < 68; i++ {
		x := 40 * math.Sin(float64(i)*stp1)
		g.scrollX = append(g.scrollX, x)
	}

	// Repeat first pattern
	stp1 = 7.0 / 180.0 * math.Pi
	stp2 = 3.0 / 180.0 * math.Pi
	for i := 0; i < 389; i++ {
		x := 20*math.Sin(float64(i)*stp1) + 30*math.Cos(float64(i)*stp2)
		g.scrollX = append(g.scrollX, x)
	}

	// Small wave
	stp1 = 72.0 / 180.0 * math.Pi
	for i := 0; i < 36; i++ {
		x := 4 * math.Sin(float64(i)*stp1)
		g.scrollX = append(g.scrollX, x)
	}

	// Final wave
	stp1 = 8.0 / 180.0 * math.Pi
	for i := 0; i < 189; i++ {
		x := 30 * math.Sin(float64(i)*stp1)
		g.scrollX = append(g.scrollX, x)
	}

	g.scrollXMod = len(g.scrollX)
}

// loadAssets loads all image assets from embedded data
func (g *Game) loadAssets() error {
	var err error

	// Initialize cube positions
	for i := 0; i < nbCubes; i++ {
		g.spritePos[i] = float64(0.15) * float64(i+1)
		// Create cubes with different initial rotations
		g.cubes[i] = NewCube3D(20) // 20 pixel size cubes
		g.cubes[i].angleX = float64(i) * 0.3
		g.cubes[i].angleY = float64(i) * 0.5
		g.cubes[i].angleZ = float64(i) * 0.2
	}

	// Load logo
	img, _, err := image.Decode(bytes.NewReader(logoImg))
	if err != nil {
		return fmt.Errorf("failed to load logo image: %v", err)
	}
	g.logo = ebiten.NewImageFromImage(img)
	g.logoWidth, _ = g.logo.Size()

	// Load bars image
	img, _, err = image.Decode(bytes.NewReader(barsImg))
	if err != nil {
		return fmt.Errorf("failed to load bars image: %v", err)
	}
	g.bars = ebiten.NewImageFromImage(img)
	if _, height := g.bars.Size(); height < 20 {
		return fmt.Errorf("copper bars image height is %d, want at least 20", height)
	}

	// Load scroll font
	img, _, err = image.Decode(bytes.NewReader(scrollFontData))
	if err != nil {
		return fmt.Errorf("failed to load scroll font: %v", err)
	}
	g.scrollFont = ebiten.NewImageFromImage(img)
	g.solidImage = ebiten.NewImage(3, 3)
	g.solidImage.Fill(color.White)

	return g.initLogoDeformation()
}

// initScrollText initializes the scrolling text with soap font
func (g *Game) initScrollText() {
	const text = `      HELLO, BILIZIR FROM DMA IS PROUD TO PRESENT HIS NEW GOLANG/EBITEN INTRO... NOT SO BAD FOR A FEW HOURS OF HARD WORK :)  HI TO ALL MEMBERS OF DMA (COUCOU PHILIPPE ET DIDIER ALORS PAS MAL NON ?), ALL MEMBERS OF THE UNION, ALL DEMOSCENE FANS...   LET'S WRAP...      `
	spec, _ := presets.FindFont("bilizir-demo")
	metrics, err := spec.Build(g.scrollFont.Bounds())
	if err != nil {
		panic(err)
	}
	renderer, err := scrolling.New(scrolling.Config{Text: text, Fonts: map[string]scrolling.Face{"default": {Atlas: g.scrollFont, Metrics: metrics}}})
	if err != nil {
		panic(err)
	}
	warp, err := composite.NewStripWarp(image.Pt(screenWidth, scrollHeight), g.deformation)
	if err != nil {
		panic(err)
	}
	g.scrollText = &ScrollText{renderer: renderer, workBuffer: ebiten.NewImage(screenWidth+1024, scrollHeight), warp: warp}
}

// loadMusic loads and plays the YM music
func (g *Game) loadMusic() error {
	var err error

	// Audio must be initialized after the Android activity and Ebiten view are ready.
	g.audioContext = audio.NewContext(sampleRate)

	// Create YM player
	g.ymPlayer, err = ymaudio.New(musicData, sampleRate, true)
	if err != nil {
		return fmt.Errorf("failed to create YM player: %w", err)
	}

	// Create audio player
	g.audioPlayer, err = g.audioContext.NewPlayer(g.ymPlayer)
	if err != nil {
		g.ymPlayer.Close()
		g.ymPlayer = nil
		return fmt.Errorf("failed to create audio player: %w", err)
	}

	g.audioPlayer.Play()
	return nil
}

// Init initializes the game
func (g *Game) Init() error {
	if g.initialized {
		return nil
	}

	// Load all assets
	if err := g.loadAssets(); err != nil {
		return err
	}

	// Initialize scrolling text
	g.initScrollText()

	// Load music
	if err := g.loadMusic(); err != nil {
		log.Printf("Failed to load music: %v", err)
		// Continue without music
	}

	g.initialized = true
	return nil
}

// Update updates the game state
func (g *Game) Update() error {
	if !g.initialized {
		return g.Init()
	}

	// Handle input for volume control
	if g.ymPlayer != nil {
		if ebiten.IsKeyPressed(ebiten.KeyUp) {
			vol := g.ymPlayer.Volume() + 0.01
			if vol > 1.0 {
				vol = 1.0
			}
			g.ymPlayer.SetVolume(vol)
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) {
			vol := g.ymPlayer.Volume() - 0.01
			if vol < 0 {
				vol = 0
			}
			g.ymPlayer.SetVolume(vol)
		}
	}

	// Speed control with +/- keys
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		g.speedMultiplier += 0.1
		if g.speedMultiplier > 2.0 {
			g.speedMultiplier = 2.0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		g.speedMultiplier -= 0.1
		if g.speedMultiplier < 0.5 {
			g.speedMultiplier = 0.5
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		g.SetLogoDeformation(!g.logoDeformationEnabled)
	}

	// Update copper bars animation
	g.cnt = (g.cnt + 3) & 0x3ff
	g.cnt2 = (g.cnt2 - 5) & 0x3ff

	// Update logo position
	g.logoPos += 0.05 * g.speedMultiplier

	// Update ball sprites and cube rotations
	for i := 0; i < nbCubes; i++ {
		g.spritePos[i] += 0.04 * g.speedMultiplier

		// Update cube rotations
		g.cubes[i].Rotate(
			0.02*g.speedMultiplier*(1+float64(i)*0.1),
			0.03*g.speedMultiplier*(1+float64(i)*0.15),
			0.01*g.speedMultiplier*(1+float64(i)*0.05),
		)
	}

	// Update scroll text
	g.scrollText.x -= scrollSpeed * g.speedMultiplier
	// Adjusted for 2x font scale
	textWidth := g.scrollText.renderer.Length() * 2
	if g.scrollText.x < -textWidth {
		g.scrollText.x = float64(screenWidth)
	}

	// Update animation counters
	g.vbl++
	g.offsetScr += 0.1 * g.speedMultiplier

	return nil
}

// drawCopperBars draws the animated copper bars effect
func (g *Game) drawCopperBars(screen *ebiten.Image) {
	if g.bars == nil {
		return
	}
	w := g.bars.Bounds().Dx()
	g.rasterBatch.Begin(screen, g.bars)
	for i := 0; i < copperBars; i++ {
		value := g.copperSin[(g.cnt+i*7)&0x3ff] + g.copperSin[(g.cnt2+i*10)&0x3ff] + 60
		y := i * 2
		cc := (i * 2) % 20
		g.rasterBatch.Rect(image.Rect(0, cc, w, cc+2), float32(value>>1), float32(y), float32(w), float32(screenHeight-y))
	}
	g.rasterBatch.Flush()
}

// drawLogo draws the animated DMA logo
func (g *Game) drawLogo(screen *ebiten.Image) {
	if g.logoDeformationEnabled {
		g.drawWarpedLogo(screen)
		return
	}
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-g.logoWidth)/2+math.Sin(g.logoPos)*float64(screenWidth-g.logoWidth)/2, 0)
	composite.Instance{Image: g.logo, Options: op}.Draw(screen)
}

// drawCubes draws the rotating 3D cubes
func (g *Game) drawCubes(screen *ebiten.Image) {
	for i := 0; i < nbCubes; i++ {
		xPos := float64((screenWidth-40)/2) + (float64((screenWidth-40)/2) * math.Sin(g.spritePos[i]))
		yPos := 186 + (84 * math.Cos(g.spritePos[i]*2.5))

		// Draw the 3D cube
		g.cubes[i].Draw(screen, g.solidImage, xPos, yPos)
	}
}

// drawScrollText draws the TCB-style scrolling text with deformation
func (g *Game) drawScrollText(screen *ebiten.Image) {
	st := g.scrollText
	st.workBuffer.Clear()
	state := scrolling.IdentityState()
	state.X = st.x
	state.ScaleX = 2
	state.ScaleY = 2
	state.Map = func(sample scrolling.Sample, op *ebiten.DrawImageOptions) bool {
		return sample.X > -64 && sample.X < float64(st.workBuffer.Bounds().Dx())
	}
	st.renderer.DrawAt(st.workBuffer, state)
	frame := kit.Frame{Tick: uint64(g.vbl)}
	st.warp.DrawAt(screen, st.workBuffer, frame, 0, float64(screenHeight-140))
}

// Draw draws the entire demo
func (g *Game) Draw(screen *ebiten.Image) {
	if !g.initialized {
		return
	}

	// Clear screen with black background
	screen.Fill(color.Black)

	// Draw copper bars first (background)
	g.drawCopperBars(screen)

	// Draw logo on top
	g.drawLogo(screen)

	// Draw cubes
	g.drawCubes(screen)

	// Draw scrolling text with its deformation effect
	g.drawScrollText(screen)
}

// Layout returns the game's logical screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// Cleanup cleans up resources
func (g *Game) Cleanup() {
	if g.logoWarp != nil {
		g.logoWarp.Close()
	}
	if g.logoBuffer != nil {
		g.logoBuffer.Deallocate()
		g.logoBuffer = nil
	}
	if g.scrollText != nil && g.scrollText.warp != nil {
		g.scrollText.warp.Close()
	}
	if g.audioPlayer != nil {
		g.audioPlayer.Close()
	}
	if g.ymPlayer != nil {
		g.ymPlayer.Close()
	}
}
