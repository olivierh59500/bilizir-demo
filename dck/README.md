# DCK version

This directory contains the construction-kit version of bilizir-demo. The original Go sources are preserved at their original paths (revision `4720c97e03f3d2392211a37755d41b752fbad288`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/bilizir-demo` and this version with `go run ./dck/cmd/bilizir-demo` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Music is opened with `sound.Open`; DCK selects the decoder from the asset and
provides the configured stereo PCM format. The demo keeps its playback level and loop settings.

## Logo variation

The DCK logo shares the text deformation by default; L toggles the historical logo.

```sh
go run ./dck/cmd/bilizir-demo -logo-row-phase=-40 -logo-column-phase=12 -logo-x-gain=.7 -logo-y-gain=-1.2
```

Also available: `-logo-row-height` and `-logo-column-width`. Phases are signed strip offsets, gains multiply amplitudes (zero disables an axis). Use `DefaultLogoWarpOptions` and `SetLogoWarpOptions` from Go. Changing logo options leaves the text and animation clocks independent.

Native checks: `go test -tags dck_rendercheck ./dck`.

See the [DCK effect configuration guide](../../../lib/democonstructionkit/docs/EFFECT_OPTIONS.md) for the shared API and examples.
