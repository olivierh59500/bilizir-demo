//go:build dck_rendercheck

package bilizir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	fidelitycapture "github.com/olivierh59500/democonstructionkit/fidelity/ebiten"
	"github.com/olivierh59500/democonstructionkit/render"
)

var renderCheckFrames = []int{0, 1, 60, 240, 600, 1200, 2400, 4800}

type renderCheckResult struct{ Frame, ScrollDifferentPixels, LogoDifferentPixels, PhaseDifferentPixels, DisabledAxesDifferentPixels, BottomMarkerPixels int }
type logoRenderCheck struct {
	game                                                  *Game
	frame, index                                          int
	directory                                             string
	actual, expected, logo, plain, marker, work, deformed *ebiten.Image
	a, b                                                  []byte
	results                                               []renderCheckResult
	err                                                   error
}

// This opt-in native check uses the real GPU and keeps generated images outside Git.
func TestMain(m *testing.M) {
	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
	dir := os.Getenv("DCK_BILIZIR_CAPTURES")
	if dir == "" {
		var err error
		dir, err = os.MkdirTemp("", "bilizir-logo-warp-")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	var check *logoRenderCheck
	err := fidelitycapture.Run(fidelitycapture.Config{Directory: dir, Frames: renderCheckFrames, Width: screenWidth, Height: screenHeight}, func() (ebiten.Game, error) {
		g := NewGame()
		if err := g.loadAssets(); err != nil {
			return nil, err
		}
		g.initScrollText()
		g.initialized = true
		check = &logoRenderCheck{game: g, directory: dir, actual: render.NewSurface(screenWidth, screenHeight), expected: render.NewSurface(screenWidth, screenHeight), logo: render.NewSurface(screenWidth, screenHeight), plain: render.NewSurface(screenWidth, screenHeight), work: render.NewSurface(screenWidth+1024, scrollHeight), deformed: render.NewSurface(screenWidth, scrollHeight), a: make([]byte, screenWidth*screenHeight*4), b: make([]byte, screenWidth*screenHeight*4)}
		// A solid red last scanline catches truncation of the odd 171-pixel height.
		pixels := image.NewRGBA(image.Rect(0, 0, g.logo.Bounds().Dx(), g.logo.Bounds().Dy()))
		for y := 0; y < pixels.Bounds().Dy(); y++ {
			for x := 0; x < pixels.Bounds().Dx(); x++ {
				c := color.RGBA{255, 255, 255, 255}
				if y == pixels.Bounds().Dy()-1 {
					c = color.RGBA{255, 0, 0, 255}
				}
				pixels.SetRGBA(x, y, c)
			}
		}
		check.marker = ebiten.NewImageFromImage(pixels)
		return check, nil
	})
	if err == nil && check != nil {
		err = check.err
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(check.results, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(dir, "checks.json"), append(data, '\n'), 0644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Logo variation verified; captures: %s\n", dir)
}

func (c *logoRenderCheck) Layout(int, int) (int, int) { return screenWidth, screenHeight }
func (c *logoRenderCheck) Update() error {
	if c.err != nil {
		return c.err
	}
	c.frame++
	return c.game.Update()
}
func (c *logoRenderCheck) Draw(screen *ebiten.Image) {
	c.game.Draw(screen)
	if c.err != nil || c.index >= len(renderCheckFrames) || c.frame != renderCheckFrames[c.index] {
		return
	}
	c.index++
	c.actual.Clear()
	c.expected.Clear()
	c.game.drawScrollText(c.actual)
	c.drawReferenceScroll(c.expected)
	c.actual.ReadPixels(c.a)
	c.expected.ReadPixels(c.b)
	scrollDifferences := differentPixels(c.a, c.b)
	if scrollDifferences != 0 {
		c.err = fmt.Errorf("frame %d: scrolling changed at %d pixels", c.frame, scrollDifferences)
		return
	}
	c.logo.Clear()
	c.plain.Clear()
	c.game.SetLogoDeformation(true)
	c.game.drawLogo(c.logo)
	c.game.SetLogoDeformation(false)
	c.game.drawLogo(c.plain)
	c.game.SetLogoDeformation(true)
	c.logo.ReadPixels(c.a)
	c.plain.ReadPixels(c.b)
	logoDifferences := differentPixels(c.a, c.b)
	if logoDifferences == 0 {
		c.err = fmt.Errorf("frame %d: logo deformation has no visible effect", c.frame)
		return
	}
	// Repeated Draw calls must not advance either phase or move the logo.
	c.logo.Clear()
	c.game.drawLogo(c.logo)
	c.logo.ReadPixels(c.b)
	if !bytes.Equal(c.a, c.b) {
		c.err = fmt.Errorf("frame %d: repeated logo drawing changes state", c.frame)
		return
	}
	if err := saveCheckPNG(filepath.Join(c.directory, fmt.Sprintf("logo-%06d.png", c.frame)), c.logo); err != nil {
		c.err = err
		return
	}
	// Independent phase, signed amplitude and odd strip sizes must change the logo
	// without resetting the clocks shared with the text.
	options := DefaultLogoWarpOptions()
	options.RowPhase = -40
	options.ColumnPhase = 12
	options.HorizontalGain = .75
	options.VerticalGain = -.6
	options.RowHeight = 3
	options.ColumnWidth = 11
	tick, phase := c.game.warpClock.Tick(), c.game.warpClock.Phase()
	if err := c.game.SetLogoWarpOptions(options); err != nil {
		c.err = err
		return
	}
	c.plain.Clear()
	c.game.drawLogo(c.plain)
	c.plain.ReadPixels(c.b)
	phaseDifferences := differentPixels(c.a, c.b)
	if phaseDifferences == 0 || c.game.warpClock.Tick() != tick || c.game.warpClock.Phase() != phase {
		c.err = fmt.Errorf("frame %d: independent logo phase failed", c.frame)
		return
	}
	if err := saveCheckPNG(filepath.Join(c.directory, fmt.Sprintf("logo-shifted-%06d.png", c.frame)), c.plain); err != nil {
		c.err = err
		return
	}
	options.HorizontalGain = 0
	options.VerticalGain = 0
	if err := c.game.SetLogoWarpOptions(options); err != nil {
		c.err = err
		return
	}
	c.plain.Clear()
	c.game.drawLogo(c.plain)
	c.plain.ReadPixels(c.a)
	c.expected.Clear()
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(c.game.warpedLogoX(), 35)
	c.expected.DrawImage(c.game.logo, &op)
	c.expected.ReadPixels(c.b)
	disabledDifferences := differentPixels(c.a, c.b)
	if disabledDifferences != 0 {
		c.err = fmt.Errorf("frame %d: zero-gain logo differs at %d pixels", c.frame, disabledDifferences)
		return
	}
	if err := c.game.SetLogoWarpOptions(DefaultLogoWarpOptions()); err != nil {
		c.err = err
		return
	}
	original := c.game.logo
	c.game.logo = c.marker
	c.logo.Clear()
	c.game.drawLogo(c.logo)
	c.game.logo = original
	c.logo.ReadPixels(c.a)
	red := 0
	for i := 0; i < len(c.a); i += 4 {
		if c.a[i] == 255 && c.a[i+1] == 0 && c.a[i+2] == 0 && c.a[i+3] == 255 {
			red++
		}
	}
	if red != c.marker.Bounds().Dx() {
		c.err = fmt.Errorf("frame %d: last logo row has %d pixels, want %d", c.frame, red, c.marker.Bounds().Dx())
		return
	}
	c.results = append(c.results, renderCheckResult{c.frame, scrollDifferences, logoDifferences, phaseDifferences, disabledDifferences, red})
}

// Retain the previous two-pass drawing operations as an independent visual oracle.
func (c *logoRenderCheck) drawReferenceScroll(dst *ebiten.Image) {
	g := c.game
	c.work.Clear()
	c.deformed.Clear()
	state := g.scrollText.window.At(g.scrollText.loop.At(0))
	state.ScaleY = 2
	g.scrollText.renderer.DrawAt(c.work, state)
	for row := 0; row < 32; row++ {
		table := g.warpClock.Table()
		x := int(table[(int(g.warpClock.Tick()%uint64(len(table)))+row)%len(table)] + 64)
		r := image.Rect(max(0, x), row*2, min(x+screenWidth, c.work.Bounds().Dx()), (row+1)*2)
		op := ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(row*2))
		c.deformed.DrawImage(c.work.SubImage(r).(*ebiten.Image), &op)
	}
	for column := 0; column < 50; column++ {
		r := image.Rect(column*16, 0, (column+1)*16, scrollHeight)
		op := ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(column*16), float64(screenHeight-140)+35+math.Cos(g.warpClock.Phase()+float64(column)*.1)*35)
		dst.DrawImage(c.deformed.SubImage(r).(*ebiten.Image), &op)
	}
}
func differentPixels(a, b []byte) int {
	n := 0
	for i := 0; i < len(a); i += 4 {
		if !bytes.Equal(a[i:i+4], b[i:i+4]) {
			n++
		}
	}
	return n
}
func saveCheckPNG(path string, source *ebiten.Image) error {
	pixels := image.NewRGBA(source.Bounds())
	source.ReadPixels(pixels.Pix)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = png.Encode(f, pixels)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
