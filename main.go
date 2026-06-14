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

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unable to process the stream: %s", res.Status)
	}

	decoded, err := mp3.NewDecoder(res.Body)
	if err != nil {
		return err
	}

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

	player := otoCtx.NewPlayer(decoded)
	player.Play()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for player.IsPlaying() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}

	return nil
}
