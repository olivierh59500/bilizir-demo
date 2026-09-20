package bilizir

import (
	"fmt"
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
			if index < 0 {
				index += g.scrollXMod
			}
			return int(g.scrollX[index] + deformationMargin)
		},
		OffsetY: func(column int, _ kit.Frame) float64 {
			return math.Cos(g.offsetScr+float64(column)*0.1) * 35
		},
	}
}

func (g *Game) initLogoDeformation() error {
	height := g.logo.Bounds().Dy()
	config := g.deformation
	config.RowHeight, config.ColumnWidth = g.logoOptions.RowHeight, g.logoOptions.ColumnWidth
	warp, err := composite.NewStripWarp(image.Pt(screenWidth, height), config)
	if err != nil {
		return err
	}
	variation := composite.WarpVariation{
		RowPhase: g.logoOptions.RowPhase, ColumnPhase: g.logoOptions.ColumnPhase,
		HorizontalGain: g.logoOptions.HorizontalGain, VerticalGain: g.logoOptions.VerticalGain,
		SampleOrigin: deformationMargin, SampleOffset: float64(g.logoMargin - deformationMargin),
	}
	if err = warp.SetVariation(variation); err != nil {
		warp.Close()
		return err
	}
	if g.logoWarp != nil {
		g.logoWarp.Close()
	}
	if g.logoBuffer != nil {
		g.logoBuffer.Deallocate()
	}
	g.logoWarp = warp
	g.logoBuffer = ebiten.NewImageWithOptions(
		image.Rect(0, 0, screenWidth+2*g.logoMargin, height),
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
	amplitude := math.Max(0, center-float64(g.logoMargin))
	return center + math.Sin(g.logoPos)*amplitude
}

func (g *Game) drawWarpedLogo(screen *ebiten.Image) {
	g.logoBuffer.Clear()
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.warpedLogoX()+float64(g.logoMargin), 0)
	composite.Instance{Image: g.logo, Options: op}.Draw(g.logoBuffer)
	g.logoWarp.DrawAt(screen, g.logoBuffer, kit.Frame{Tick: uint64(g.vbl)}, 0, 0)
}

// LogoWarpOptions changes the logo independently while both effects keep their
// original clocks. Phases use row/column units; negative gains mirror the wave.
type LogoWarpOptions struct {
	RowPhase, ColumnPhase        int
	HorizontalGain, VerticalGain float64
	RowHeight, ColumnWidth       int
}

func DefaultLogoWarpOptions() LogoWarpOptions {
	return LogoWarpOptions{HorizontalGain: 1, VerticalGain: 1, RowHeight: 2, ColumnWidth: 16}
}

// SetLogoWarpOptions can run before initialization or on the graphics goroutine.
// Larger amplitudes reserve extra source padding; screen edges still clip the
// final image normally. The scroller's configuration remains independent.
func (g *Game) SetLogoWarpOptions(options LogoWarpOptions) error {
	for _, v := range []float64{options.HorizontalGain, options.VerticalGain} {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 16 {
			return fmt.Errorf("bilizir: logo gains must be finite and within [-16,16]")
		}
	}
	if options.RowHeight < 1 || options.ColumnWidth < 1 {
		return fmt.Errorf("bilizir: strip dimensions must be positive")
	}
	g.logoOptions = options
	g.logoMargin = deformationMargin + int(math.Ceil(50*math.Max(0, math.Abs(options.HorizontalGain)-1)))
	if g.logo != nil {
		return g.initLogoDeformation()
	}
	return nil
}
