package bilizir

import (
	"bytes"
	"image"
	"math"
	"testing"

	kit "github.com/olivierh59500/democonstructionkit"
)

func TestLogoVariationIsDefaultAndCanBeDisabled(t *testing.T) {
	g := NewGame()
	if !g.LogoDeformationEnabled() {
		t.Fatal("logo deformation must be enabled by default")
	}
	g.warpClock.SetTick(100)
	if err := g.warpClock.SetPhase(12.3); err != nil {
		t.Fatal(err)
	}
	if err := g.logoClock.SetPhase(.7); err != nil {
		t.Fatal(err)
	}
	g.SetLogoDeformation(false)
	if g.LogoDeformationEnabled() || g.warpClock.Tick() != 100 || g.warpClock.Phase() != 12.3 || g.logoClock.Phase() != .7 {
		t.Fatal("mode change modified animation clocks")
	}
	g.SetLogoDeformation(true)
	if !g.LogoDeformationEnabled() {
		t.Fatal("could not re-enable deformation")
	}
}

func TestIndependentLogoPhaseAndSignedAmplitude(t *testing.T) {
	g := NewGame()
	g.warpClock.SetTick(123)
	if err := g.warpClock.SetPhase(4.5); err != nil {
		t.Fatal(err)
	}
	if err := g.logoClock.SetPhase(.7); err != nil {
		t.Fatal(err)
	}
	before := g.deformation.SampleX(5, kit.Frame{Tick: 123})
	options := DefaultLogoWarpOptions()
	options.RowPhase = -90
	options.ColumnPhase = 12
	options.HorizontalGain = -1.5
	options.VerticalGain = .4
	if err := g.SetLogoWarpOptions(options); err != nil {
		t.Fatal(err)
	}
	if g.warpClock.Tick() != 123 || g.warpClock.Phase() != 4.5 || g.logoClock.Phase() != .7 || g.deformation.SampleX(5, kit.Frame{Tick: 123}) != before {
		t.Fatal("logo variation modified the scroll or clocks")
	}
	if g.logoMargin < 75 {
		t.Fatal("stronger horizontal deformation did not reserve source padding")
	}
	for row := -g.warpClock.Len() * 2; row < 0; row++ {
		g.deformation.SampleX(row, kit.Frame{Tick: 123})
	}
	options.HorizontalGain = math.NaN()
	if err := g.SetLogoWarpOptions(options); err == nil {
		t.Fatal("NaN gain accepted")
	}
}

func TestLogoPaddingCoversEveryHorizontalWave(t *testing.T) {
	g := NewGame()
	config, _, err := image.DecodeConfig(bytes.NewReader(logoImg))
	if err != nil {
		t.Fatal(err)
	}
	g.logoWidth = config.Width
	if err := g.configureLogoPaths(); err != nil {
		t.Fatal(err)
	}
	// Check both movement extremes against every entry in the original table.
	for _, phase := range []float64{-math.Pi / 2, math.Pi / 2} {
		if err := g.logoClock.SetPhase(phase); err != nil {
			t.Fatal(err)
		}
		for tick := 0; tick < g.warpClock.Len(); tick++ {
			g.warpClock.SetTick(uint64(tick))
			for row := 0; row < (config.Height+1)/2; row++ {
				x := g.warpedLogoX() + deformationMargin - float64(g.deformation.SampleX(row, kit.Frame{Tick: uint64(tick)}))
				if x < 0 || x+float64(config.Width) > screenWidth {
					t.Fatalf("logo clipped at phase=%g tick=%d row=%d: x=%g", phase, tick, row, x)
				}
			}
		}
	}
}
