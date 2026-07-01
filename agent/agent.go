package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"zoneout/internal/audioanalysis"
	"zoneout/internal/playback"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const agentAddr = "127.0.0.1:17777"

type playRequest struct {
	StreamURL string `json:"stream_url"`
}

type statusResponse struct {
	State     string `json:"state"`
	StreamURL string `json:"stream_url,omitempty"`
	Error     string `json:"error,omitempty"`
}

type playbackState string

const (
	stateIdle       playbackState = "idle"
	stateConnecting playbackState = "connecting"
	statePlaying    playbackState = "playing"
	stateError      playbackState = "error"
)

type agentState struct {
	mu        sync.Mutex
	state     playbackState
	streamURL string
	err       string
	cancel    context.CancelFunc
}

var state = &agentState{state: "idle"}
const barCount = 8

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--ensure-running" {
		if err := ensureRunning(); err != nil {
			log.Fatal(err)
		}
		return
	}

	runServer()
}

func runServer() {
	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		_, _ = w.Write([]byte("ok"))
	})

	router.Get("/status", statusHandler)
	router.Post("/play", playMusicHandler)
	router.Post("/stop", stopMusicHandler)

	log.Printf("Starting the agent server at %s", agentAddr)
	if err := http.ListenAndServe(agentAddr, router); err != nil {
		log.Fatal(err)
	}
}

func playMusicHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody playRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.StreamURL == "" {
		http.Error(w, "stream_url is required", http.StatusBadRequest)
		return
	}

	analyzer := audioanalysis.NewAnalyzer(barCount)

	ctx, cancel := context.WithCancel(context.Background())

	state.mu.Lock()
	if state.cancel != nil {
		state.cancel()
	}

	state.state = stateConnecting
	state.streamURL = reqBody.StreamURL
	state.err = ""
	state.cancel = cancel
	state.mu.Unlock()

	go func() {
		state.mu.Lock()
		state.state = statePlaying
		state.mu.Unlock()

		err := playback.Play(ctx, reqBody.StreamURL)

		state.mu.Lock()
		defer state.mu.Unlock()

		if errors.Is(err, context.Canceled) {
			state.state = stateIdle
			state.err = ""
			return
		}

		if err != nil {
			state.state = stateError
			state.err = err.Error()
			return
		}

		state.state = stateIdle
		state.err = ""
	}()

	writeJson(w, http.StatusAccepted, currentStatus())
}

func stopMusicHandler(w http.ResponseWriter, r *http.Request) {
	state.mu.Lock()

	cancel := state.cancel
	state.cancel = nil
	state.state = stateIdle
	state.err = ""
	state.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	writeJson(w, http.StatusOK, currentStatus())
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	writeJson(w, http.StatusOK, currentStatus())
}

func writeJson(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func currentStatus() statusResponse {
	state.mu.Lock()
	defer state.mu.Unlock()

	return statusResponse{
		StreamURL: state.streamURL,
		Error:     state.err,
		State:     string(state.state),
	}
}

func ensureRunning() error {
	if healthOk() {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return err
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if healthOk() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("agent did not become healthy")
}

func healthOk() bool {
	client := http.Client{Timeout: 500 * time.Millisecond}

	res, err := client.Get("http://" + agentAddr + "/health")
	if err != nil {
		return false
	}

	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 32))
	return res.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "ok"
}
