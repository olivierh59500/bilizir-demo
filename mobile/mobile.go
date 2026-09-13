// Package mobile exposes the Bilizir demo to ebitenmobile.
package mobile

import (
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"

	bilizir "bilizir-demo"
)

func init() {
	enginemobile.SetGame(bilizir.NewGame())
}

// Dummy forces gomobile to include this package in the Android binding.
func Dummy() {}
