package bilizir

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	kit "github.com/olivierh59500/democonstructionkit"
	"github.com/olivierh59500/democonstructionkit/composite"
)

// The source margin cancels the scroll's original +64 sampling origin. It also
// reserves room for the horizontal displacement, whose amplitude is at most 50.
const deformationMargin = 64

func (g *Game) configureDeformation() {
	g.deformation = composite.StripWarpConfig{
		RowHeight:    2,
		ColumnWidth:  16,
		VerticalBias: 35,
		SampleX: func(row int, frame kit.Frame) int {
			index := (int(frame.Tick%uint64(g.scrollXMod)) + row) % g.scrollXMod
			return int(g.scrollX[index] + deformationMargin)
		},
		OffsetY: func(column int, _ kit.Frame) float64 {
			return math.Cos(g.offsetScr+float64(column)*0.1) * 35
		},
	}
}

func (g *Game) initLogoDeformation() error {
	height := g.logo.Bounds().Dy()
	warp, err := composite.NewStripWarp(image.Pt(screenWidth, height), g.deformation)
	if err != nil {
		return err
	}
	g.logoWarp = warp
	g.logoBuffer = ebiten.NewImageWithOptions(
		image.Rect(0, 0, screenWidth+2*deformationMargin, height),
		&ebiten.NewImageOptions{Unmanaged: true},
	)
	return nil
}

// SetLogoDeformation switches between the variation and the original logo path.
// The shared animation clocks keep advancing in either mode.
func (g *Game) SetLogoDeformation(enabled bool) { g.logoDeformationEnabled = enabled }

// LogoDeformationEnabled reports the current mode, toggled with L on desktop.
func (g *Game) LogoDeformationEnabled() bool { return g.logoDeformationEnabled }

func (g *Game) warpedLogoX() float64 {
	center := float64(screenWidth-g.logoWidth) / 2
	// Keep the whole moving logo inside the screen after the row displacement.
	amplitude := math.Max(0, center-deformationMargin)
	return center + math.Sin(g.logoPos)*amplitude
}

func (g *Game) drawWarpedLogo(screen *ebiten.Image) {
	g.logoBuffer.Clear()
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.warpedLogoX()+deformationMargin, 0)
	composite.Instance{Image: g.logo, Options: op}.Draw(g.logoBuffer)
	g.logoWarp.DrawAt(screen, g.logoBuffer, kit.Frame{Tick: uint64(g.vbl)}, 0, 0)
}
