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
	g.vbl = 100
	g.offsetScr = 12.3
	g.SetLogoDeformation(false)
	if g.LogoDeformationEnabled() || g.vbl != 100 || g.offsetScr != 12.3 {
		t.Fatal("mode change modified animation clocks")
	}
	g.SetLogoDeformation(true)
	if !g.LogoDeformationEnabled() {
		t.Fatal("could not re-enable deformation")
	}
}

func TestLogoPaddingCoversEveryHorizontalWave(t *testing.T) {
	g := NewGame()
	config, _, err := image.DecodeConfig(bytes.NewReader(logoImg))
	if err != nil {
		t.Fatal(err)
	}
	g.logoWidth = config.Width
	// Check both movement extremes against every entry in the original table.
	for _, phase := range []float64{-math.Pi / 2, math.Pi / 2} {
		g.logoPos = phase
		for tick := range g.scrollX {
			for row := 0; row < (config.Height+1)/2; row++ {
				x := g.warpedLogoX() + deformationMargin - float64(g.deformation.SampleX(row, kit.Frame{Tick: uint64(tick)}))
				if x < 0 || x+float64(config.Width) > screenWidth {
					t.Fatalf("logo clipped at phase=%g tick=%d row=%d: x=%g", phase, tick, row, x)
				}
			}
		}
	}
}
