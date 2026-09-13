package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	bilizir "bilizir-demo"
)

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Bilizir from DMA - the Weird intro")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := bilizir.NewGame()
	defer game.Cleanup()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
