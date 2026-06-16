package playback

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

func Play(ctx context.Context, streamURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		log.Println("closing the response body")
		_ = res.Body.Close()
	}()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unable to process the stream: %s", res.Status)
	}

	log.Println("starting MP3 decode")
	decoded, err := mp3.NewDecoder(res.Body)
	if err != nil {
		return err
	}
	log.Println("MP3 decode is ready")

	log.Println("creating audio player")
	op := &oto.NewContextOptions{
		SampleRate:   decoded.SampleRate(),
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	}
	otoCtx, ready, err := oto.NewContext(op)
	if err != nil {
		return err
	}
	<-ready
	log.Println("audio context is ready")

	player := otoCtx.NewPlayer(decoded)
	defer player.Pause()

	player.Play()
	log.Println("playback started")

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for player.IsPlaying() {
		select {
		case <-ctx.Done():
			log.Println("stopping: context cancelled")
			player.Pause()
			return ctx.Err()
		case <-ticker.C:
		}
	}

	log.Println("Playback ended normally")
	return nil
}
