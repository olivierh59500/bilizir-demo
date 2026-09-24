// Package bilizir implements the Bilizir demo for desktop and mobile frontends.
package bilizir

import (
	originalassets "bilizir-demo"
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/olivierh59500/democonstructionkit/sound"

	_ "image/png"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	kit "github.com/olivierh59500/democonstructionkit"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/effects"
	"github.com/olivierh59500/democonstructionkit/motion"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/scrolling"

	audio "github.com/olivierh59500/democonstructionkit/sound/output"
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

// Game represents the main game state
type Game struct {
	logoOptions            LogoWarpOptions
	logoMargin             int
	deformation            composite.StripWarpConfig
	logoWarp               *composite.StripWarp
	logoBuffer             *ebiten.Image
	logoDeformationEnabled bool
	// Demo assets
	cubes     [nbCubes]*effects.SolidCube
	spritePos [nbCubes]float64
	logo      *ebiten.Image
	logoPos   float64
	logoWidth int
	bars      *ebiten.Image

	// Copper bars animation
	copper *composite.CopperBars

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
	musicStream  *sound.Stream

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
		logoDeformationEnabled: true,
	}

	// Initialize scroll deformation data
	g.initScrollX()
	g.configureDeformation()

	return g
}

func (g *Game) initScrollX() {
	var err error
	g.scrollX, err = motion.CompileWaveTable(presets.BilizirWaveSections()...)
	if err != nil {
		panic(err)
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
		g.cubes[i], err = effects.NewSolidCube(presets.BilizirCube(20))
		if err != nil {
			return err
		}
		g.cubes[i].Rotation.X = float64(i) * 0.3
		g.cubes[i].Rotation.Y = float64(i) * 0.5
		g.cubes[i].Rotation.Z = float64(i) * 0.2
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
	g.copper, err = composite.NewCopperBars(presets.BilizirCopperBars(g.bars, screenHeight, composite.CopperQuads, composite.MaskedClock))
	if err != nil {
		return err
	}

	// Load scroll font
	img, _, err = image.Decode(bytes.NewReader(scrollFontData))
	if err != nil {
		return fmt.Errorf("failed to load scroll font: %v", err)
	}
	g.scrollFont = ebiten.NewImageFromImage(img)

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

// loadMusic opens and plays the soundtrack through DCK.
func (g *Game) loadMusic() error {
	var err error

	// Audio must be initialized after the Android activity and Ebiten view are ready.
	g.audioContext = audio.NewContext(sampleRate)

	// Let DCK choose and configure the music decoder.
	g.musicStream, err = sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.5})
	if err != nil {
		return fmt.Errorf("failed to open music: %w", err)
	}

	// Create audio player
	g.audioPlayer, err = g.audioContext.NewPlayer(g.musicStream)
	if err != nil {
		g.musicStream.Close()
		g.musicStream = nil
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
	if g.musicStream != nil {
		if ebiten.IsKeyPressed(ebiten.KeyUp) {
			vol := g.musicStream.Volume() + 0.01
			if vol > 1.0 {
				vol = 1.0
			}
			g.musicStream.SetVolume(vol)
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) {
			vol := g.musicStream.Volume() - 0.01
			if vol < 0 {
				vol = 0
			}
			g.musicStream.SetVolume(vol)
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
	if err := g.copper.Update(kit.Frame{}); err != nil {
		return err
	}

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
		g.cubes[i].DrawAt(screen, xPos, yPos)
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
	g.copper.Draw(screen)

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
	for _, cube := range g.cubes {
		if cube != nil {
			cube.Close()
		}
	}
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
	if g.musicStream != nil {
		g.musicStream.Close()
	}
}
