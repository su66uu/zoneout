package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

const url = "https://coderadio-admin-v2.freecodecamp.org/listen/coderadio/radio.mp3"

func Run() error {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	start := time.Now()
	log.Printf("connecting to %s", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	log.Printf("connected in %s: %s content-type=%q", time.Since(start), res.Status, res.Header.Get("Content-Type"))

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
	player.Play()
	log.Println("playback started")

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for player.IsPlaying() {
		select {
		case <-ctx.Done():
			log.Println("stopping: context cancelled")
			return ctx.Err()
		case <-ticker.C:
		}
	}

	log.Println("Playback ended normally")
	return nil
}
