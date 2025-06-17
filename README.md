# Bilizir Demo

A original demo coded in Go using the Ebiten game engine.

## Features

- **Classic Demo Effects**:
  - Animated copper bars with dual sine wave movement
  - Bouncing DMA logo with horizontal sine motion
  - Multiple rotating 3D cubes with complex movement patterns
  - TCB-style deformed scrolling text with wave effects

- **Audio Support**:
  - YM music playback (Atari ST chip music format)
  - Volume control with real-time adjustment
  - Infinite loop playback

- **Interactive Controls**:
  - Volume adjustment (Up/Down arrow keys)
  - Speed control (+/- keys)
  - Window resizing support

## Requirements

- Go 1.16 or higher
- Ebiten v2 game engine
- YM player library

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
go run main.go
```

### Build executable:
```bash
go build -o bilizir-demo main.go
```

### Build with embedded assets:
The demo uses Go's embed directive to include all assets in the binary:
```bash
go build -ldflags="-s -w" -o bilizir-demo main.go
```

## Controls

- **Arrow Up**: Increase volume
- **Arrow Down**: Decrease volume
- **+/=**: Increase animation speed (max 2.0x)
- **-**: Decrease animation speed (min 0.5x)

## Technical Details

### Screen Resolution
- Width: 800 pixels
- Height: 600 pixels

### Demo Components

1. **Copper Bars**: Animated bars with dual sine wave movement creating a fluid motion effect
2. **Logo Animation**: DMA logo with horizontal sine movement
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
