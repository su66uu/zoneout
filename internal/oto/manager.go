package oto

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ebitengine/oto/v3"
)

type PlayerManager struct {
	otoCtx     *oto.Context
	mu         sync.Mutex
	sampleRate int
}

func (m *PlayerManager) EnsureContext(sampleRate int) (*PlayerManager, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.otoCtx != nil {
		if m.sampleRate != sampleRate {
			return nil, fmt.Errorf("sample rate changed: got %d, want %d", sampleRate, m.sampleRate)
		}
		return m, nil
	}

	otoCtx, ready, err := oto.NewContext(
		&oto.NewContextOptions{
			SampleRate:   sampleRate,
			ChannelCount: 2,
			Format:       oto.FormatSignedInt16LE,
		},
	)
	if err != nil {
		return m, err
	}

	<-ready
	m.otoCtx = otoCtx
	m.sampleRate = sampleRate

	return m, nil
}

func (m *PlayerManager) NewPlayer(r io.Reader) (*oto.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.otoCtx == nil {
		return nil, errors.New("oto context is not initialized")
	}
	return m.otoCtx.NewPlayer(r), nil
}
