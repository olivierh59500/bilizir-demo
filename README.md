# Bilizir Demo

A demo coded in Go using Ebitengine and the local democonstructionkit module.

The default version now applies the scrolling text's row/column deformation to
the DMA logo as well. Press **L** to switch between the warped logo and its original
appearance. Both use the same animation clocks, and the scrolling text retains
its original rendering.

## Features

- **Classic Demo Effects**:
  - Animated copper bars with dual sine wave movement
  - DMA logo with shared scrolling deformation and horizontal sine motion
  - Multiple rotating 3D cubes with complex movement patterns
  - TCB-style deformed scrolling text with wave effects

- **Audio Support**:
  - YM music playback (Atari ST chip music format)
  - Volume control with real-time adjustment
  - Infinite loop playback

- **Interactive Controls**:
  - Volume adjustment (Up/Down arrow keys)
  - Speed control (+/- keys)
  - Logo deformation toggle (L)
  - Window resizing support

## Requirements

- Go 1.26 or higher
- Ebiten v2 game engine
- YM player library
- Local DCK checkout at `../../lib/democonstructionkit` (see the `go.mod` replacement)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/olivierh59500/bilizir-demo.git
cd bilizir-demo
```

2. Install dependencies:
```bash
go mod init bilizir-demo
go get github.com/hajimehoshi/ebiten/v2
go get github.com/hajimehoshi/ebiten/v2/audio
go get github.com/olivierh59500/ym-player/pkg/stsound
```

3. Create the assets directory structure:
```bash
mkdir assets
```

4. Add the required assets to the `assets/` directory:
   - `logo.png` - DMA logo
   - `bars.png` - Copper bars image (at least 20 pixels height)
   - `soap-font.png` - Scrolling font (32x32 pixels per character, 10x6 grid)
   - `music.ym` - Background music in YM format (Atari ST chip music)

## Building and Running

### Run directly:
```bash
go run ./cmd/bilizir-demo
```

### Build executable:
```bash
go build -o bilizir-demo ./cmd/bilizir-demo
```

### Build with embedded assets:
The demo uses Go's embed directive to include all assets in the binary:
```bash
go build -ldflags="-s -w" -o bilizir-demo ./cmd/bilizir-demo
```

### Run on an Android device:

With one authorized arm64 Android device connected over USB:

```bash
./scripts/run-android.sh
```

The script builds the Ebitengine AAR and debug APK, installs it with ADB, and
launches `com.olivierh.bilizirdemo/.MainActivity`.

## Controls

- **Arrow Up**: Increase volume
- **Arrow Down**: Decrease volume
- **+/=**: Increase animation speed (max 2.0x)
- **-**: Decrease animation speed (min 0.5x)
- **L**: Toggle logo deformation; the variation is enabled by default

## Shared logo and scrolling deformation

`deformation.go` defines one `composite.StripWarpConfig`: two-pixel rows sample the
original horizontal wave table, then sixteen-pixel columns follow the original
vertical cosine. The text and logo each have their own `StripWarp` and scratch
surface, using that same configuration and phase values.

The logo keeps all 171 source rows, including its final partial two-pixel strip.
Its horizontal oscillation is inset by 64 pixels to leave room for the deformation.
The original path is restored when L disables the effect. Applications can select
the mode through `Game.SetLogoDeformation` before or during playback.

Native rendering verification, with device audio disabled:

```sh
go test -race ./...
DCK_BILIZIR_CAPTURES=/tmp/bilizir-logo-warp \
  go test -tags dck_rendercheck -run '^$' -count=1 -v .
```

The rendering check compares the scrolling against its previous implementation,
checks that the logo changes, verifies repeated drawing is deterministic, and uses
a bottom-row marker to detect clipping. It writes full-scene and isolated-logo
PNGs plus `checks.json` at eight original animation ticks.

## Technical Details

### Screen Resolution
- Width: 800 pixels
- Height: 600 pixels

### Demo Components

1. **Copper Bars**: Animated bars with dual sine wave movement creating a fluid motion effect
2. **Logo Animation**: DMA logo with the scroll's two deformation passes, a padded
   horizontal sine movement, and an original-mode toggle
3. **3D Cubes**: 12 rotating cubes with:
   - Real-time 3D rotation on all axes
   - Pink/magenta color scheme matching the demo aesthetic
   - Individual rotation speeds
4. **Scrolling Text**: TCB-style deformed text with:
   - Horizontal deformation using pre-calculated wave tables
   - Vertical sine wave movement
   - 32x32 pixel characters from soap font
   - Support for uppercase letters, numbers, and basic punctuation

### Font Layout
The soap font bitmap (soap-font.png) contains 6 rows of 10 characters:
- Row 0: ABCDEFGHIJ
- Row 1: KLMNOPQRST
- Row 2: UVWXYZ0123
- Row 3: 456789(),.
- Row 4: ![NA][NA][NA][NA][NA][NA][NA][NA][NA]
- Row 5: [NA][NA][NA][NA][NA][NA][NA][NA][NA][NA]

Each character is 32x32 pixels. The font supports uppercase letters, numbers, and basic punctuation.

### Audio System
- YM player integration for authentic Atari ST chip music
- Real-time volume control
- Thread-safe audio streaming
- Automatic looping

### Performance Optimization
- Pre-calculated deformation tables for smooth scrolling
- Efficient buffer management for text rendering
- Optimized sprite drawing with transformation matrices

## Credits

- Original demo by Olivier H
- YM player library by Olivier H
- Ebiten game engine by Hajime Hoshi
