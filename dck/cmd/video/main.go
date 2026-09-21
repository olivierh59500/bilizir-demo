// Command video exports the complete game canvas and its own audio.
package main

import (
	"flag"
	"log"
	"time"

	demo "bilizir-demo/dck"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/video"
)

func main() {
	config := video.Config{Output: "bilizir-demo.mp4", Title: "Bilizir", Width: 800, Height: 600, FPS: 60, TPS: 60, SampleRate: 44100, Duration: 3 * time.Minute}
	config.Flags(flag.CommandLine)
	flag.Parse()
	if err := video.Run(config, func() (ebiten.Game, error) {
		return demo.NewGame(), nil
	}); err != nil {
		log.Fatal(err)
	}
}
