# DCK version

This directory contains the construction-kit version of bilizir-demo. The original Go sources are preserved at their original paths (revision `4720c97e03f3d2392211a37755d41b752fbad288`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/bilizir-demo` and this version with `go run ./dck/cmd/bilizir-demo` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Music is opened with `sound.Open`; DCK selects the decoder from the asset and
provides the configured stereo PCM format. The demo keeps its playback level and loop settings.

The 300 background copper bars use one `composite.CopperBars` instance with
batched quads and the shared two-clock table preset. Width, source-strip period,
bar count, sample spacing and clock phases can be changed in its config. The
20-second migration capture matched all 1,200 original decoded frames.
The twelve rotating cubes now use one `effects.SolidCubeTrain`. DCK owns their
independent paths, rotations and bounded draw batch; the Bilizir preset keeps
the original 20-pixel material, phase spacing and live speed multiplier.
The pure 5,000-tick pose comparison passes; an opt-in GPU test is ready to
compare the new batch with the previous twelve individual cube draws.

## Logo variation

The DCK logo shares the text deformation by default; L toggles the historical logo.

```sh
go run ./dck/cmd/bilizir-demo -logo-row-phase=-40 -logo-column-phase=12 -logo-x-gain=.7 -logo-y-gain=-1.2
```

Also available: `-logo-row-height` and `-logo-column-width`. Phases are signed strip offsets, gains multiply amplitudes (zero disables an axis). Use `DefaultLogoWarpOptions` and `SetLogoWarpOptions` from Go. Changing logo options does not reset the shared wave clock or the text's parameters.
The text and logo warps now share a DCK `motion.WarpTableClock`, while each
`StripWarp` keeps its own phase, gain and strip-size variation. The DCK game no
longer stores a duplicate wave table or computes the column cosine itself.
The proportional text now uses a relative DCK wrap clock and two virtual
glyph copies, so its next pass is already entering when the previous one
leaves. This intentionally removes the blank interval after the first full
message while preserving all positions before that boundary. The original Go
version remains unchanged.
`scrolling.CyclicWindow` visits only the proportional glyphs near the 1,824-pixel
work surface, including their bearings; the two full message copies are never
submitted as unbounded per-frame work.

Native checks: `go test -tags dck_rendercheck ./dck`.

See the [DCK effect configuration guide](../../../lib/democonstructionkit/docs/EFFECT_OPTIONS.md) for the shared API and examples.
