package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/olivierh59500/ym-player/pkg/stsound"
)

const (
	screenWidth  = 800
	screenHeight = 600
	nbCubes      = 12
	scrollHeight = 64 // Increased from 50 to 64 for 2x font
	scrollSpeed  = 4.0
	sampleRate   = 44100
)

// Embed all assets
//
//go:embed assets/logo.png
var logoImg []byte

//go:embed assets/bars.png
var barsImg []byte

//go:embed assets/soap-font.png
var scrollFontData []byte

//go:embed assets/music.ym
var musicData []byte

// YMPlayer wraps the YM player for Ebiten audio
type YMPlayer struct {
	player       *stsound.StSound
	sampleRate   int
	buffer       []int16
	mutex        sync.Mutex
	position     int64
	totalSamples int64
	loop         bool
	volume       float64
}

// NewYMPlayer creates a new YM player instance
func NewYMPlayer(data []byte, sampleRate int, loop bool) (*YMPlayer, error) {
	player := stsound.CreateWithRate(sampleRate)

	if err := player.LoadMemory(data); err != nil {
		player.Destroy()
		return nil, fmt.Errorf("failed to load YM data: %w", err)
	}

	player.SetLoopMode(loop)

	info := player.GetInfo()
	totalSamples := int64(info.MusicTimeInMs) * int64(sampleRate) / 1000

	return &YMPlayer{
		player:       player,
		sampleRate:   sampleRate,
		buffer:       make([]int16, 4096),
		totalSamples: totalSamples,
		loop:         loop,
		volume:       0.5,
	}, nil
}

// Read implements io.Reader for audio streaming
func (y *YMPlayer) Read(p []byte) (n int, err error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	samplesNeeded := len(p) / 4
	outBuffer := make([]int16, samplesNeeded*2)

	processed := 0
	for processed < samplesNeeded {
		chunkSize := samplesNeeded - processed
		if chunkSize > len(y.buffer) {
			chunkSize = len(y.buffer)
		}

		if !y.player.Compute(y.buffer[:chunkSize], chunkSize) {
			if !y.loop {
				for i := processed * 2; i < len(outBuffer); i++ {
					outBuffer[i] = 0
				}
				err = io.EOF
				break
			}
		}

		for i := 0; i < chunkSize; i++ {
			sample := int16(float64(y.buffer[i]) * y.volume)
			outBuffer[(processed+i)*2] = sample
			outBuffer[(processed+i)*2+1] = sample
		}

		processed += chunkSize
		y.position += int64(chunkSize)
	}

	buf := make([]byte, 0, len(outBuffer)*2)
	for _, sample := range outBuffer {
		buf = append(buf, byte(sample), byte(sample>>8))
	}

	copy(p, buf)
	n = len(buf)
	if n > len(p) {
		n = len(p)
	}

	return n, err
}

// SetVolume sets the playback volume (0.0 to 1.0)
func (y *YMPlayer) SetVolume(volume float64) {
	y.mutex.Lock()
	defer y.mutex.Unlock()
	y.volume = volume
}

// GetVolume returns the current volume
func (y *YMPlayer) GetVolume() float64 {
	y.mutex.Lock()
	defer y.mutex.Unlock()
	return y.volume
}

// Seek implements io.Seeker
func (y *YMPlayer) Seek(offset int64, whence int) (int64, error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = y.position + offset
	case io.SeekEnd:
		newPos = y.totalSamples + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		newPos = 0
	}
	if newPos > y.totalSamples {
		newPos = y.totalSamples
	}

	y.position = newPos
	return newPos, nil
}

// Close releases resources
func (y *YMPlayer) Close() error {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	if y.player != nil {
		y.player.Destroy()
		y.player = nil
	}
	return nil
}

// ScrollText manages the scrolling text with deformation effects
type ScrollText struct {
	text         string
	x            float64
	fontImage    *ebiten.Image
	charWidth    int
	charHeight   int
	charsPerRow  int
	scrollBuffer *ebiten.Image
	workBuffer   *ebiten.Image
	deformBuffer *ebiten.Image
}

// Cube3D represents a rotating 3D cube
type Cube3D struct {
	angleX float64
	angleY float64
	angleZ float64
	size   float64
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
func (c *Cube3D) Draw(screen *ebiten.Image, centerX, centerY float64) {
	// Define cube vertices in 3D space
	vertices := [][3]float64{
		{-c.size / 2, -c.size / 2, -c.size / 2}, // 0
		{c.size / 2, -c.size / 2, -c.size / 2},  // 1
		{c.size / 2, c.size / 2, -c.size / 2},   // 2
		{-c.size / 2, c.size / 2, -c.size / 2},  // 3
		{-c.size / 2, -c.size / 2, c.size / 2},  // 4
		{c.size / 2, -c.size / 2, c.size / 2},   // 5
		{c.size / 2, c.size / 2, c.size / 2},    // 6
		{-c.size / 2, c.size / 2, c.size / 2},   // 7
	}

	// Define cube faces (indices into vertices array)
	faces := [][4]int{
		{0, 1, 2, 3}, // Back
		{4, 5, 6, 7}, // Front
		{0, 1, 5, 4}, // Bottom
		{2, 3, 7, 6}, // Top
		{0, 3, 7, 4}, // Left
		{1, 2, 6, 5}, // Right
	}

	// Define face colors (pink/magenta tones)
	faceColors := []color.Color{
		color.RGBA{255, 80, 160, 255},  // Hot pink
		color.RGBA{255, 120, 200, 255}, // Light pink
		color.RGBA{200, 60, 140, 255},  // Dark pink
		color.RGBA{255, 100, 180, 255}, // Medium pink
		color.RGBA{220, 80, 160, 255},  // Rose
		color.RGBA{255, 140, 200, 255}, // Pale pink
	}

	// Rotate vertices
	rotated := make([][3]float64, len(vertices))
	for i, v := range vertices {
		x, y, z := v[0], v[1], v[2]

		// Rotate around X axis
		cosX, sinX := math.Cos(c.angleX), math.Sin(c.angleX)
		y1 := y*cosX - z*sinX
		z1 := y*sinX + z*cosX
		y, z = y1, z1

		// Rotate around Y axis
		cosY, sinY := math.Cos(c.angleY), math.Sin(c.angleY)
		x1 := x*cosY + z*sinY
		z2 := -x*sinY + z*cosY
		x, z = x1, z2

		// Rotate around Z axis
		cosZ, sinZ := math.Cos(c.angleZ), math.Sin(c.angleZ)
		x2 := x*cosZ - y*sinZ
		y2 := x*sinZ + y*cosZ
		x, y = x2, y2

		rotated[i] = [3]float64{x, y, z}
	}

	// Calculate face depths for sorting
	type faceDepth struct {
		index int
		depth float64
	}
	depths := make([]faceDepth, len(faces))

	for i, face := range faces {
		// Calculate center of face
		centerZ := 0.0
		for _, vi := range face {
			centerZ += rotated[vi][2]
		}
		depths[i] = faceDepth{i, centerZ / 4}
	}

	// Sort faces by depth (back to front)
	for i := 0; i < len(depths)-1; i++ {
		for j := i + 1; j < len(depths); j++ {
			if depths[i].depth > depths[j].depth {
				depths[i], depths[j] = depths[j], depths[i]
			}
		}
	}

	// Draw faces
	for _, fd := range depths {
		face := faces[fd.index]
		faceColor := faceColors[fd.index]

		// Project vertices to 2D
		points := make([]float64, 0, 8)
		for _, vi := range face {
			v := rotated[vi]
			x2d, y2d := project3D(v[0], v[1], v[2])
			points = append(points, centerX+x2d, centerY+y2d)
		}

		// Draw filled polygon
		drawPolygon(screen, points, faceColor)

		// Draw edges with darker color for better visibility
		edgeColor := color.RGBA{
			uint8(faceColor.(color.RGBA).R * 3 / 4),
			uint8(faceColor.(color.RGBA).G * 3 / 4),
			uint8(faceColor.(color.RGBA).B * 3 / 4),
			255,
		}
		for i := 0; i < 4; i++ {
			j := (i + 1) % 4
			vector.StrokeLine(screen,
				float32(points[i*2]), float32(points[i*2+1]),
				float32(points[j*2]), float32(points[j*2+1]),
				1, edgeColor, false)
		}
	}
}

// drawPolygon draws a filled polygon
func drawPolygon(screen *ebiten.Image, points []float64, fillColor color.Color) {
	if len(points) < 6 {
		return
	}

	// Draw as a filled rectangle using vector
	if len(points) >= 8 {
		// Find bounding box
		minX, minY := points[0], points[1]
		maxX, maxY := points[0], points[1]
		for i := 2; i < len(points); i += 2 {
			if points[i] < minX {
				minX = points[i]
			}
			if points[i] > maxX {
				maxX = points[i]
			}
			if points[i+1] < minY {
				minY = points[i+1]
			}
			if points[i+1] > maxY {
				maxY = points[i+1]
			}
		}

		// Draw filled quadrilateral as two triangles
		// Triangle 1: points 0, 1, 2
		drawTriangle(screen,
			float32(points[0]), float32(points[1]),
			float32(points[2]), float32(points[3]),
			float32(points[4]), float32(points[5]),
			fillColor)

		// Triangle 2: points 0, 2, 3
		drawTriangle(screen,
			float32(points[0]), float32(points[1]),
			float32(points[4]), float32(points[5]),
			float32(points[6]), float32(points[7]),
			fillColor)
	}
}

// drawTriangle draws a filled triangle
func drawTriangle(screen *ebiten.Image, x1, y1, x2, y2, x3, y3 float32, clr color.Color) {
	// Draw triangle using lines to fill it
	// Sort vertices by Y coordinate
	if y1 > y2 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}
	if y1 > y3 {
		x1, y1, x3, y3 = x3, y3, x1, y1
	}
	if y2 > y3 {
		x2, y2, x3, y3 = x3, y3, x2, y2
	}

	// Draw horizontal lines to fill the triangle
	for y := y1; y <= y3; y++ {
		var xStart, xEnd float32

		if y < y2 {
			// Upper part of triangle
			if y2-y1 > 0 {
				t := (y - y1) / (y2 - y1)
				x12 := x1 + (x2-x1)*t
				t13 := (y - y1) / (y3 - y1)
				x13 := x1 + (x3-x1)*t13
				xStart, xEnd = x12, x13
			}
		} else {
			// Lower part of triangle
			if y3-y2 > 0 && y3-y1 > 0 {
				t := (y - y2) / (y3 - y2)
				x23 := x2 + (x3-x2)*t
				t13 := (y - y1) / (y3 - y1)
				x13 := x1 + (x3-x1)*t13
				xStart, xEnd = x23, x13
			}
		}

		if xStart > xEnd {
			xStart, xEnd = xEnd, xStart
		}

		vector.StrokeLine(screen, xStart, y, xEnd, y, 1, clr, false)
	}
}

// Game represents the main game state
type Game struct {
	// Demo assets
	cubes     [nbCubes]*Cube3D
	spritePos [nbCubes]float64
	logo      *ebiten.Image
	logoPos   float64
	wl, hl    int
	bars      *ebiten.Image

	// Copper bars animation
	copperSin []int
	cnt       int
	cnt2      int

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
	ymPlayer     *YMPlayer

	// Speed control
	speedMultiplier float64

	// Initialization flag
	initialized bool
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		speedMultiplier: 1.0,
		cnt:             0,
		cnt2:            0,
	}

	// Initialize scroll deformation data
	g.initScrollX()

	// Initialize copper bars sine table
	g.initCopperSin()

	// Initialize audio context
	g.audioContext = audio.NewContext(sampleRate)

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
	g.scrollX = make([]float64, 0)

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
	g.wl, g.hl = g.logo.Size()

	// Load bars image
	img, _, err = image.Decode(bytes.NewReader(barsImg))
	if err != nil {
		return fmt.Errorf("failed to load bars image: %v", err)
	}
	g.bars = ebiten.NewImageFromImage(img)

	// Load scroll font
	img, _, err = image.Decode(bytes.NewReader(scrollFontData))
	if err != nil {
		return fmt.Errorf("failed to load scroll font: %v", err)
	}
	g.scrollFont = ebiten.NewImageFromImage(img)

	return nil
}

// initScrollText initializes the scrolling text with soap font
func (g *Game) initScrollText() {
	scrollText := `      HELLO, BILIZIR FROM DMA IS PROUD TO PRESENT HIS NEW GOLANG/EBITEN INTRO... NOT SO BAD FOR A FEW HOURS OF HARD WORK :)  HI TO ALL MEMBERS OF DMA (COUCOU PHILIPPE ET DIDIER ALORS PAS MAL NON ?), ALL MEMBERS OF THE UNION, ALL DEMOSCENE FANS...   LET'S WRAP...      `

	g.scrollText = &ScrollText{
		text:         scrollText,
		x:            0,
		fontImage:    g.scrollFont,
		charWidth:    32,
		charHeight:   32,
		charsPerRow:  10,
		scrollBuffer: ebiten.NewImage(screenWidth+512, scrollHeight),  // Increased buffer for 2x font
		workBuffer:   ebiten.NewImage(screenWidth+1024, scrollHeight), // Even larger for 2x deformation
		deformBuffer: ebiten.NewImage(screenWidth, scrollHeight),
	}
}

// loadMusic loads and plays the YM music
func (g *Game) loadMusic() error {
	var err error

	// Create YM player
	g.ymPlayer, err = NewYMPlayer(musicData, sampleRate, true)
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

// charToFontIndex converts a character to its position in the soap font bitmap
func charToFontIndex(ch rune) (int, bool) {
	// Font layout (6 rows of 10 characters):
	// Row 0: ABCDEFGHIJ
	// Row 1: KLMNOPQRST
	// Row 2: UVWXYZ0123
	// Row 3: 456789(),.
	// Row 4: ![NA][NA][NA][NA][NA][NA][NA][NA][NA]
	// Row 5: [NA][NA][NA][NA][NA][NA][NA][NA][NA][NA]

	// Convert to uppercase for case-insensitive matching
	ch = rune(byte(ch) & ^byte(0x20))

	switch ch {
	// Row 0: ABCDEFGHIJ
	case 'A':
		return 0, true
	case 'B':
		return 1, true
	case 'C':
		return 2, true
	case 'D':
		return 3, true
	case 'E':
		return 4, true
	case 'F':
		return 5, true
	case 'G':
		return 6, true
	case 'H':
		return 7, true
	case 'I':
		return 8, true
	case 'J':
		return 9, true
	// Row 1: KLMNOPQRST
	case 'K':
		return 10, true
	case 'L':
		return 11, true
	case 'M':
		return 12, true
	case 'N':
		return 13, true
	case 'O':
		return 14, true
	case 'P':
		return 15, true
	case 'Q':
		return 16, true
	case 'R':
		return 17, true
	case 'S':
		return 18, true
	case 'T':
		return 19, true
	// Row 2: UVWXYZ0123
	case 'U':
		return 20, true
	case 'V':
		return 21, true
	case 'W':
		return 22, true
	case 'X':
		return 23, true
	case 'Y':
		return 24, true
	case 'Z':
		return 25, true
	case '0':
		return 26, true
	case '1':
		return 27, true
	case '2':
		return 28, true
	case '3':
		return 29, true
	// Row 3: 456789(),.
	case '4':
		return 30, true
	case '5':
		return 31, true
	case '6':
		return 32, true
	case '7':
		return 33, true
	case '8':
		return 34, true
	case '9':
		return 35, true
	case '(':
		return 36, true
	case ')':
		return 37, true
	case ',':
		return 38, true
	case '.':
		return 39, true
	// Row 4: ![NA][NA][NA][NA][NA][NA][NA][NA][NA]
	case '!':
		return 40, true
	// Treat unknown characters as spaces
	default:
		return -1, false
	}
}

// Update updates the game state
func (g *Game) Update() error {
	if !g.initialized {
		return g.Init()
	}

	// Handle input for volume control
	if g.ymPlayer != nil {
		if ebiten.IsKeyPressed(ebiten.KeyUp) {
			vol := g.ymPlayer.GetVolume() + 0.01
			if vol > 1.0 {
				vol = 1.0
			}
			g.ymPlayer.SetVolume(vol)
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) {
			vol := g.ymPlayer.GetVolume() - 0.01
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
	textWidth := float64(len(g.scrollText.text) * g.scrollText.charWidth * 2)
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

	barsWidth, barsHeight := g.bars.Size()
	if barsHeight < 20 {
		return
	}

	// Draw 210 copper bars (adjusted to fill the screen)
	cc := 0
	for i := 0; i < 300; i++ { // Increased from 210 to 300 to fill 600px height
		// Calculate sine positions
		val2 := (g.cnt + i*7) & 0x3ff
		val := g.copperSin[val2]
		val2 = (g.cnt2 + i*10) & 0x3ff
		val += g.copperSin[val2]
		val += 60

		// Position and size
		xPos := val >> 1
		yPos := i << 1 // i * 2
		height := screenHeight - yPos

		if height > 0 && yPos < screenHeight {
			op := &ebiten.DrawImageOptions{}

			// Source rectangle: 2 pixels high from bars
			srcRect := image.Rect(0, cc, barsWidth, cc+2)
			if srcRect.Max.Y > barsHeight {
				srcRect.Max.Y = barsHeight
			}

			// Scale to stretch the 2 pixels to fill the height
			scaleY := float64(height) / 2.0

			op.GeoM.Scale(1, scaleY)
			op.GeoM.Translate(float64(xPos), float64(yPos))

			screen.DrawImage(g.bars.SubImage(srcRect).(*ebiten.Image), op)
		}

		// Cycle through the bars
		cc += 2
		if cc >= 20 {
			cc = 0
		}
	}
}

// drawLogo draws the animated DMA logo
func (g *Game) drawLogo(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Reset()
	xPos := (float64(screenWidth-g.wl) / 2) + (math.Sin(g.logoPos) * float64(screenWidth-g.wl) / 2)
	op.GeoM.Translate(xPos, 0)
	screen.DrawImage(g.logo, op)
}

// drawCubes draws the rotating 3D cubes
func (g *Game) drawCubes(screen *ebiten.Image) {
	for i := 0; i < nbCubes; i++ {
		xPos := float64((screenWidth-40)/2) + (float64((screenWidth-40)/2) * math.Sin(g.spritePos[i]))
		yPos := 186 + (84 * math.Cos(g.spritePos[i]*2.5))

		// Draw the 3D cube
		g.cubes[i].Draw(screen, xPos, yPos)
	}
}

// drawScrollText draws the TCB-style scrolling text with deformation
func (g *Game) drawScrollText(screen *ebiten.Image) {
	// Clear buffers
	g.scrollText.workBuffer.Clear()
	g.scrollText.deformBuffer.Clear()

	// Scale factor for the font
	const fontScale = 2.0
	scaledCharWidth := float64(g.scrollText.charWidth) * fontScale

	// Draw text to work buffer with 2x scale
	x := g.scrollText.x
	for _, ch := range g.scrollText.text {
		if ch == ' ' {
			x += scaledCharWidth
			continue
		}

		// Get character position in font
		charIndex, found := charToFontIndex(ch)
		if !found {
			// Character not in font, treat as space
			x += scaledCharWidth
			continue
		}

		row := charIndex / g.scrollText.charsPerRow
		col := charIndex % g.scrollText.charsPerRow

		sx := col * g.scrollText.charWidth
		sy := row * g.scrollText.charHeight

		if x > -scaledCharWidth && x < float64(g.scrollText.workBuffer.Bounds().Dx()) {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(fontScale, fontScale)
			op.GeoM.Translate(x, 0)

			subImg := g.scrollText.fontImage.SubImage(
				image.Rect(sx, sy, sx+g.scrollText.charWidth, sy+g.scrollText.charHeight),
			).(*ebiten.Image)

			g.scrollText.workBuffer.DrawImage(subImg, op)
		}

		x += scaledCharWidth
	}

	// Apply deformation line by line (adjusted for 2x scale)
	for y := 0; y < 32; y++ { // Increased from 25 to 32 for larger font
		offsetX := g.scrollX[(g.vbl+y)%g.scrollXMod] + 64

		// Draw each line with horizontal offset
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-offsetX, 0)

		srcRect := image.Rect(int(offsetX), y*2, int(offsetX)+screenWidth, (y+1)*2)
		if srcRect.Min.X < 0 {
			srcRect.Min.X = 0
		}
		if srcRect.Max.X > g.scrollText.workBuffer.Bounds().Dx() {
			srcRect.Max.X = g.scrollText.workBuffer.Bounds().Dx()
		}

		subImg := g.scrollText.workBuffer.SubImage(srcRect).(*ebiten.Image)

		dstOp := &ebiten.DrawImageOptions{}
		dstOp.GeoM.Translate(0, float64(y*2))
		g.scrollText.deformBuffer.DrawImage(subImg, dstOp)
	}

	// Draw deformed scroll with vertical wave
	for x := 0; x < 50; x++ { // Adjusted for 800px width
		yOffset := 35 + math.Cos(g.offsetScr+float64(x)*0.1)*35

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x*16), float64(screenHeight-140)+yOffset) // Adjusted Y position for larger text

		subImg := g.scrollText.deformBuffer.SubImage(
			image.Rect(x*16, 0, (x+1)*16, scrollHeight),
		).(*ebiten.Image)

		screen.DrawImage(subImg, op)
	}
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
	if g.audioPlayer != nil {
		g.audioPlayer.Close()
	}
	if g.ymPlayer != nil {
		g.ymPlayer.Close()
	}
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Bilizir from DMA - the Weird intro")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	// Ensure cleanup on exit
	defer game.Cleanup()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
