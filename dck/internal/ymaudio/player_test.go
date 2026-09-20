package ymaudio

import (
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

type fakeDecoder struct {
	sample      int16
	failCalls   int
	restarts    int
	destroyed   bool
	computeRuns int
}

func (f *fakeDecoder) Compute(buffer []int16, samples int) bool {
	f.computeRuns++
	if f.failCalls > 0 {
		f.failCalls--
		clear(buffer[:samples])
		return false
	}
	for i := range buffer[:samples] {
		buffer[i] = f.sample
	}
	return true
}

func (f *fakeDecoder) Destroy() { f.destroyed = true }
func (f *fakeDecoder) Restart() { f.restarts++ }

func TestReadProducesStereoPCM(t *testing.T) {
	decoder := &fakeDecoder{sample: 1000}
	player := newPlayer(decoder, true)
	player.SetVolume(0.5)
	destination := make([]byte, (decodeBufferSize+1)*4)

	n, err := player.Read(destination)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if n != len(destination) {
		t.Fatalf("Read() n = %d, want %d", n, len(destination))
	}
	if decoder.computeRuns != 2 {
		t.Fatalf("Compute() calls = %d, want 2", decoder.computeRuns)
	}
	for position := 0; position < n; position += 4 {
		left := int16(binary.LittleEndian.Uint16(destination[position : position+2]))
		right := int16(binary.LittleEndian.Uint16(destination[position+2 : position+4]))
		if left != 500 || right != left {
			t.Fatalf("frame %d = (%d, %d), want (500, 500)", position/4, left, right)
		}
	}
}

func TestReadRestartsLoop(t *testing.T) {
	decoder := &fakeDecoder{sample: 42, failCalls: 1}
	player := newPlayer(decoder, true)
	destination := make([]byte, 4)

	if _, err := player.Read(destination); err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if decoder.restarts != 1 {
		t.Fatalf("Restart() calls = %d, want 1", decoder.restarts)
	}
}

func TestReadReturnsSilenceAtEnd(t *testing.T) {
	decoder := &fakeDecoder{failCalls: 1}
	player := newPlayer(decoder, false)
	destination := []byte{1, 2, 3, 4}

	n, err := player.Read(destination)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Read() error = %v, want io.EOF", err)
	}
	if n != len(destination) {
		t.Fatalf("Read() n = %d, want %d", n, len(destination))
	}
	for i, value := range destination {
		if value != 0 {
			t.Fatalf("destination[%d] = %d, want silence", i, value)
		}
	}
}

func TestReaderContractAndClose(t *testing.T) {
	decoder := &fakeDecoder{}
	player := newPlayer(decoder, true)
	if _, ok := any(player).(io.Seeker); ok {
		t.Fatal("Player unexpectedly implements io.Seeker")
	}
	if _, err := player.Read(make([]byte, 3)); !errors.Is(err, io.ErrShortBuffer) {
		t.Fatalf("unaligned Read() error = %v, want io.ErrShortBuffer", err)
	}

	player.SetVolume(-1)
	if got := player.Volume(); got != 0 {
		t.Fatalf("Volume() after -1 = %v, want 0", got)
	}
	player.SetVolume(2)
	if got := player.Volume(); got != 1 {
		t.Fatalf("Volume() after 2 = %v, want 1", got)
	}

	if err := player.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !decoder.destroyed {
		t.Fatal("Close() did not destroy the decoder")
	}
	if _, err := player.Read(make([]byte, 4)); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Read() after Close() error = %v, want io.ErrClosedPipe", err)
	}
}

func BenchmarkRead(b *testing.B) {
	player := newPlayer(&fakeDecoder{sample: 1000}, true)
	destination := make([]byte, 4096)
	b.ReportAllocs()
	b.SetBytes(int64(len(destination)))
	b.ResetTimer()
	for range b.N {
		if _, err := player.Read(destination); err != nil {
			b.Fatal(err)
		}
	}
}
