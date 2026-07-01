package audioanalysis

import (
	"encoding/binary"
	"math"
	"sync"
)

type Analyzer struct {
	mu     sync.RWMutex
	levels []uint8
}

func NewAnalyzer(barCount int) *Analyzer {
	return &Analyzer{
		levels: make([]uint8, barCount),
	}
}

func (a *Analyzer) ObservePCM16StereoLE(p []byte) {
	const (
		frameSize = 4
		maxInt16  = 32768.0
		maxLevel  = 8
	)

	if len(a.levels) == 0 {
		return
	}

	frameCount := len(p) / frameSize
	if frameCount == 0 {
		return
	}

	var sumSquares float64

	for i := 0; i+frameSize <= len(p); i += frameSize {
		left := int16(binary.LittleEndian.Uint16(p[i : i+2]))
		right := int16(binary.LittleEndian.Uint16(p[i+2 : i+4]))

		mono := (float64(left) + float64(right)) / 2
		normalized := mono / maxInt16
		sumSquares += normalized * normalized
	}

	rms := math.Sqrt(sumSquares / float64(frameCount))
	level := min(uint8(math.Round(rms*maxLevel)), maxLevel)

	a.mu.Lock()
	defer a.mu.Unlock()

	copy(a.levels, a.levels[1:])
	a.levels[len(a.levels)-1] = level
}

func (a *Analyzer) Levels() []uint8 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make([]uint8, len(a.levels))
	copy(out, a.levels)
	return out
}
