// Package ymaudio adapts a mono YM decoder to Ebitengine's stereo PCM stream.
package ymaudio

import (
	"fmt"
	"io"
	"sync"

	"github.com/olivierh59500/ym-player/pkg/stsound"
)

const decodeBufferSize = 4096

type decoder interface {
	Compute(buffer []int16, samples int) bool
	Destroy()
	Restart()
}

// Player streams signed 16-bit little-endian stereo samples.
type Player struct {
	decoder decoder
	buffer  []int16
	mutex   sync.Mutex
	loop    bool
	volume  float64
}

// New creates a player from YM file data.
func New(data []byte, sampleRate int, loop bool) (*Player, error) {
	decoder := stsound.CreateWithRate(sampleRate)
	if err := decoder.LoadMemory(data); err != nil {
		decoder.Destroy()
		return nil, fmt.Errorf("load YM data: %w", err)
	}
	decoder.SetLoopMode(loop)
	return newPlayer(decoder, loop), nil
}

func newPlayer(decoder decoder, loop bool) *Player {
	return &Player{
		decoder: decoder,
		buffer:  make([]int16, decodeBufferSize),
		loop:    loop,
		volume:  0.5,
	}
}

// Read implements io.Reader without allocating on the audio callback path.
func (p *Player) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	if len(destination)%4 != 0 {
		return 0, io.ErrShortBuffer
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.decoder == nil {
		return 0, io.ErrClosedPipe
	}

	samplesNeeded := len(destination) / 4
	for processed := 0; processed < samplesNeeded; {
		chunkSize := min(samplesNeeded-processed, len(p.buffer))
		if !p.decoder.Compute(p.buffer[:chunkSize], chunkSize) {
			if !p.loop {
				clear(destination[processed*4:])
				return len(destination), io.EOF
			}
			p.decoder.Restart()
			if !p.decoder.Compute(p.buffer[:chunkSize], chunkSize) {
				clear(destination[processed*4:])
				return len(destination), io.ErrUnexpectedEOF
			}
		}

		for i, mono := range p.buffer[:chunkSize] {
			sample := int16(float64(mono) * p.volume)
			position := (processed + i) * 4
			low, high := byte(sample), byte(sample>>8)
			destination[position] = low
			destination[position+1] = high
			destination[position+2] = low
			destination[position+3] = high
		}
		processed += chunkSize
	}

	return len(destination), nil
}

// SetVolume sets and clamps the playback volume to [0, 1].
func (p *Player) SetVolume(volume float64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.volume = max(0, min(volume, 1))
}

// Volume returns the current playback volume.
func (p *Player) Volume() float64 {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.volume
}

// Close releases the YM decoder.
func (p *Player) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.decoder != nil {
		p.decoder.Destroy()
		p.decoder = nil
	}
	return nil
}
