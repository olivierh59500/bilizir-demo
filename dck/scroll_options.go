package bilizir

import "fmt"

// SetOriginalScrollReset selects the preserved Go production's strict text
// reset before initialization. The default uses the seamless two-copy loop.
// Call this before the first Update; changing a running transport would jump.
func (g *Game) SetOriginalScrollReset(enabled bool) error {
	if g.initialized {
		return fmt.Errorf("bilizir: scroll reset mode must be selected before initialization")
	}
	g.originalScrollReset = enabled
	return nil
}
