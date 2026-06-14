package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const url = "https://coderadio-admin-v2.freecodecamp.org/listen/coderadio/radio.mp3"

func main() {
	client := &http.Client{}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		log.Fatalf("Unable to process the stream: %s", res.Status)
	}

	out, err := os.Create("capture.mp3")
	if err != nil {
		log.Fatal(err)
	}

	written, err := io.Copy(out, res.Body)
	if err != nil {
		log.Fatal(err)
	}

	if err := out.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Bytes written", written)
}
