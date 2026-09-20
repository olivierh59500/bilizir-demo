package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	bilizir "bilizir-demo/dck"
)

func main() {
	options := bilizir.DefaultLogoWarpOptions()
	flag.IntVar(&options.RowPhase, "logo-row-phase", 0, "horizontal wave phase in row strips")
	flag.IntVar(&options.ColumnPhase, "logo-column-phase", 0, "vertical wave phase in column strips")
	flag.Float64Var(&options.HorizontalGain, "logo-x-gain", 1, "horizontal deformation amplitude multiplier")
	flag.Float64Var(&options.VerticalGain, "logo-y-gain", 1, "vertical deformation amplitude multiplier")
	flag.IntVar(&options.RowHeight, "logo-row-height", 2, "horizontal strip height in pixels")
	flag.IntVar(&options.ColumnWidth, "logo-column-width", 16, "vertical strip width in pixels")
	flag.Parse()
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Bilizir from DMA - the Weird intro")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := bilizir.NewGame()
	if err := game.SetLogoWarpOptions(options); err != nil {
		log.Fatal(err)
	}
	defer game.Cleanup()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
