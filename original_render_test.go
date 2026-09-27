//go:build bilizir_original_rendercheck

package bilizir

import (
	"fmt"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	capture "github.com/olivierh59500/democonstructionkit/fidelity/ebiten"
)

// Capture the preserved Go implementation at fixed ticks without audio output.
func TestMain(m *testing.M) {
	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
	directory := os.Getenv("BILIZIR_ORIGINAL_CAPTURES")
	if directory == "" {
		var err error
		directory, err = os.MkdirTemp("", "bilizir-original-")
		if err != nil {
			panic(err)
		}
	}
	frames := []int{0, 1, 60, 240, 600, 1200, 2400, 4800}
	var game *Game
	err := capture.Run(capture.Config{Directory: directory, Frames: frames,
		Width: screenWidth, Height: screenHeight}, func() (ebiten.Game, error) {
		game = NewGame()
		if err := game.loadAssets(); err != nil {
			return nil, err
		}
		game.initScrollText()
		game.initialized = true
		return game, nil
	})
	if game != nil {
		game.Cleanup()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Bilizir original captures:", directory)
}
